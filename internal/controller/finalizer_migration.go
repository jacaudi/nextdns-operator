package controller

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// migrateFinalizerDomain replaces old finalizer names (nextdns.jacaudi.com/*)
// with the standardized nextdns.io/* domain. This ensures resources created
// before the finalizer rename can still be deleted after upgrade.
func migrateFinalizerDomain(ctx context.Context, c client.Client, obj client.Object, oldFinalizer, newFinalizer string) (bool, error) {
	if controllerutil.ContainsFinalizer(obj, oldFinalizer) {
		logger := log.FromContext(ctx)
		logger.Info("Migrating finalizer", "old", oldFinalizer, "new", newFinalizer)
		controllerutil.RemoveFinalizer(obj, oldFinalizer)
		controllerutil.AddFinalizer(obj, newFinalizer)
		if err := c.Update(ctx, obj); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
