package controller

import (
	"context"
	"fmt"
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

// NextDNSProfileReconciler reconciles a NextDNSProfile object
type NextDNSProfileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles/finalizers,verbs=update
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsallowlists,verbs=get;list;watch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsdenylists,verbs=get;list;watch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnstldlists,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *NextDNSProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

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
	if !profile.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, profile)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(profile, FinalizerName) {
		logger.Info("Adding finalizer to NextDNSProfile")
		controllerutil.AddFinalizer(profile, FinalizerName)
		if err := r.Update(ctx, profile); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Get API credentials
	apiKey, err := r.getAPIKey(ctx, profile)
	if err != nil {
		logger.Error(err, "Failed to get API credentials")
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
		r.setCondition(profile, ConditionTypeSynced, metav1.ConditionFalse, "SyncFailed", err.Error())
		r.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "SyncFailed", "Failed to sync with NextDNS API")
		if updateErr := r.Status().Update(ctx, profile); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

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

	return ctrl.Result{}, nil
}

// handleDeletion handles the deletion of a NextDNSProfile
func (r *NextDNSProfileReconciler) handleDeletion(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(profile, FinalizerName) {
		logger.Info("Handling deletion of NextDNSProfile")

		// TODO: Delete profile from NextDNS API if it was created by us
		// For now, we just remove the finalizer

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
	Allowlist      []string
	Denylist       []string
	TLDs           []string
	ResourceStatus *nextdnsv1alpha1.ReferencedResources
}

// resolveListReferences resolves all list references and merges with inline lists
func (r *NextDNSProfileReconciler) resolveListReferences(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile) (*ResolvedLists, error) {
	resolved := &ResolvedLists{
		Allowlist: make([]string, 0),
		Denylist:  make([]string, 0),
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
			if entry.Active == nil || *entry.Active {
				resolved.Allowlist = append(resolved.Allowlist, entry.Domain)
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
		if entry.Active == nil || *entry.Active {
			resolved.Allowlist = append(resolved.Allowlist, entry.Domain)
		}
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
			if entry.Active == nil || *entry.Active {
				resolved.Denylist = append(resolved.Denylist, entry.Domain)
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
		if entry.Active == nil || *entry.Active {
			resolved.Denylist = append(resolved.Denylist, entry.Domain)
		}
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

	// TODO: Implement actual NextDNS API sync using github.com/jacaudi/nextdns-go
	// For now, this is a placeholder that simulates the sync

	logger.Info("Syncing with NextDNS API",
		"profileName", profile.Spec.Name,
		"profileID", profile.Spec.ProfileID)

	// If no profile ID is set, we would create a new profile
	if profile.Status.ProfileID == "" {
		if profile.Spec.ProfileID != "" {
			// Adopt existing profile
			profile.Status.ProfileID = profile.Spec.ProfileID
		} else {
			// TODO: Create new profile via API
			// For now, generate a placeholder ID
			profile.Status.ProfileID = "placeholder-id"
		}
		profile.Status.Fingerprint = profile.Status.ProfileID + ".dns.nextdns.io"
	}

	return nil
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
