package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sdknextdns "github.com/jacaudi/nextdns-go/nextdns"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
	"github.com/jacaudi/nextdns-operator/internal/metrics"
	"github.com/jacaudi/nextdns-operator/internal/nextdns"
)

const (
	// FinalizerName is the finalizer used by this controller
	FinalizerName = "nextdns.io/profile-finalizer"

	// ConditionTypeReady indicates the profile is ready
	ConditionTypeReady = "Ready"

	// ConditionTypeSynced indicates the profile is synced with NextDNS
	ConditionTypeSynced = "Synced"

	// ConditionTypeReferencesResolved indicates all references are resolved
	ConditionTypeReferencesResolved = "ReferencesResolved"

	// ConditionTypeObserveOnly indicates the profile is in observe-only mode
	ConditionTypeObserveOnly = "ObserveOnly"
)

const (
	// credentialsRefIndexField is the field index key for looking up profiles by their secret reference
	credentialsRefIndexField = ".spec.credentialsRef"
)

// credentialsRefIndexFunc extracts the secret reference key (namespace/name) from a NextDNSProfile
// for use with controller-runtime's field indexer. This enables efficient lookups when a Secret changes.
func credentialsRefIndexFunc(obj client.Object) []string {
	profile, ok := obj.(*nextdnsv1alpha1.NextDNSProfile)
	if !ok {
		return nil
	}
	ns := profile.Spec.CredentialsRef.Namespace
	if ns == "" {
		ns = profile.Namespace
	}
	return []string{ns + "/" + profile.Spec.CredentialsRef.Name}
}

// ClientFactory is a function that creates a NextDNS client
type ClientFactory func(apiKey string) (nextdns.ClientInterface, error)

// DefaultClientFactory creates a real NextDNS client
func DefaultClientFactory(apiKey string) (nextdns.ClientInterface, error) {
	return nextdns.NewClient(apiKey)
}

// NextDNSProfileReconciler reconciles a NextDNSProfile object
type NextDNSProfileReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	ClientFactory     ClientFactory
	SyncPeriod        time.Duration
	lastMetricsUpdate time.Time
}

// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles/finalizers,verbs=update
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsallowlists,verbs=get;list;watch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsdenylists,verbs=get;list;watch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnstldlists,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *NextDNSProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Update resource count metrics (throttled to once per sync period)
	if time.Since(r.lastMetricsUpdate) > r.SyncPeriod {
		r.updateResourceMetrics(ctx)
		r.lastMetricsUpdate = time.Now()
	}

	// Fetch the NextDNSProfile instance
	profile := &nextdnsv1alpha1.NextDNSProfile{}
	if err := r.Get(ctx, req.NamespacedName, profile); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			logger.Info("NextDNSProfile resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get NextDNSProfile")
		return ctrl.Result{}, err
	}

	// Deep copy to avoid mutating the controller-runtime cache
	profile = profile.DeepCopy()

	// Migrate old finalizer name if present
	if migrated, err := migrateFinalizerDomain(ctx, r.Client, profile, "nextdns.io/finalizer", FinalizerName); err != nil {
		return ctrl.Result{}, err
	} else if migrated {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Check if the resource is being deleted
	if !profile.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, profile)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(profile, FinalizerName) {
		logger.Info("Adding finalizer to NextDNSProfile")
		controllerutil.AddFinalizer(profile, FinalizerName)
		if err := r.Update(ctx, profile); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Get API credentials
	apiKey, err := r.getAPIKey(ctx, profile)
	if err != nil {
		logger.Error(err, "Failed to get API credentials")
		metrics.RecordProfileSyncError(profile.Name, profile.Namespace, "CredentialsNotFound")
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "CredentialsNotFound", err.Error())
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Determine mode (default: managed)
	mode := profile.Spec.Mode
	if mode == "" {
		mode = nextdnsv1alpha1.ProfileModeManaged
	}

	// Observe mode: read-only reconciliation
	if mode == nextdnsv1alpha1.ProfileModeObserve {
		return r.reconcileObserveMode(ctx, profile, apiKey)
	}

	// Managed mode: validate name is set
	if profile.Spec.Name == "" {
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "NameRequired",
			"spec.name is required in managed mode")
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Transition guard: block if switching from observe to managed with empty spec
	if profile.Status.ObservedConfig != nil && !specHasConfig(&profile.Spec) {
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "TransitionBlocked",
			"Cannot switch to managed mode with empty spec. Copy desired config from status.observedConfig into spec first.")
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Clear observedConfig on first successful managed reconciliation
	if profile.Status.ObservedConfig != nil {
		profile.Status.ObservedConfig = nil
		profile.Status.SuggestedSpec = nil
		r.setCondition(profile, ConditionTypeObserveOnly, metav1.ConditionFalse, "ManagedMode",
			"Profile transitioned to managed mode")
	}

	// Resolve list references
	resolvedLists, err := r.resolveListReferences(ctx, profile)
	if err != nil {
		logger.Error(err, "Failed to resolve list references")
		metrics.RecordProfileSyncError(profile.Name, profile.Namespace, "ReferencesNotResolved")
		r.setCondition(profile, ConditionTypeReferencesResolved, metav1.ConditionFalse, "ResolutionFailed", err.Error())
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "ReferencesNotResolved", "Failed to resolve list references")
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Mark references as resolved
	r.setCondition(profile, ConditionTypeReferencesResolved, metav1.ConditionTrue, "AllResolved", "All referenced lists found and valid")

	// Sync with NextDNS API
	if err := r.syncWithNextDNS(ctx, profile, apiKey, resolvedLists); err != nil {
		logger.Error(err, "Failed to sync with NextDNS")
		metrics.RecordProfileSyncError(profile.Name, profile.Namespace, "SyncFailed")
		r.setCondition(profile, ConditionTypeSynced, metav1.ConditionFalse, "SyncFailed", err.Error())
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "SyncFailed", "Failed to sync with NextDNS API")
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	// Record successful sync
	metrics.RecordProfileSync(profile.Name, profile.Namespace)

	// Update status
	now := metav1.Now()
	profile.Status.LastSyncTime = &now
	profile.Status.ObservedGeneration = profile.Generation
	profile.Status.AggregatedCounts = &nextdnsv1alpha1.AggregatedCounts{
		AllowlistDomains: len(resolvedLists.Allowlist),
		DenylistDomains:  len(resolvedLists.Denylist),
		BlockedTLDs:      len(resolvedLists.TLDs),
	}
	profile.Status.ReferencedResources = resolvedLists.ResourceStatus

	r.setCondition(profile, ConditionTypeSynced, metav1.ConditionTrue, "Success", "All settings applied")
	r.setCondition(profile, ConditionTypeReady, metav1.ConditionTrue, "Synced", "Profile successfully synced with NextDNS")

	// Reconcile ConfigMap if enabled
	if err := r.reconcileConfigMap(ctx, profile); err != nil {
		logger.Error(err, "Failed to reconcile ConfigMap")
		// Don't fail the reconciliation for ConfigMap errors, just log
	}

	if err := r.Status().Update(ctx, profile); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled NextDNSProfile",
		"profileID", profile.Status.ProfileID,
		"allowlistCount", len(resolvedLists.Allowlist),
		"denylistCount", len(resolvedLists.Denylist),
		"tldCount", len(resolvedLists.TLDs))

	// Schedule next sync with jitter for drift detection
	syncInterval := CalculateSyncInterval(r.SyncPeriod)
	if syncInterval > 0 {
		logger.V(1).Info("Scheduling next drift detection sync", "interval", syncInterval)
	}

	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// handleDeletion handles the deletion of a NextDNSProfile
