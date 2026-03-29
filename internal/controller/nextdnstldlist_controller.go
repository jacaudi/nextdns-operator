package controller

import (
	"context"
	"time"

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
	TLDListFinalizerName = "nextdns.io/tldlist-finalizer"
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

	// Migrate old finalizer name if present
	if migrated, err := migrateFinalizerDomain(ctx, r.Client, &list, "nextdns.jacaudi.com/tldlist-finalizer", TLDListFinalizerName); err != nil {
		return ctrl.Result{}, err
	} else if migrated {
		return ctrl.Result{RequeueAfter: time.Second}, nil
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
	count := countActiveTLDs(list.Spec.TLDs)

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
	setListConditions(&list.Status.Conditions, count, len(profileRefs), "TLDs")

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

// findProfileReferences finds all profiles that reference this TLD list.
// Note: Searches cluster-wide to support cross-namespace references.
func (r *NextDNSTLDListReconciler) findProfileReferences(ctx context.Context, list *nextdnsv1alpha1.NextDNSTLDList) ([]nextdnsv1alpha1.ResourceReference, error) {
	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles); err != nil {
		return nil, err
	}

	return findRefsForList(profiles.Items, list.Name, list.Namespace, func(spec *nextdnsv1alpha1.NextDNSProfileSpec) []nextdnsv1alpha1.ListReference {
		return spec.TLDListRefs
	}), nil
}

// handleDeletion handles the deletion of a TLD list
func (r *NextDNSTLDListReconciler) handleDeletion(ctx context.Context, list *nextdnsv1alpha1.NextDNSTLDList) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Check if any profiles reference this list
	if len(list.Status.ProfileRefs) > 0 {
		logger.Info("Deletion blocked - list is in use", "profileRefs", list.Status.ProfileRefs)

		setDeletionBlockedCondition(&list.Status.Conditions, list.Status.ProfileRefs)

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
