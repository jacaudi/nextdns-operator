package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
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
	_ = log.FromContext(ctx)

	// TODO: Implement reconciliation logic

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