func (r *NextDNSProfileReconciler) handleDeletion(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(profile, FinalizerName) {
		logger.Info("Handling deletion of NextDNSProfile")

		// Only delete the profile from NextDNS if we created it (no profileID was specified in spec)
		// and we have a profile ID in status
		if profile.Spec.Mode == nextdnsv1alpha1.ProfileModeObserve {
			logger.Info("Skipping NextDNS profile deletion (observe mode, profile not owned)", "profileID", profile.Status.ProfileID)
		} else if profile.Spec.ProfileID == "" && profile.Status.ProfileID != "" {
			// Get API credentials
			apiKey, err := r.getAPIKey(ctx, profile)
			if err != nil {
				logger.Error(err, "Failed to get API credentials for deletion, proceeding with finalizer removal")
			} else {
				// Create NextDNS client using factory
				factory := r.ClientFactory
				if factory == nil {
					factory = DefaultClientFactory
				}
				client, err := factory(apiKey)
				if err != nil {
					logger.Error(err, "Failed to create NextDNS client for deletion")
				} else {
					if err := client.DeleteProfile(ctx, profile.Status.ProfileID); err != nil {
						logger.Error(err, "Failed to delete profile from NextDNS", "profileID", profile.Status.ProfileID)
						// Continue with finalizer removal even if deletion fails
					} else {
						logger.Info("Deleted NextDNS profile", "profileID", profile.Status.ProfileID)
					}
				}
			}
		} else if profile.Spec.ProfileID != "" {
			logger.Info("Skipping NextDNS profile deletion (profile was adopted, not created)", "profileID", profile.Status.ProfileID)
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(profile, FinalizerName)
		if err := r.Update(ctx, profile); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// getAPIKey retrieves the NextDNS API key from the referenced Secret
func (r *NextDNSProfileReconciler) getAPIKey(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile) (string, error) {
	secretName := profile.Spec.CredentialsRef.Name
	secretKey := profile.Spec.CredentialsRef.Key
	if secretKey == "" {
		secretKey = "api-key"
	}

	// Use credentialsRef.namespace if set, otherwise default to the profile's namespace
	secretNamespace := profile.Spec.CredentialsRef.Namespace
	if secretNamespace == "" {
		secretNamespace = profile.Namespace
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: secretNamespace,
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", secretNamespace, secretName, err)
	}

	apiKey, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s/%s", secretKey, secretNamespace, secretName)
	}

	return string(apiKey), nil
}

// ResolvedLists contains the merged lists from all sources
type ResolvedLists struct {
	Allowlist      []nextdns.DomainEntry
	Denylist       []nextdns.DomainEntry
	TLDs           []string // TLDs stay as strings - NextDNS API doesn't support active field for TLDs
	ResourceStatus *nextdnsv1alpha1.ReferencedResources
}

// resolveListReferences resolves all list references and merges with inline lists
func (r *NextDNSProfileReconciler) resolveListReferences(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile) (*ResolvedLists, error) {
	resolved := &ResolvedLists{
		Allowlist: make([]nextdns.DomainEntry, 0),
		Denylist:  make([]nextdns.DomainEntry, 0),
		TLDs:      make([]string, 0),
		ResourceStatus: &nextdnsv1alpha1.ReferencedResources{
			Allowlists: make([]nextdnsv1alpha1.ReferencedResourceStatus, 0),
			Denylists:  make([]nextdnsv1alpha1.ReferencedResourceStatus, 0),
			TLDLists:   make([]nextdnsv1alpha1.ReferencedResourceStatus, 0),
		},
	}

	// Resolve allowlist references
	for _, ref := range profile.Spec.AllowlistRefs {
		ns := ref.Namespace
		if ns == "" {
			ns = profile.Namespace
		}

		allowlist := &nextdnsv1alpha1.NextDNSAllowlist{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, allowlist); err != nil {
			return nil, fmt.Errorf("failed to get allowlist %s/%s: %w", ns, ref.Name, err)
		}

		count := 0
		for _, entry := range allowlist.Spec.Domains {
			active := entry.Active == nil || *entry.Active
			resolved.Allowlist = append(resolved.Allowlist, nextdns.DomainEntry{
				Domain: entry.Domain,
				Active: active,
			})
			if active {
				count++
			}
		}

		resolved.ResourceStatus.Allowlists = append(resolved.ResourceStatus.Allowlists, nextdnsv1alpha1.ReferencedResourceStatus{
			Name:      ref.Name,
			Namespace: ns,
			Ready:     true,
			Count:     count,
		})
	}

	// Add inline allowlist entries
	for _, entry := range profile.Spec.Allowlist {
		active := entry.Active == nil || *entry.Active
		resolved.Allowlist = append(resolved.Allowlist, nextdns.DomainEntry{
			Domain: entry.Domain,
			Active: active,
		})
	}

	// Resolve denylist references
	for _, ref := range profile.Spec.DenylistRefs {
		ns := ref.Namespace
		if ns == "" {
			ns = profile.Namespace
		}

		denylist := &nextdnsv1alpha1.NextDNSDenylist{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, denylist); err != nil {
			return nil, fmt.Errorf("failed to get denylist %s/%s: %w", ns, ref.Name, err)
		}

		count := 0
		for _, entry := range denylist.Spec.Domains {
			active := entry.Active == nil || *entry.Active
			resolved.Denylist = append(resolved.Denylist, nextdns.DomainEntry{
				Domain: entry.Domain,
				Active: active,
			})
			if active {
				count++
			}
		}

		resolved.ResourceStatus.Denylists = append(resolved.ResourceStatus.Denylists, nextdnsv1alpha1.ReferencedResourceStatus{
			Name:      ref.Name,
			Namespace: ns,
			Ready:     true,
			Count:     count,
		})
	}

	// Add inline denylist entries
	for _, entry := range profile.Spec.Denylist {
		active := entry.Active == nil || *entry.Active
		resolved.Denylist = append(resolved.Denylist, nextdns.DomainEntry{
			Domain: entry.Domain,
			Active: active,
		})
	}

	// Resolve TLD list references
	for _, ref := range profile.Spec.TLDListRefs {
		ns := ref.Namespace
		if ns == "" {
			ns = profile.Namespace
		}

		tldList := &nextdnsv1alpha1.NextDNSTLDList{}
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, tldList); err != nil {
			return nil, fmt.Errorf("failed to get TLD list %s/%s: %w", ns, ref.Name, err)
		}

		count := 0
		for _, entry := range tldList.Spec.TLDs {
			if entry.Active == nil || *entry.Active {
				resolved.TLDs = append(resolved.TLDs, entry.TLD)
				count++
			}
		}

		resolved.ResourceStatus.TLDLists = append(resolved.ResourceStatus.TLDLists, nextdnsv1alpha1.ReferencedResourceStatus{
			Name:      ref.Name,
			Namespace: ns,
			Ready:     true,
			Count:     count,
		})
	}

	return resolved, nil
}

// syncWithNextDNS syncs the profile with the NextDNS API
func (r *NextDNSProfileReconciler) syncWithNextDNS(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile, apiKey string, lists *ResolvedLists) error {
	logger := log.FromContext(ctx)

	// Create NextDNS client using factory
	factory := r.ClientFactory
	if factory == nil {
		factory = DefaultClientFactory
	}
	client, err := factory(apiKey)
	if err != nil {
		return fmt.Errorf("failed to create NextDNS client: %w", err)
	}

	logger.Info("Syncing with NextDNS API",
		"profileName", profile.Spec.Name,
		"profileID", profile.Spec.ProfileID)

	// If no profile ID is set, create a new profile or adopt existing one
	if profile.Status.ProfileID == "" {
		var existingProfile, newProfile *sdknextdns.Profile
		if profile.Spec.ProfileID != "" {
			// Adopt existing profile - verify it exists
			existingProfile, err = client.GetProfile(ctx, profile.Spec.ProfileID)
			if err != nil {
				return fmt.Errorf("failed to get existing profile %s: %w", profile.Spec.ProfileID, err)
			}
			profile.Status.ProfileID = profile.Spec.ProfileID
		} else {
			// Create new profile via API
			newProfileID, err := client.CreateProfile(ctx, profile.Spec.Name)
			if err != nil {
				return fmt.Errorf("failed to create profile: %w", err)
			}
			profile.Status.ProfileID = newProfileID
			logger.Info("Created new NextDNS profile", "profileID", newProfileID)
			newProfile, err = client.GetProfile(ctx, newProfileID)
			if err != nil {
				logger.Error(err, "Failed to get fingerprint for new profile", "profileID", newProfileID)
			}
		}
		// Set fingerprint from API response
		switch {
		case existingProfile != nil:
			profile.Status.Fingerprint = existingProfile.Fingerprint
		case newProfile != nil:
			profile.Status.Fingerprint = newProfile.Fingerprint
		default:
			logger.Info("WARNING: could not retrieve fingerprint from API, fingerprint will be empty")
		}
	}

	profileID := profile.Status.ProfileID

	// Update profile name if needed
	if err := client.UpdateProfile(ctx, profileID, profile.Spec.Name); err != nil {
		return fmt.Errorf("failed to update profile name: %w", err)
	}

	// Sync security settings
	if profile.Spec.Security != nil {
		securityConfig := &nextdns.SecurityConfig{
			ThreatIntelligenceFeeds: boolValue(profile.Spec.Security.ThreatIntelligenceFeeds, true),
			AIThreatDetection:       boolValue(profile.Spec.Security.AIThreatDetection, true),
			GoogleSafeBrowsing:      boolValue(profile.Spec.Security.GoogleSafeBrowsing, true),
			Cryptojacking:           boolValue(profile.Spec.Security.Cryptojacking, true),
			DNSRebinding:            boolValue(profile.Spec.Security.DNSRebinding, true),
			IDNHomographs:           boolValue(profile.Spec.Security.IDNHomographs, true),
			Typosquatting:           boolValue(profile.Spec.Security.Typosquatting, true),
			DGA:                     boolValue(profile.Spec.Security.DGA, true),
			NRD:                     boolValue(profile.Spec.Security.NRD, false),
			DDNS:                    boolValue(profile.Spec.Security.DDNS, false),
			Parking:                 boolValue(profile.Spec.Security.Parking, true),
			CSAM:                    boolValue(profile.Spec.Security.CSAM, true),
		}
		if err := client.UpdateSecurity(ctx, profileID, securityConfig); err != nil {
			return fmt.Errorf("failed to update security settings: %w", err)
		}
	}

	// Sync privacy settings
	if profile.Spec.Privacy != nil {
		privacyConfig := &nextdns.PrivacyConfig{
			DisguisedTrackers: boolValue(profile.Spec.Privacy.DisguisedTrackers, true),
			AllowAffiliate:    boolValue(profile.Spec.Privacy.AllowAffiliate, false),
		}
		if err := client.UpdatePrivacy(ctx, profileID, privacyConfig); err != nil {
			return fmt.Errorf("failed to update privacy settings: %w", err)
		}

		// Sync blocklists
		if len(profile.Spec.Privacy.Blocklists) > 0 {
			blocklists := make([]string, 0, len(profile.Spec.Privacy.Blocklists))
			for _, bl := range profile.Spec.Privacy.Blocklists {
				if bl.Active == nil || *bl.Active {
					blocklists = append(blocklists, bl.ID)
				}
			}
			if err := client.SyncPrivacyBlocklists(ctx, profileID, blocklists); err != nil {
				return fmt.Errorf("failed to sync privacy blocklists: %w", err)
			}
		}

		// Sync native tracking protection
		if len(profile.Spec.Privacy.Natives) > 0 {
			natives := make([]string, 0, len(profile.Spec.Privacy.Natives))
			for _, n := range profile.Spec.Privacy.Natives {
				if n.Active == nil || *n.Active {
					natives = append(natives, n.ID)
				}
			}
			if err := client.SyncPrivacyNatives(ctx, profileID, natives); err != nil {
				return fmt.Errorf("failed to sync privacy natives: %w", err)
			}
		}
	}

	// Sync parental control settings
	if profile.Spec.ParentalControl != nil {
		categories := make([]string, 0)
		for _, c := range profile.Spec.ParentalControl.Categories {
			if c.Active == nil || *c.Active {
				categories = append(categories, c.ID)
			}
		}
		services := make([]string, 0)
		for _, s := range profile.Spec.ParentalControl.Services {
			if s.Active == nil || *s.Active {
				services = append(services, s.ID)
			}
		}

		pcConfig := &nextdns.ParentalControlConfig{
			Categories:            categories,
			Services:              services,
			SafeSearch:            boolValue(profile.Spec.ParentalControl.SafeSearch, false),
			YouTubeRestrictedMode: boolValue(profile.Spec.ParentalControl.YouTubeRestrictedMode, false),
			BlockBypass:           boolValue(profile.Spec.ParentalControl.BlockBypass, false),
		}
		if err := client.UpdateParentalControl(ctx, profileID, pcConfig); err != nil {
			return fmt.Errorf("failed to update parental control settings: %w", err)
		}
	}

	// Sync settings (logs, block page, performance, web3)
	if profile.Spec.Settings != nil {
		settingsConfig := &nextdns.SettingsConfig{
			// Log defaults
			LogsEnabled:   true,
			LogClientsIPs: false,
			LogDomains:    true,
			// Block page default
			BlockPageEnable: true,
			// Performance defaults
			Ecs:             true,
			CacheBoost:      true,
			CnameFlattening: true,
			// Web3 default
			Web3: false,
		}
		if profile.Spec.Settings.Logs != nil {
			settingsConfig.LogsEnabled = boolValue(profile.Spec.Settings.Logs.Enabled, true)
			settingsConfig.LogClientsIPs = boolValue(profile.Spec.Settings.Logs.LogClientsIPs, false)
			settingsConfig.LogDomains = boolValue(profile.Spec.Settings.Logs.LogDomains, true)
			settingsConfig.LogRetention = parseRetentionDays(profile.Spec.Settings.Logs.Retention)
			settingsConfig.Location = profile.Spec.Settings.Logs.Location
		}
		if profile.Spec.Settings.BlockPage != nil {
			settingsConfig.BlockPageEnable = boolValue(profile.Spec.Settings.BlockPage.Enabled, true)
		}
		if profile.Spec.Settings.Performance != nil {
			settingsConfig.Ecs = boolValue(profile.Spec.Settings.Performance.ECS, true)
			settingsConfig.CacheBoost = boolValue(profile.Spec.Settings.Performance.CacheBoost, true)
			settingsConfig.CnameFlattening = boolValue(profile.Spec.Settings.Performance.CNAMEFlattening, true)
		}
		settingsConfig.Web3 = boolValue(profile.Spec.Settings.Web3, false)
		if err := client.UpdateSettings(ctx, profileID, settingsConfig); err != nil {
			return fmt.Errorf("failed to update settings: %w", err)
		}
	}

	// Sync rewrites (nil = field omitted, don't touch remote; empty = explicit clear)
	if profile.Spec.Rewrites != nil {
		rewriteEntries := make([]nextdns.RewriteEntry, 0, len(profile.Spec.Rewrites))
		for _, rw := range profile.Spec.Rewrites {
			if rw.Active == nil || *rw.Active {
				rewriteEntries = append(rewriteEntries, nextdns.RewriteEntry{
					Name:    rw.From,
					Content: rw.To,
				})
			}
		}
		if err := client.SyncRewrites(ctx, profileID, rewriteEntries); err != nil {
			return fmt.Errorf("failed to sync rewrites: %w", err)
		}
	}

	// Sync denylist
	if len(lists.Denylist) > 0 {
		if err := client.SyncDenylist(ctx, profileID, lists.Denylist); err != nil {
			return fmt.Errorf("failed to sync denylist: %w", err)
		}
	}

	// Sync allowlist
	if len(lists.Allowlist) > 0 {
		if err := client.SyncAllowlist(ctx, profileID, lists.Allowlist); err != nil {
			return fmt.Errorf("failed to sync allowlist: %w", err)
		}
	}

	// Sync TLDs
	if len(lists.TLDs) > 0 {
		if err := client.SyncSecurityTLDs(ctx, profileID, lists.TLDs); err != nil {
			return fmt.Errorf("failed to sync TLDs: %w", err)
		}
	}

	logger.Info("Successfully synced with NextDNS API", "profileID", profileID)
	return nil
}

// reconcileObserveMode handles reconciliation when mode is "observe"
func (r *NextDNSProfileReconciler) reconcileObserveMode(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile, apiKey string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Validate profileID is set
	if profile.Spec.ProfileID == "" {
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "ProfileIDRequired",
			"spec.profileID is required in observe mode")
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Create NextDNS client
	factory := r.ClientFactory
	if factory == nil {
		factory = DefaultClientFactory
	}
	client, err := factory(apiKey)
	if err != nil {
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "ObserveFailed",
			fmt.Sprintf("Failed to create API client: %v", err))
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	// Read full profile from NextDNS
	observed, fingerprint, err := r.readFullProfile(ctx, client, profile.Spec.ProfileID)
	if err != nil {
		logger.Error(err, "Failed to read full profile from NextDNS")
		metrics.RecordProfileSyncError(profile.Name, profile.Namespace, "ObserveFailed")
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "ObserveFailed", err.Error())
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	// Update status
	profile.Status.ProfileID = profile.Spec.ProfileID
	profile.Status.Fingerprint = fingerprint
	profile.Status.ObservedConfig = observed
	profile.Status.SuggestedSpec = buildSuggestedSpec(observed)
	now := metav1.Now()
	profile.Status.LastSyncTime = &now
	profile.Status.ObservedGeneration = profile.Generation

	r.setCondition(profile, ConditionTypeObserveOnly, metav1.ConditionTrue, "ObserveMode", "Profile is in observe-only mode")
	r.setCondition(profile, ConditionTypeSynced, metav1.ConditionTrue, "ObserveSuccess", "Remote profile read successfully")
	r.setCondition(profile, ConditionTypeReady, metav1.ConditionTrue, "Observed", "Profile observed successfully")

	metrics.RecordProfileSync(profile.Name, profile.Namespace)

	if err := r.Status().Update(ctx, profile); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully observed NextDNS profile",
		"profileID", profile.Spec.ProfileID,
		"profileName", observed.Name)

	syncInterval := CalculateSyncInterval(r.SyncPeriod)
	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// readFullProfile reads all sections of a NextDNS profile
func (r *NextDNSProfileReconciler) readFullProfile(ctx context.Context, client nextdns.ClientInterface, profileID string) (*nextdnsv1alpha1.ObservedConfig, string, error) {
	observed := &nextdnsv1alpha1.ObservedConfig{}

	// Get profile name and fingerprint
	profile, err := client.GetProfile(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get profile: %w", err)
	}
	apiFingerprint := profile.Fingerprint
	observed.Name = profile.Name

	// Get security settings
	security, err := client.GetSecurity(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get security: %w", err)
	}
	observed.Security = &nextdnsv1alpha1.ObservedSecurity{
		AIThreatDetection:       security.AiThreatDetection,
		ThreatIntelligenceFeeds: security.ThreatIntelligenceFeeds,
		GoogleSafeBrowsing:      security.GoogleSafeBrowsing,
		Cryptojacking:           security.Cryptojacking,
		DNSRebinding:            security.DNSRebinding,
		IDNHomographs:           security.IdnHomographs,
		Typosquatting:           security.Typosquatting,
		DGA:                     security.Dga,
		NRD:                     security.Nrd,
		DDNS:                    security.DDNS,
		Parking:                 security.Parking,
		CSAM:                    security.Csam,
	}

	// Get privacy settings
	privacy, err := client.GetPrivacy(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get privacy: %w", err)
	}
	observed.Privacy = &nextdnsv1alpha1.ObservedPrivacy{
		DisguisedTrackers: privacy.DisguisedTrackers,
		AllowAffiliate:    privacy.AllowAffiliate,
	}

	// Get privacy blocklists
	blocklists, err := client.GetPrivacyBlocklists(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get privacy blocklists: %w", err)
	}
	for _, bl := range blocklists {
		observed.Privacy.Blocklists = append(observed.Privacy.Blocklists, nextdnsv1alpha1.ObservedBlocklistEntry{ID: bl.ID})
	}

	// Get privacy natives
	natives, err := client.GetPrivacyNatives(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get privacy natives: %w", err)
	}
	for _, n := range natives {
		observed.Privacy.Natives = append(observed.Privacy.Natives, nextdnsv1alpha1.ObservedNativeEntry{ID: n.ID})
	}

	// Get parental control settings
	pc, err := client.GetParentalControl(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get parental control: %w", err)
	}
	observed.ParentalControl = &nextdnsv1alpha1.ObservedParentalControl{
		SafeSearch:            pc.SafeSearch,
		YouTubeRestrictedMode: pc.YoutubeRestrictedMode,
		BlockBypass:           pc.BlockBypass,
	}

	// Map recreation schedule if present
	if pc.Recreation != nil {
		observed.ParentalControl.Recreation = &nextdnsv1alpha1.ObservedRecreation{
			Timezone: pc.Recreation.Timezone,
		}
		if pc.Recreation.Times != nil {
			observed.ParentalControl.Recreation.Times = &nextdnsv1alpha1.ObservedRecreationTimes{}
			t := pc.Recreation.Times
			rt := observed.ParentalControl.Recreation.Times
			if t.Monday != nil {
				rt.Monday = &nextdnsv1alpha1.ObservedRecreationInterval{Start: t.Monday.Start, End: t.Monday.End}
			}
			if t.Tuesday != nil {
				rt.Tuesday = &nextdnsv1alpha1.ObservedRecreationInterval{Start: t.Tuesday.Start, End: t.Tuesday.End}
			}
			if t.Wednesday != nil {
				rt.Wednesday = &nextdnsv1alpha1.ObservedRecreationInterval{Start: t.Wednesday.Start, End: t.Wednesday.End}
			}
			if t.Thursday != nil {
				rt.Thursday = &nextdnsv1alpha1.ObservedRecreationInterval{Start: t.Thursday.Start, End: t.Thursday.End}
			}
			if t.Friday != nil {
				rt.Friday = &nextdnsv1alpha1.ObservedRecreationInterval{Start: t.Friday.Start, End: t.Friday.End}
			}
			if t.Saturday != nil {
				rt.Saturday = &nextdnsv1alpha1.ObservedRecreationInterval{Start: t.Saturday.Start, End: t.Saturday.End}
			}
			if t.Sunday != nil {
				rt.Sunday = &nextdnsv1alpha1.ObservedRecreationInterval{Start: t.Sunday.Start, End: t.Sunday.End}
			}
		}
	}

	// Get parental control categories
	categories, err := client.GetParentalControlCategories(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get parental control categories: %w", err)
	}
	for _, cat := range categories {
		observed.ParentalControl.Categories = append(observed.ParentalControl.Categories, nextdnsv1alpha1.ObservedCategoryEntry{
			ID:         cat.ID,
			Active:     cat.Active,
			Recreation: cat.Recreation,
		})
	}

	// Get parental control services
	services, err := client.GetParentalControlServices(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get parental control services: %w", err)
	}
	for _, svc := range services {
		observed.ParentalControl.Services = append(observed.ParentalControl.Services, nextdnsv1alpha1.ObservedServiceEntry{
			ID:     svc.ID,
			Active: svc.Active,
		})
	}

	// Get denylist
	denylist, err := client.GetDenylist(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get denylist: %w", err)
	}
	for _, d := range denylist {
		observed.Denylist = append(observed.Denylist, nextdnsv1alpha1.ObservedDomainEntry{
			Domain: d.ID,
			Active: d.Active,
		})
	}

	// Get allowlist
	allowlist, err := client.GetAllowlist(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get allowlist: %w", err)
	}
	for _, a := range allowlist {
		observed.Allowlist = append(observed.Allowlist, nextdnsv1alpha1.ObservedDomainEntry{
			Domain: a.ID,
			Active: a.Active,
		})
	}

	// Get blocked TLDs
	tlds, err := client.GetSecurityTLDs(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get security TLDs: %w", err)
	}
	for _, tld := range tlds {
		observed.BlockedTLDs = append(observed.BlockedTLDs, tld.ID)
	}

	// Get settings
	settings, err := client.GetSettings(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get settings: %w", err)
	}
	observed.Settings = &nextdnsv1alpha1.ObservedSettings{
		Web3: settings.Web3,
	}
	if settings.Logs != nil {
		observed.Settings.Logs = &nextdnsv1alpha1.ObservedLogs{
			Enabled:   settings.Logs.Enabled,
			Retention: settings.Logs.Retention,
			Location:  settings.Logs.Location,
		}
		// Invert Drop fields to user-friendly positive semantics:
		// API Drop.IP=true means "don't log IPs" -> LogClientsIPs=false
		if settings.Logs.Drop != nil {
			observed.Settings.Logs.LogClientsIPs = !settings.Logs.Drop.IP
			observed.Settings.Logs.LogDomains = !settings.Logs.Drop.Domain
		} else {
			// Default: log both when Drop is not set
			observed.Settings.Logs.LogClientsIPs = true
			observed.Settings.Logs.LogDomains = true
		}
	}
	if settings.BlockPage != nil {
		observed.Settings.BlockPage = &nextdnsv1alpha1.ObservedBlockPage{
			Enabled: settings.BlockPage.Enabled,
		}
	}
	if settings.Performance != nil {
		observed.Settings.Performance = &nextdnsv1alpha1.ObservedPerformance{
			ECS:             settings.Performance.Ecs,
			CacheBoost:      settings.Performance.CacheBoost,
			CNAMEFlattening: settings.Performance.CnameFlattening,
		}
	}

	// Get rewrites
	rewrites, err := client.GetRewrites(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get rewrites: %w", err)
	}
	for _, rw := range rewrites {
		observed.Rewrites = append(observed.Rewrites, nextdnsv1alpha1.ObservedRewriteEntry{
			Name:    rw.Name,
			Content: rw.Content,
		})
	}

	// Get setup (read-only endpoint data)
	setup, err := client.GetSetup(ctx, profileID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get setup: %w", err)
	}
	observed.Setup = &nextdnsv1alpha1.ObservedSetup{
		IPv4:     setup.Ipv4,
		IPv6:     setup.Ipv6,
		DNSCrypt: setup.Dnscrypt,
	}
	if setup.LinkedIP != nil {
		observed.Setup.LinkedIP = &nextdnsv1alpha1.ObservedLinkedIP{
			Servers: setup.LinkedIP.Servers,
			IP:      setup.LinkedIP.IP,
			DDNS:    setup.LinkedIP.Ddns,
			// updateToken intentionally excluded (sensitive)
		}
	}

	return observed, apiFingerprint, nil
}

// buildSuggestedSpec translates an ObservedConfig into spec-compatible types
// that users can copy directly into their NextDNSProfile spec.
// Fields not available from the API are omitted.
func buildSuggestedSpec(observed *nextdnsv1alpha1.ObservedConfig) *nextdnsv1alpha1.SuggestedSpec {
	if observed == nil {
		return nil
	}

	suggested := &nextdnsv1alpha1.SuggestedSpec{
		Name:        observed.Name,
		BlockedTLDs: observed.BlockedTLDs,
	}

	// Security: bool -> *bool
	if observed.Security != nil {
		suggested.Security = &nextdnsv1alpha1.SecuritySpec{
			ThreatIntelligenceFeeds: boolPtr(observed.Security.ThreatIntelligenceFeeds),
			AIThreatDetection:       boolPtr(observed.Security.AIThreatDetection),
			GoogleSafeBrowsing:      boolPtr(observed.Security.GoogleSafeBrowsing),
			Cryptojacking:           boolPtr(observed.Security.Cryptojacking),
			DNSRebinding:            boolPtr(observed.Security.DNSRebinding),
			IDNHomographs:           boolPtr(observed.Security.IDNHomographs),
			Typosquatting:           boolPtr(observed.Security.Typosquatting),
			DGA:                     boolPtr(observed.Security.DGA),
			NRD:                     boolPtr(observed.Security.NRD),
			DDNS:                    boolPtr(observed.Security.DDNS),
			Parking:                 boolPtr(observed.Security.Parking),
			CSAM:                    boolPtr(observed.Security.CSAM),
		}
	}

	// Privacy: bool -> *bool, blocklists/natives default Active to true
	if observed.Privacy != nil {
		suggested.Privacy = &nextdnsv1alpha1.PrivacySpec{
			DisguisedTrackers: boolPtr(observed.Privacy.DisguisedTrackers),
			AllowAffiliate:    boolPtr(observed.Privacy.AllowAffiliate),
		}
		for _, bl := range observed.Privacy.Blocklists {
			suggested.Privacy.Blocklists = append(suggested.Privacy.Blocklists, nextdnsv1alpha1.BlocklistEntry{
				ID:     bl.ID,
				Active: boolPtr(true),
			})
		}
		for _, n := range observed.Privacy.Natives {
			suggested.Privacy.Natives = append(suggested.Privacy.Natives, nextdnsv1alpha1.NativeEntry{
				ID:     n.ID,
				Active: boolPtr(true),
			})
		}
	}

	// ParentalControl: bool -> *bool, categories/services preserve Active as *bool
	if observed.ParentalControl != nil {
		suggested.ParentalControl = &nextdnsv1alpha1.ParentalControlSpec{
			SafeSearch:            boolPtr(observed.ParentalControl.SafeSearch),
			YouTubeRestrictedMode: boolPtr(observed.ParentalControl.YouTubeRestrictedMode),
			BlockBypass:           boolPtr(observed.ParentalControl.BlockBypass),
		}
		for _, cat := range observed.ParentalControl.Categories {
			suggested.ParentalControl.Categories = append(suggested.ParentalControl.Categories, nextdnsv1alpha1.CategoryEntry{
				ID:         cat.ID,
				Active:     boolPtr(cat.Active),
				Recreation: boolPtr(cat.Recreation),
			})
		}
		for _, svc := range observed.ParentalControl.Services {
			suggested.ParentalControl.Services = append(suggested.ParentalControl.Services, nextdnsv1alpha1.ServiceEntry{
				ID:     svc.ID,
				Active: boolPtr(svc.Active),
			})
		}
	}

	// Denylist/Allowlist: Active bool -> *bool
	for _, d := range observed.Denylist {
		suggested.Denylist = append(suggested.Denylist, nextdnsv1alpha1.DomainEntry{
			Domain: d.Domain,
			Active: boolPtr(d.Active),
		})
	}
	for _, a := range observed.Allowlist {
		suggested.Allowlist = append(suggested.Allowlist, nextdnsv1alpha1.DomainEntry{
			Domain: a.Domain,
			Active: boolPtr(a.Active),
		})
	}

	// Rewrites: ObservedRewriteEntry (Name/Content) -> RewriteEntry (From/To)
	for _, rw := range observed.Rewrites {
		suggested.Rewrites = append(suggested.Rewrites, nextdnsv1alpha1.RewriteEntry{
			From:   rw.Name,
			To:     rw.Content,
			Active: boolPtr(true),
		})
	}

	// Settings: bool -> *bool, retention int -> string
	if observed.Settings != nil {
		suggested.Settings = &nextdnsv1alpha1.SettingsSpec{
			Web3: boolPtr(observed.Settings.Web3),
		}
		if observed.Settings.Logs != nil {
			suggested.Settings.Logs = &nextdnsv1alpha1.LogsSpec{
				Enabled:       boolPtr(observed.Settings.Logs.Enabled),
				Retention:     formatRetentionString(observed.Settings.Logs.Retention),
				Location:      observed.Settings.Logs.Location,
				LogClientsIPs: boolPtr(observed.Settings.Logs.LogClientsIPs),
				LogDomains:    boolPtr(observed.Settings.Logs.LogDomains),
			}
		}
		if observed.Settings.BlockPage != nil {
			suggested.Settings.BlockPage = &nextdnsv1alpha1.BlockPageSpec{
				Enabled: boolPtr(observed.Settings.BlockPage.Enabled),
			}
		}
		if observed.Settings.Performance != nil {
			suggested.Settings.Performance = &nextdnsv1alpha1.PerformanceSpec{
				ECS:             boolPtr(observed.Settings.Performance.ECS),
				CacheBoost:      boolPtr(observed.Settings.Performance.CacheBoost),
				CNAMEFlattening: boolPtr(observed.Settings.Performance.CNAMEFlattening),
			}
		}
	}

	return suggested
}

// specHasConfig checks if the spec has any configuration sections populated
func specHasConfig(spec *nextdnsv1alpha1.NextDNSProfileSpec) bool {
	return spec.Security != nil ||
		spec.Privacy != nil ||
		spec.ParentalControl != nil ||
		spec.Settings != nil ||
		len(spec.Denylist) > 0 ||
		len(spec.Allowlist) > 0 ||
		len(spec.Rewrites) > 0 ||
		len(spec.DenylistRefs) > 0 ||
		len(spec.AllowlistRefs) > 0 ||
		len(spec.TLDListRefs) > 0
}

// reconcileConfigMap creates or updates the ConfigMap with connection details
func (r *NextDNSProfileReconciler) reconcileConfigMap(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile) error {
	// Skip if ConfigMapRef is not enabled
	if profile.Spec.ConfigMapRef == nil || !profile.Spec.ConfigMapRef.Enabled {
		return nil
	}

	// Skip if we don't have a profile ID yet
	if profile.Status.ProfileID == "" {
		return nil
	}

	logger := log.FromContext(ctx)

	// Determine ConfigMap name
	configMapName := profile.Spec.ConfigMapRef.Name
	if configMapName == "" {
		configMapName = profile.Name + "-nextdns"
	}

	profileID := profile.Status.ProfileID

	// Build ConfigMap data with DNS protocol endpoints.
	// Note: These use {profileID}.dns.nextdns.io which is the DNS server hostname,
	// NOT the API fingerprint (status.fingerprint). These are different concepts.
	data := map[string]string{
		"NEXTDNS_PROFILE_ID": profileID,
		"NEXTDNS_DOT":        fmt.Sprintf("%s.dns.nextdns.io", profileID),
		"NEXTDNS_DOH":        fmt.Sprintf("https://dns.nextdns.io/%s", profileID),
		"NEXTDNS_DOQ":        fmt.Sprintf("quic://%s.dns.nextdns.io", profileID),
		"NEXTDNS_IPV4_1":     "45.90.28.0",
		"NEXTDNS_IPV4_2":     "45.90.30.0",
		"NEXTDNS_IPV6_1":     "2a07:a8c0::",
		"NEXTDNS_IPV6_2":     "2a07:a8c1::",
	}

	// Check if ConfigMap already exists
	existingConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: profile.Namespace,
	}, existingConfigMap)

	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get ConfigMap: %w", err)
		}

		// Create new ConfigMap
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: profile.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(profile, nextdnsv1alpha1.GroupVersion.WithKind("NextDNSProfile")),
				},
			},
			Data: data,
		}

		if err := r.Create(ctx, configMap); err != nil {
			return fmt.Errorf("failed to create ConfigMap: %w", err)
		}
		logger.Info("Created ConfigMap with connection details", "configMap", configMapName)
		return nil
	}

	// Update existing ConfigMap
	existingConfigMap.Data = data
	// Ensure owner reference is set
	if len(existingConfigMap.OwnerReferences) == 0 {
		existingConfigMap.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(profile, nextdnsv1alpha1.GroupVersion.WithKind("NextDNSProfile")),
		}
	}

	if err := r.Update(ctx, existingConfigMap); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}
	logger.V(1).Info("Updated ConfigMap with connection details", "configMap", configMapName)
	return nil
}

