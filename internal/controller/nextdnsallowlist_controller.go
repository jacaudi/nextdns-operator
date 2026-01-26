package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

const (
	// AllowlistFinalizerName is the finalizer added to NextDNSAllowlist resources
	AllowlistFinalizerName = "nextdns.jacaudi.com/allowlist-finalizer"
)

// NextDNSAllowlistReconciler reconciles a NextDNSAllowlist object
type NextDNSAllowlistReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nextdns.jacaudi.com,resources=nextdnsallowlists,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nextdns.jacaudi.com,resources=nextdnsallowlists/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nextdns.jacaudi.com,resources=nextdnsallowlists/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NextDNSAllowlistReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the allowlist
	var list nextdnsv1alpha1.NextDNSAllowlist
	if err := r.Get(ctx, req.NamespacedName, &list); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !list.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &list)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&list, AllowlistFinalizerName) {
		logger.Info("Adding finalizer to NextDNSAllowlist")
		controllerutil.AddFinalizer(&list, AllowlistFinalizerName)
		if err := r.Update(ctx, &list); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Count active domains
	count := r.countActiveDomains(list.Spec.Domains)

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
	r.setConditions(&list, count, len(profileRefs))

	// Update status subresource
	if err := r.Status().Update(ctx, &list); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NextDNSAllowlistReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nextdnsv1alpha1.NextDNSAllowlist{}).
		Complete(r)
}

// countActiveDomains counts the number of domains where Active is nil or true
func (r *NextDNSAllowlistReconciler) countActiveDomains(domains []nextdnsv1alpha1.DomainEntry) int {
	count := 0
	for _, domain := range domains {
		// If Active is nil or true, the domain is active
		if domain.Active == nil || *domain.Active {
			count++
		}
	}
	return count
}

// findProfileReferences finds all profiles that reference this allowlist
func (r *NextDNSAllowlistReconciler) findProfileReferences(ctx context.Context, list *nextdnsv1alpha1.NextDNSAllowlist) ([]nextdnsv1alpha1.ResourceReference, error) {
	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles); err != nil {
		return nil, err
	}

	var refs []nextdnsv1alpha1.ResourceReference

	for _, profile := range profiles.Items {
		for _, ref := range profile.Spec.AllowlistRefs {
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

// handleDeletion handles the deletion of an allowlist
func (r *NextDNSAllowlistReconciler) handleDeletion(ctx context.Context, list *nextdnsv1alpha1.NextDNSAllowlist) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Check if any profiles reference this list
	if len(list.Status.ProfileRefs) > 0 {
		logger.Info("Deletion blocked - list is in use", "profileRefs", list.Status.ProfileRefs)

		// Set DeletionBlocked condition
		meta.SetStatusCondition(&list.Status.Conditions, metav1.Condition{
			Type:    "DeletionBlocked",
			Status:  metav1.ConditionTrue,
			Reason:  "InUseByProfiles",
			Message: fmt.Sprintf("Cannot delete: used by profiles %v. Remove references first.", formatProfileRefs(list.Status.ProfileRefs)),
		})

		// Update status and requeue
		if err := r.Status().Update(ctx, list); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// No references - safe to delete
	logger.Info("Removing finalizer from NextDNSAllowlist")
	controllerutil.RemoveFinalizer(list, AllowlistFinalizerName)
	if err := r.Update(ctx, list); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// setConditions sets status conditions based on current state
func (r *NextDNSAllowlistReconciler) setConditions(list *nextdnsv1alpha1.NextDNSAllowlist, count, refCount int) {
	// Valid condition
	meta.SetStatusCondition(&list.Status.Conditions, metav1.Condition{
		Type:    "Valid",
		Status:  metav1.ConditionTrue,
		Reason:  "AllDomainsValid",
		Message: fmt.Sprintf("All %d domains are valid", count),
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

// formatProfileRefs formats profile references for display
func formatProfileRefs(refs []nextdnsv1alpha1.ResourceReference) string {
	var names []string
	for _, ref := range refs {
		if ref.Namespace != "" {
			names = append(names, fmt.Sprintf("%s/%s", ref.Namespace, ref.Name))
		} else {
			names = append(names, ref.Name)
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(names, ", "))
}
