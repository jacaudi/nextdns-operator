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
	// DenylistFinalizerName is the finalizer added to NextDNSDenylist resources
	DenylistFinalizerName = "nextdns.io/denylist-finalizer"
)

// NextDNSDenylistReconciler reconciles a NextDNSDenylist object
type NextDNSDenylistReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	SyncPeriod time.Duration
}

// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsdenylists,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsdenylists/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsdenylists/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NextDNSDenylistReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the denylist
	var list nextdnsv1alpha1.NextDNSDenylist
	if err := r.Get(ctx, req.NamespacedName, &list); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Migrate old finalizer name if present
	if migrated, err := migrateFinalizerDomain(ctx, r.Client, &list, "nextdns.jacaudi.com/denylist-finalizer", DenylistFinalizerName); err != nil {
		return ctrl.Result{}, err
	} else if migrated {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Handle deletion
	if !list.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &list)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&list, DenylistFinalizerName) {
		logger.Info("Adding finalizer to NextDNSDenylist")
		controllerutil.AddFinalizer(&list, DenylistFinalizerName)
		if err := r.Update(ctx, &list); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Count active domains
	count := countActiveDomains(list.Spec.Domains)

	// Find profile references
	profileRefs, err := r.findProfileReferences(ctx, &list)
	if err != nil {
		logger.Error(err, "Failed to find profile references")
		return ctrl.Result{}, err
	}

	// Update status
	list.Status.DomainCount = count
	list.Status.ProfileRefs = profileRefs

	// Set conditions
	setListConditions(&list.Status.Conditions, count, len(profileRefs), "domains")

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
func (r *NextDNSDenylistReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nextdnsv1alpha1.NextDNSDenylist{}).
		Watches(
			&nextdnsv1alpha1.NextDNSProfile{},
			handler.EnqueueRequestsFromMapFunc(r.findDenylistsForProfile),
		).
		Complete(r)
}

// findDenylistsForProfile returns reconcile requests for all denylists referenced by a profile
func (r *NextDNSDenylistReconciler) findDenylistsForProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profile, ok := obj.(*nextdnsv1alpha1.NextDNSProfile)
	if !ok {
		return nil
	}

	var requests []reconcile.Request
	for _, ref := range profile.Spec.DenylistRefs {
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

// findProfileReferences finds all profiles that reference this denylist.
// Note: Searches cluster-wide to support cross-namespace references.
func (r *NextDNSDenylistReconciler) findProfileReferences(ctx context.Context, list *nextdnsv1alpha1.NextDNSDenylist) ([]nextdnsv1alpha1.ResourceReference, error) {
	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles); err != nil {
		return nil, err
	}

	return findRefsForList(profiles.Items, list.Name, list.Namespace, func(spec *nextdnsv1alpha1.NextDNSProfileSpec) []nextdnsv1alpha1.ListReference {
		return spec.DenylistRefs
	}), nil
}

// handleDeletion handles the deletion of a denylist
func (r *NextDNSDenylistReconciler) handleDeletion(ctx context.Context, list *nextdnsv1alpha1.NextDNSDenylist) (ctrl.Result, error) {
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
	logger.Info("Removing finalizer from NextDNSDenylist")
	controllerutil.RemoveFinalizer(list, DenylistFinalizerName)
	if err := r.Update(ctx, list); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