// boolValue returns the value of a bool pointer, or the default if nil
func boolValue(ptr *bool, defaultValue bool) bool {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// parseRetentionDays parses a retention string (e.g., "7d", "30d") and returns days as int
func parseRetentionDays(retention string) int {
	if retention == "" {
		return 7 // default 7 days
	}

	retention = strings.TrimSpace(strings.ToLower(retention))

	// Handle special cases
	switch retention {
	case "1h":
		return 0 // Less than a day
	case "6h":
		return 0
	case "1y":
		return 365
	case "2y":
		return 730
	}

	// Parse numeric values with 'd' suffix
	if strings.HasSuffix(retention, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(retention, "d"))
		if err == nil {
			return days
		}
	}

	return 7 // default
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// formatRetentionString converts a retention value in days to the spec string format
// formatRetentionString converts a retention value in days to the nearest valid
// CRD enum value. The NextDNS API may return unexpected values (e.g., 31536000
// for "unlimited"), so we clamp to the nearest supported retention period.
// Valid values: 1h, 6h, 1d, 7d, 30d, 90d, 1y, 2y
func formatRetentionString(days int) string {
	switch {
	case days <= 0:
		return "1h"
	case days == 1:
		return "1d"
	case days <= 7:
		return "7d"
	case days <= 30:
		return "30d"
	case days <= 90:
		return "90d"
	case days <= 365:
		return "1y"
	default:
		return "2y"
	}
}

// setCondition sets a condition on the profile
func (r *NextDNSProfileReconciler) setCondition(profile *nextdnsv1alpha1.NextDNSProfile, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&profile.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: profile.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// findProfilesForAllowlist returns reconcile requests for profiles referencing the allowlist
func (r *NextDNSProfileReconciler) findProfilesForAllowlist(ctx context.Context, obj client.Object) []reconcile.Request {
	allowlist, ok := obj.(*nextdnsv1alpha1.NextDNSAllowlist)
	if !ok {
		return nil
	}

	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list profiles for allowlist watch")
		return nil
	}

	var requests []reconcile.Request
	for _, profile := range profiles.Items {
		for _, ref := range profile.Spec.AllowlistRefs {
			refNs := ref.Namespace
			if refNs == "" {
				refNs = profile.Namespace
			}
			if ref.Name == allowlist.Name && refNs == allowlist.Namespace {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      profile.Name,
						Namespace: profile.Namespace,
					},
				})
				break
			}
		}
	}
	return requests
}

