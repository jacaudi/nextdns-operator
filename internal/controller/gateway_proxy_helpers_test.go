package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

func TestFindProxyStrategy_EnvoyGateway(t *testing.T) {
	strategy := findProxyStrategy("gateway.envoyproxy.io/gatewayclass-controller")
	require.NotNil(t, strategy, "should find strategy for Envoy Gateway")
}

func TestFindProxyStrategy_Unknown(t *testing.T) {
	strategy := findProxyStrategy("example.com/unknown-controller")
	assert.Nil(t, strategy, "should return nil for unknown controller")
}

func TestEnvoyGatewayStrategy_ReconcileProxyReplicas(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       "test-uid",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	strategy := &envoyGatewayStrategy{}
	replicas := int32(3)

	ref, err := strategy.ReconcileProxyReplicas(context.Background(), fakeClient, scheme, coreDNS, replicas)
	require.NoError(t, err)
	require.NotNil(t, ref)

	assert.Equal(t, "gateway.envoyproxy.io", ref.Group)
	assert.Equal(t, "EnvoyProxy", ref.Kind)
	assert.Equal(t, "test-coredns-envoyproxy", ref.Name)

	// Verify the CR was created with correct replicas
	created := &unstructured.Unstructured{}
	created.SetGroupVersionKind(envoyProxyGVK())
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "test-coredns-envoyproxy",
		Namespace: "default",
	}, created)
	require.NoError(t, err)

	// Navigate the unstructured spec to verify replicas
	spec, _ := created.Object["spec"].(map[string]interface{})
	provider, _ := spec["provider"].(map[string]interface{})
	k8s, _ := provider["kubernetes"].(map[string]interface{})
	deployment, _ := k8s["envoyDeployment"].(map[string]interface{})
	assert.Equal(t, int64(3), deployment["replicas"])
}

func TestEnvoyGatewayStrategy_CleanupProxyReplicas(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	// Create the CR first
	existing := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.envoyproxy.io/v1alpha1",
			"kind":       "EnvoyProxy",
			"metadata": map[string]interface{}{
				"name":      "test-coredns-envoyproxy",
				"namespace": "default",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

	strategy := &envoyGatewayStrategy{}
	err := strategy.CleanupProxyReplicas(context.Background(), fakeClient, coreDNS)
	require.NoError(t, err)

	// Verify deleted
	check := &unstructured.Unstructured{}
	check.SetGroupVersionKind(envoyProxyGVK())
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-coredns-envoyproxy", Namespace: "default"}, check)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestEnvoyGatewayStrategy_CleanupProxyReplicas_NotFound(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	strategy := &envoyGatewayStrategy{}
	err := strategy.CleanupProxyReplicas(context.Background(), fakeClient, coreDNS)
	assert.NoError(t, err, "cleanup of non-existent CR should succeed silently")
}

func TestEnvoyGatewayStrategy_ReconcileProxyReplicas_Update(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       "test-uid",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	strategy := &envoyGatewayStrategy{}

	// First reconcile: 2 replicas
	_, err := strategy.ReconcileProxyReplicas(context.Background(), fakeClient, scheme, coreDNS, 2)
	require.NoError(t, err)

	// Second reconcile: 5 replicas
	_, err = strategy.ReconcileProxyReplicas(context.Background(), fakeClient, scheme, coreDNS, 5)
	require.NoError(t, err)

	// Verify updated
	created := &unstructured.Unstructured{}
	created.SetGroupVersionKind(envoyProxyGVK())
	err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "test-coredns-envoyproxy", Namespace: "default"}, created)
	require.NoError(t, err)

	spec, _ := created.Object["spec"].(map[string]interface{})
	provider, _ := spec["provider"].(map[string]interface{})
	k8s, _ := provider["kubernetes"].(map[string]interface{})
	deployment, _ := k8s["envoyDeployment"].(map[string]interface{})
	assert.Equal(t, int64(5), deployment["replicas"])
}
