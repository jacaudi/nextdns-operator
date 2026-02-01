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

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
	"github.com/jacaudi/nextdns-operator/internal/metrics"
	"github.com/jacaudi/nextdns-operator/internal/nextdns"
)

const (
	// FinalizerName is the finalizer used by this controller
	FinalizerName = "nextdns.io/finalizer"

	// ConditionTypeReady indicates the profile is ready
	ConditionTypeReady = "Ready"

	// ConditionTypeSynced indicates the profile is synced with NextDNS
	ConditionTypeSynced = "Synced"

	// ConditionTypeReferencesResolved indicates all references are resolved
	ConditionTypeReferencesResolved = "ReferencesResolved"
)

// ClientFactory is a function that creates a NextDNS client
type ClientFactory func(apiKey string) (nextdns.ClientInterface, error)

// DefaultClientFactory creates a real NextDNS client
func DefaultClientFactory(apiKey string) (nextdns.ClientInterface, error) {
	return nextdns.NewClient(apiKey)
}

// NextDNSProfileReconciler reconciles a NextDNSProfile object
type NextDNSProfileReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ClientFactory ClientFactory
	SyncPeriod    time.Duration
}

// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles/finalizers,verbs=update
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsallowlists,verbs=get;list;watch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsdenylists,verbs=get;list;watch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnstldlists,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *NextDNSProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Update resource count metrics
	r.updateResourceMetrics(ctx)

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
		if profile.Spec.ProfileID == "" && profile.Status.ProfileID != "" {
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

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: profile.Namespace,
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	apiKey, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", secretKey, secretName)
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
		if profile.Spec.ProfileID != "" {
			// Adopt existing profile - verify it exists
			_, err := client.GetProfile(ctx, profile.Spec.ProfileID)
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
		}
		profile.Status.Fingerprint = profile.Status.ProfileID + ".dns.nextdns.io"
	}

	profileID := profile.Status.ProfileID

	// Update profile name if needed
	if err := client.UpdateProfile(ctx, profileID, profile.Spec.Name); err != nil {
		return fmt.Errorf("failed to update profile name: %w", err)
	}

	// Sync security settings
	if profile.Spec.Security != nil {
		securityConfig := &nextdns.SecurityConfig{
			AIThreatDetection:  boolValue(profile.Spec.Security.AIThreatDetection, true),
			GoogleSafeBrowsing: boolValue(profile.Spec.Security.GoogleSafeBrowsing, true),
			Cryptojacking:      boolValue(profile.Spec.Security.Cryptojacking, true),
			DNSRebinding:       boolValue(profile.Spec.Security.DNSRebinding, true),
			IDNHomographs:      boolValue(profile.Spec.Security.IDNHomographs, true),
			Typosquatting:      boolValue(profile.Spec.Security.Typosquatting, true),
			DGA:                boolValue(profile.Spec.Security.DGA, true),
			NRD:                boolValue(profile.Spec.Security.NRD, false),
			DDNS:               boolValue(profile.Spec.Security.DDNS, false),
			Parking:            boolValue(profile.Spec.Security.Parking, true),
			CSAM:               boolValue(profile.Spec.Security.CSAM, true),
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
		}
		if err := client.UpdateParentalControl(ctx, profileID, pcConfig); err != nil {
			return fmt.Errorf("failed to update parental control settings: %w", err)
		}
	}

	// Sync settings (logs, block page)
	if profile.Spec.Settings != nil {
		settingsConfig := &nextdns.SettingsConfig{
			LogsEnabled:     true,
			BlockPageEnable: true,
		}
		if profile.Spec.Settings.Logs != nil {
			settingsConfig.LogsEnabled = boolValue(profile.Spec.Settings.Logs.Enabled, true)
			settingsConfig.LogRetention = parseRetentionDays(profile.Spec.Settings.Logs.Retention)
		}
		if profile.Spec.Settings.BlockPage != nil {
			settingsConfig.BlockPageEnable = boolValue(profile.Spec.Settings.BlockPage.Enabled, true)
		}
		if err := client.UpdateSettings(ctx, profileID, settingsConfig); err != nil {
			return fmt.Errorf("failed to update settings: %w", err)
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

// findProfilesForSecret returns reconcile requests for profiles referencing the secret
func (r *NextDNSProfileReconciler) findProfilesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles, client.InNamespace(secret.Namespace)); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, profile := range profiles.Items {
		if profile.Spec.CredentialsRef.Name == secret.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      profile.Name,
					Namespace: profile.Namespace,
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
		Complete(r)
}