// findProfilesForDenylist returns reconcile requests for profiles referencing the denylist
func (r *NextDNSProfileReconciler) findProfilesForDenylist(ctx context.Context, obj client.Object) []reconcile.Request {
	denylist, ok := obj.(*nextdnsv1alpha1.NextDNSDenylist)
	if !ok {
		return nil
	}

	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list profiles for denylist watch")
		return nil
	}

	var requests []reconcile.Request
	for _, profile := range profiles.Items {
		for _, ref := range profile.Spec.DenylistRefs {
			refNs := ref.Namespace
			if refNs == "" {
				refNs = profile.Namespace
			}
			if ref.Name == denylist.Name && refNs == denylist.Namespace {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      profile.Name,
						Namespace: profile.Namespace,
					},
				})
				break
			}
		}
	}
	return requests
}

// findProfilesForTLDList returns reconcile requests for profiles referencing the TLD list
func (r *NextDNSProfileReconciler) findProfilesForTLDList(ctx context.Context, obj client.Object) []reconcile.Request {
	tldList, ok := obj.(*nextdnsv1alpha1.NextDNSTLDList)
	if !ok {
		return nil
	}

	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list profiles for TLD list watch")
		return nil
	}

	var requests []reconcile.Request
	for _, profile := range profiles.Items {
		for _, ref := range profile.Spec.TLDListRefs {
			refNs := ref.Namespace
			if refNs == "" {
				refNs = profile.Namespace
			}
			if ref.Name == tldList.Name && refNs == tldList.Namespace {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      profile.Name,
						Namespace: profile.Namespace,
					},
				})
				break
			}
		}
	}
	return requests
}

