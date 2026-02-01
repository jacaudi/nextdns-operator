package controller

import (
	"context"
	"fmt"
	"time"

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
	// TLDListFinalizerName is the finalizer added to NextDNSTLDList resources
	TLDListFinalizerName = "nextdns.jacaudi.com/tldlist-finalizer"
)

// NextDNSTLDListReconciler reconciles a NextDNSTLDList object
type NextDNSTLDListReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	SyncPeriod time.Duration
}

// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnstldlists,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnstldlists/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnstldlists/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NextDNSTLDListReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the tldlist
	var list nextdnsv1alpha1.NextDNSTLDList
	if err := r.Get(ctx, req.NamespacedName, &list); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !list.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &list)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&list, TLDListFinalizerName) {
		logger.Info("Adding finalizer to NextDNSTLDList")
		controllerutil.AddFinalizer(&list, TLDListFinalizerName)
		if err := r.Update(ctx, &list); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Count active TLDs
	count := r.countActiveTLDs(list.Spec.TLDs)

	// Find profile references
	profileRefs, err := r.findProfileReferences(ctx, &list)
	if err != nil {
		logger.Error(err, "Failed to find profile references")
		return ctrl.Result{}, err
	}

	// Update status
	list.Status.TLDCount = count
	list.Status.ProfileRefs = profileRefs

	// Set conditions
	r.setConditions(&list, count, len(profileRefs))

	// Update status subresource
	if err := r.Status().Update(ctx, &list); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Schedule next sync with jitter for drift detection
	syncInterval := CalculateSyncInterval(r.SyncPeriod)
	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NextDNSTLDListReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nextdnsv1alpha1.NextDNSTLDList{}).
		Watches(
			&nextdnsv1alpha1.NextDNSProfile{},
			handler.EnqueueRequestsFromMapFunc(r.findTLDListsForProfile),
		).
		Complete(r)
}

// findTLDListsForProfile returns reconcile requests for all tldlists referenced by a profile
func (r *NextDNSTLDListReconciler) findTLDListsForProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profile, ok := obj.(*nextdnsv1alpha1.NextDNSProfile)
	if !ok {
		return nil
	}

	var requests []reconcile.Request
	for _, ref := range profile.Spec.TLDListRefs {
		namespace := ref.Namespace
		if namespace == "" {
			namespace = profile.Namespace
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      ref.Name,
				Namespace: namespace,
			},
		})
	}

	return requests
}

// countActiveTLDs counts the number of TLDs where Active is nil or true
func (r *NextDNSTLDListReconciler) countActiveTLDs(tlds []nextdnsv1alpha1.TLDEntry) int {
	count := 0
	for _, tld := range tlds {
		// If Active is nil or true, the TLD is active
		if tld.Active == nil || *tld.Active {
			count++
		}
	}
	return count
}

// findProfileReferences finds all profiles that reference this tldlist
// Note: Searches cluster-wide to support cross-namespace references
func (r *NextDNSTLDListReconciler) findProfileReferences(ctx context.Context, list *nextdnsv1alpha1.NextDNSTLDList) ([]nextdnsv1alpha1.ResourceReference, error) {
	var profiles nextdnsv1alpha1.NextDNSProfileList
	// List all profiles cluster-wide to support cross-namespace references
	if err := r.List(ctx, &profiles); err != nil {
		return nil, err
	}

	var refs []nextdnsv1alpha1.ResourceReference

	for _, profile := range profiles.Items {
		for _, ref := range profile.Spec.TLDListRefs {
			// Determine the namespace of the referenced list
			namespace := ref.Namespace
			if namespace == "" {
				namespace = profile.Namespace
			}

			// Check if this profile references our list
			if ref.Name == list.Name && namespace == list.Namespace {
				refs = append(refs, nextdnsv1alpha1.ResourceReference{
					Name:      profile.Name,
					Namespace: profile.Namespace,
				})
				break
			}
		}
	}

	return refs, nil
}

// handleDeletion handles the deletion of an tldlist
func (r *NextDNSTLDListReconciler) handleDeletion(ctx context.Context, list *nextdnsv1alpha1.NextDNSTLDList) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Check if any profiles reference this list
	if len(list.Status.ProfileRefs) > 0 {
		logger.Info("Deletion blocked - list is in use", "profileRefs", list.Status.ProfileRefs)

		// Set DeletionBlocked condition
		meta.SetStatusCondition(&list.Status.Conditions, metav1.Condition{
			Type:    "DeletionBlocked",
			Status:  metav1.ConditionTrue,
			Reason:  "InUseByProfiles",
			Message: fmt.Sprintf("Cannot delete: used by profiles %s. Remove references first.", formatProfileRefs(list.Status.ProfileRefs)),
		})

		// Update status and requeue
		if err := r.Status().Update(ctx, list); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// No references - safe to delete
	logger.Info("Removing finalizer from NextDNSTLDList")
	controllerutil.RemoveFinalizer(list, TLDListFinalizerName)
	if err := r.Update(ctx, list); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// setConditions sets status conditions based on current state
func (r *NextDNSTLDListReconciler) setConditions(list *nextdnsv1alpha1.NextDNSTLDList, count, refCount int) {
	// Valid condition
	meta.SetStatusCondition(&list.Status.Conditions, metav1.Condition{
		Type:    "Valid",
		Status:  metav1.ConditionTrue,
		Reason:  "AllDomainsValid",
		Message: fmt.Sprintf("All %d TLDs are valid", count),
	})

	// InUse condition
	if refCount > 0 {
		meta.SetStatusCondition(&list.Status.Conditions, metav1.Condition{
			Type:    "InUse",
			Status:  metav1.ConditionTrue,
			Reason:  "ReferencedByProfiles",
			Message: fmt.Sprintf("Used by %d profile(s)", refCount),
		})
	} else {
		meta.SetStatusCondition(&list.Status.Conditions, metav1.Condition{
			Type:    "InUse",
			Status:  metav1.ConditionFalse,
			Reason:  "NotReferenced",
			Message: "Not used by any profiles",
		})
	}

	// Clear DeletionBlocked if it was set
	if refCount == 0 {
		meta.RemoveStatusCondition(&list.Status.Conditions, "DeletionBlocked")
	}
}
