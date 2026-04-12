package controller

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

// gatewayProxyStrategy handles implementation-specific proxy replica
// configuration for a particular gateway controller.
type gatewayProxyStrategy interface {
	// ReconcileProxyReplicas creates or updates the implementation-specific
	// CR and returns a GatewayParametersReference pointing at it.
	ReconcileProxyReplicas(ctx context.Context, c client.Client, scheme *runtime.Scheme, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, replicas int32) (*nextdnsv1alpha1.GatewayParametersReference, error)

	// CleanupProxyReplicas deletes the implementation-specific CR.
	CleanupProxyReplicas(ctx context.Context, c client.Client, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error
}

// proxyStrategies maps GatewayClass controllerName values to strategies.
var proxyStrategies = map[string]gatewayProxyStrategy{
	"gateway.envoyproxy.io/gatewayclass-controller": &envoyGatewayStrategy{},
}

// findProxyStrategy returns the strategy for the given controllerName,
// or nil if no strategy supports it.
func findProxyStrategy(controllerName string) gatewayProxyStrategy {
	return proxyStrategies[controllerName]
}

const (
	envoyProxyGroup   = "gateway.envoyproxy.io"
	envoyProxyVersion = "v1alpha1"
	envoyProxyKind    = "EnvoyProxy"
)

type envoyGatewayStrategy struct{}

func envoyProxyName(coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) string {
	return fmt.Sprintf("%s-envoyproxy", coreDNS.Name)
}

func envoyProxyGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   envoyProxyGroup,
		Version: envoyProxyVersion,
		Kind:    envoyProxyKind,
	}
}

func (s *envoyGatewayStrategy) ReconcileProxyReplicas(ctx context.Context, c client.Client, scheme *runtime.Scheme, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, replicas int32) (*nextdnsv1alpha1.GatewayParametersReference, error) {
	name := envoyProxyName(coreDNS)

	desired := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": envoyProxyGroup + "/" + envoyProxyVersion,
			"kind":       envoyProxyKind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": coreDNS.Namespace,
			},
			"spec": map[string]interface{}{
				"provider": map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"envoyDeployment": map[string]interface{}{
							"replicas": int64(replicas),
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(coreDNS, desired, scheme); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on EnvoyProxy: %w", err)
	}

	ref := &nextdnsv1alpha1.GatewayParametersReference{
		Group: envoyProxyGroup,
		Kind:  envoyProxyKind,
		Name:  name,
	}

	// Try to get existing
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(envoyProxyGVK())
	err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: coreDNS.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		if createErr := c.Create(ctx, desired); createErr != nil {
			return nil, fmt.Errorf("failed to create EnvoyProxy: %w", createErr)
		}
		return ref, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get EnvoyProxy: %w", err)
	}

	// Update existing spec
	existing.Object["spec"] = desired.Object["spec"]
	if updateErr := c.Update(ctx, existing); updateErr != nil {
		return nil, fmt.Errorf("failed to update EnvoyProxy: %w", updateErr)
	}
	return ref, nil
}

func (s *envoyGatewayStrategy) CleanupProxyReplicas(ctx context.Context, c client.Client, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(envoyProxyGVK())
	obj.SetName(envoyProxyName(coreDNS))
	obj.SetNamespace(coreDNS.Namespace)
	err := c.Delete(ctx, obj)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