// findProfilesForSecret returns reconcile requests for profiles referencing the secret.
// Uses a field index on credentialsRef for efficient lookups instead of listing all profiles.
// Matches both same-namespace references (credentialsRef.namespace empty) and
// cross-namespace references (credentialsRef.namespace explicitly set).
func (r *NextDNSProfileReconciler) findProfilesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	// Use field index to efficiently find profiles referencing this secret
	var profiles nextdnsv1alpha1.NextDNSProfileList
	indexKey := secret.Namespace + "/" + secret.Name
	if err := r.List(ctx, &profiles, client.MatchingFields{credentialsRefIndexField: indexKey}); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list profiles for secret watch")
		return nil
	}

	var requests []reconcile.Request
	for _, profile := range profiles.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      profile.Name,
				Namespace: profile.Namespace,
			},
		})
	}
	return requests
}

// findProfilesForConfigMap returns reconcile requests for profiles that
// reference the given ConfigMap via owner references (output ConfigMap from configMapRef)
func (r *NextDNSProfileReconciler) findProfilesForConfigMap(ctx context.Context, obj client.Object) []reconcile.Request {
	configMap, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}

	var requests []reconcile.Request

	// Check owner references (output ConfigMap from configMapRef)
	for _, ref := range configMap.OwnerReferences {
		if ref.Kind == "NextDNSProfile" && ref.APIVersion == nextdnsv1alpha1.GroupVersion.String() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      ref.Name,
					Namespace: configMap.Namespace,
				},
			})
		}
	}

	return requests
}

// updateResourceMetrics updates the gauge metrics for resource counts
func (r *NextDNSProfileReconciler) updateResourceMetrics(ctx context.Context) {
	// Count profiles
	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles); err == nil {
		metrics.ProfilesTotal.Set(float64(len(profiles.Items)))
	}

	// Count allowlists
	var allowlists nextdnsv1alpha1.NextDNSAllowlistList
	if err := r.List(ctx, &allowlists); err == nil {
		metrics.AllowlistsTotal.Set(float64(len(allowlists.Items)))
	}

	// Count denylists
	var denylists nextdnsv1alpha1.NextDNSDenylistList
	if err := r.List(ctx, &denylists); err == nil {
		metrics.DenylistsTotal.Set(float64(len(denylists.Items)))
	}

	// Count TLD lists
	var tldlists nextdnsv1alpha1.NextDNSTLDListList
	if err := r.List(ctx, &tldlists); err == nil {
		metrics.TLDListsTotal.Set(float64(len(tldlists.Items)))
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *NextDNSProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Register field index for efficient secret reference lookups
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&nextdnsv1alpha1.NextDNSProfile{},
		credentialsRefIndexField,
		credentialsRefIndexFunc,
	); err != nil {
		return fmt.Errorf("failed to create field index for credentialsRef: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nextdnsv1alpha1.NextDNSProfile{}).
		Watches(
			&nextdnsv1alpha1.NextDNSAllowlist{},
			handler.EnqueueRequestsFromMapFunc(r.findProfilesForAllowlist),
		).
		Watches(
			&nextdnsv1alpha1.NextDNSDenylist{},
			handler.EnqueueRequestsFromMapFunc(r.findProfilesForDenylist),
		).
		Watches(
			&nextdnsv1alpha1.NextDNSTLDList{},
			handler.EnqueueRequestsFromMapFunc(r.findProfilesForTLDList),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findProfilesForSecret),
		).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.findProfilesForConfigMap),
		).
		Complete(r)
}
