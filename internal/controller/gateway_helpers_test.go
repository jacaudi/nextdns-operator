package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

func newGatewayTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nextdnsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(gatewayv1.Install(scheme))
	utilruntime.Must(gatewayv1alpha2.Install(scheme))
	return scheme
}

func TestReconcileGateway(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{
						Type:  &ipType,
						Value: "192.168.1.53",
					},
				},
				Annotations: map[string]string{
					"example.com/annotation": "value",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway",
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	// Fetch the created Gateway
	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-coredns-dns",
		Namespace: "default",
	}, gw)
	require.NoError(t, err)

	// Verify name and namespace
	assert.Equal(t, "test-coredns-dns", gw.Name)
	assert.Equal(t, "default", gw.Namespace)

	// Verify GatewayClassName
	assert.Equal(t, gatewayv1.ObjectName("envoy-gateway"), gw.Spec.GatewayClassName)

	// Verify listeners
	require.Len(t, gw.Spec.Listeners, 2)

	// Find listeners by name
	var udpListener, tcpListener gatewayv1.Listener
	for _, l := range gw.Spec.Listeners {
		switch l.Name {
		case "dns-udp":
			udpListener = l
		case "dns-tcp":
			tcpListener = l
		}
	}

	// UDP listener
	assert.Equal(t, gatewayv1.SectionName("dns-udp"), udpListener.Name)
	assert.Equal(t, gatewayv1.PortNumber(53), udpListener.Port)
	assert.Equal(t, gatewayv1.UDPProtocolType, udpListener.Protocol)
	require.NotNil(t, udpListener.AllowedRoutes)
	require.Len(t, udpListener.AllowedRoutes.Kinds, 1)
	assert.Equal(t, gatewayv1.Kind("UDPRoute"), udpListener.AllowedRoutes.Kinds[0].Kind)

	// TCP listener
	assert.Equal(t, gatewayv1.SectionName("dns-tcp"), tcpListener.Name)
	assert.Equal(t, gatewayv1.PortNumber(53), tcpListener.Port)
	assert.Equal(t, gatewayv1.TCPProtocolType, tcpListener.Protocol)
	require.NotNil(t, tcpListener.AllowedRoutes)
	require.Len(t, tcpListener.AllowedRoutes.Kinds, 1)
	assert.Equal(t, gatewayv1.Kind("TCPRoute"), tcpListener.AllowedRoutes.Kinds[0].Kind)

	// Verify addresses
	require.Len(t, gw.Spec.Addresses, 1)
	require.NotNil(t, gw.Spec.Addresses[0].Type)
	assert.Equal(t, gatewayv1.IPAddressType, *gw.Spec.Addresses[0].Type)
	assert.Equal(t, "192.168.1.53", gw.Spec.Addresses[0].Value)

	// Verify annotations
	assert.Equal(t, "value", gw.Annotations["example.com/annotation"])

	// Verify owner reference
	require.Len(t, gw.OwnerReferences, 1)
	assert.Equal(t, "test-coredns", gw.OwnerReferences[0].Name)
	assert.Equal(t, "NextDNSCoreDNS", gw.OwnerReferences[0].Kind)
}

func TestReconcileGateway_CRLevelClassName(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	crClassName := "cilium"
	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				GatewayClassName: &crClassName,
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "192.168.1.53"},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway", // operator default should be overridden
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, gw)
	require.NoError(t, err)

	// CR-level className should win over operator default
	assert.Equal(t, gatewayv1.ObjectName("cilium"), gw.Spec.GatewayClassName)
}

func TestReconcileGateway_NoClassName(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "192.168.1.53"},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "", // no operator default
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no gatewayClassName")
}

func TestReconcileGateway_InfrastructureAnnotations(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "10.10.21.81"},
				},
				Infrastructure: &nextdnsv1alpha1.GatewayInfrastructure{
					Annotations: map[string]string{
						"lbipam.cilium.io/ips": "10.10.21.81",
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway",
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, gw)
	require.NoError(t, err)

	require.NotNil(t, gw.Spec.Infrastructure)
	assert.Equal(t, gatewayv1.AnnotationValue("10.10.21.81"), gw.Spec.Infrastructure.Annotations[gatewayv1.AnnotationKey("lbipam.cilium.io/ips")])
	assert.Nil(t, gw.Spec.Infrastructure.Labels)
	assert.Nil(t, gw.Spec.Infrastructure.ParametersRef)
}

func TestReconcileGateway_InfrastructureLabels(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "10.10.21.81"},
				},
				Infrastructure: &nextdnsv1alpha1.GatewayInfrastructure{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "nextdns-operator",
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway",
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, gw)
	require.NoError(t, err)

	require.NotNil(t, gw.Spec.Infrastructure)
	assert.Nil(t, gw.Spec.Infrastructure.Annotations)
	assert.Equal(t, gatewayv1.LabelValue("nextdns-operator"), gw.Spec.Infrastructure.Labels[gatewayv1.LabelKey("app.kubernetes.io/managed-by")])
	assert.Nil(t, gw.Spec.Infrastructure.ParametersRef)
}

func TestReconcileGateway_InfrastructureParametersRef(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "10.10.21.81"},
				},
				Infrastructure: &nextdnsv1alpha1.GatewayInfrastructure{
					ParametersRef: &nextdnsv1alpha1.GatewayParametersReference{
						Group: "gateway.envoyproxy.io",
						Kind:  "EnvoyProxy",
						Name:  "custom-proxy-config",
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway",
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, gw)
	require.NoError(t, err)

	require.NotNil(t, gw.Spec.Infrastructure)
	assert.Nil(t, gw.Spec.Infrastructure.Annotations)
	assert.Nil(t, gw.Spec.Infrastructure.Labels)
	require.NotNil(t, gw.Spec.Infrastructure.ParametersRef)
	assert.Equal(t, gatewayv1.Group("gateway.envoyproxy.io"), gw.Spec.Infrastructure.ParametersRef.Group)
	assert.Equal(t, gatewayv1.Kind("EnvoyProxy"), gw.Spec.Infrastructure.ParametersRef.Kind)
	assert.Equal(t, "custom-proxy-config", gw.Spec.Infrastructure.ParametersRef.Name)
}

func TestReconcileGateway_InfrastructureAllFields(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "10.10.21.81"},
				},
				Infrastructure: &nextdnsv1alpha1.GatewayInfrastructure{
					Annotations: map[string]string{
						"lbipam.cilium.io/ips": "10.10.21.81",
					},
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "nextdns-operator",
					},
					ParametersRef: &nextdnsv1alpha1.GatewayParametersReference{
						Group: "gateway.envoyproxy.io",
						Kind:  "EnvoyProxy",
						Name:  "custom-proxy-config",
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway",
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, gw)
	require.NoError(t, err)

	require.NotNil(t, gw.Spec.Infrastructure)

	// Annotations
	assert.Equal(t, gatewayv1.AnnotationValue("10.10.21.81"), gw.Spec.Infrastructure.Annotations[gatewayv1.AnnotationKey("lbipam.cilium.io/ips")])

	// Labels
	assert.Equal(t, gatewayv1.LabelValue("nextdns-operator"), gw.Spec.Infrastructure.Labels[gatewayv1.LabelKey("app.kubernetes.io/managed-by")])

	// ParametersRef
	require.NotNil(t, gw.Spec.Infrastructure.ParametersRef)
	assert.Equal(t, gatewayv1.Group("gateway.envoyproxy.io"), gw.Spec.Infrastructure.ParametersRef.Group)
	assert.Equal(t, gatewayv1.Kind("EnvoyProxy"), gw.Spec.Infrastructure.ParametersRef.Kind)
	assert.Equal(t, "custom-proxy-config", gw.Spec.Infrastructure.ParametersRef.Name)
}

func TestReconcileGateway_InfrastructureNil(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "10.10.21.81"},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway",
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, gw)
	require.NoError(t, err)

	assert.Nil(t, gw.Spec.Infrastructure)
}

func TestReconcileGateway_InfrastructureEmpty(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "10.10.21.81"},
				},
				Infrastructure: &nextdnsv1alpha1.GatewayInfrastructure{},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway",
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, gw)
	require.NoError(t, err)

	// Empty infrastructure should not set spec.infrastructure
	assert.Nil(t, gw.Spec.Infrastructure)
}

func TestReconcileTCPRoute(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	err := reconciler.reconcileTCPRoute(ctx, coreDNS, "test-coredns-dns-svc")
	require.NoError(t, err)

	// Fetch the created TCPRoute
	route := &gatewayv1alpha2.TCPRoute{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-coredns-dns-tcp",
		Namespace: "default",
	}, route)
	require.NoError(t, err)

	// Verify name
	assert.Equal(t, "test-coredns-dns-tcp", route.Name)
	assert.Equal(t, "default", route.Namespace)

	// Verify parentRef
	require.Len(t, route.Spec.ParentRefs, 1)
	parentRef := route.Spec.ParentRefs[0]
	assert.Equal(t, gatewayv1.ObjectName("test-coredns-dns"), parentRef.Name)
	require.NotNil(t, parentRef.SectionName)
	assert.Equal(t, gatewayv1.SectionName("dns-tcp"), *parentRef.SectionName)

	// Verify rules and backendRef
	require.Len(t, route.Spec.Rules, 1)
	require.Len(t, route.Spec.Rules[0].BackendRefs, 1)
	backendRef := route.Spec.Rules[0].BackendRefs[0]
	assert.Equal(t, gatewayv1.ObjectName("test-coredns-dns-svc"), backendRef.Name)
	require.NotNil(t, backendRef.Port)
	assert.Equal(t, gatewayv1.PortNumber(53), *backendRef.Port)

	// Verify owner reference
	require.Len(t, route.OwnerReferences, 1)
	assert.Equal(t, "test-coredns", route.OwnerReferences[0].Name)
	assert.Equal(t, "NextDNSCoreDNS", route.OwnerReferences[0].Kind)
}

func TestUpdateGatewayStatus(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "192.168.1.53"},
				},
			},
		},
	}

	// Create a Gateway with status addresses
	addrType := gatewayv1.IPAddressType
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns-dns",
			Namespace: "default",
		},
		Status: gatewayv1.GatewayStatus{
			Addresses: []gatewayv1.GatewayStatusAddress{
				{Type: &addrType, Value: "192.168.1.53"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS, gw).
		Build()

	r := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	r.updateGatewayStatus(ctx, coreDNS)

	assert.Equal(t, "192.168.1.53", coreDNS.Status.DNSIP)
	assert.True(t, coreDNS.Status.GatewayReady)
	require.Len(t, coreDNS.Status.Endpoints, 2)
	assert.Equal(t, "UDP", coreDNS.Status.Endpoints[0].Protocol)
	assert.Equal(t, "TCP", coreDNS.Status.Endpoints[1].Protocol)
}

func TestUpdateGatewayStatus_FallbackToRequestedAddresses(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "10.0.0.53"},
				},
			},
		},
	}

	// Gateway exists but has no status addresses yet
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns-dns",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS, gw).
		Build()

	r := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	r.updateGatewayStatus(ctx, coreDNS)

	assert.Equal(t, "10.0.0.53", coreDNS.Status.DNSIP)
	assert.False(t, coreDNS.Status.GatewayReady) // No status addresses = not ready
}

func TestReconcileUDPRoute(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	err := reconciler.reconcileUDPRoute(ctx, coreDNS, "test-coredns-dns-svc")
	require.NoError(t, err)

	// Fetch the created UDPRoute
	route := &gatewayv1alpha2.UDPRoute{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-coredns-dns-udp",
		Namespace: "default",
	}, route)
	require.NoError(t, err)

	// Verify name
	assert.Equal(t, "test-coredns-dns-udp", route.Name)
	assert.Equal(t, "default", route.Namespace)

	// Verify parentRef
	require.Len(t, route.Spec.ParentRefs, 1)
	parentRef := route.Spec.ParentRefs[0]
	assert.Equal(t, gatewayv1.ObjectName("test-coredns-dns"), parentRef.Name)
	require.NotNil(t, parentRef.SectionName)
	assert.Equal(t, gatewayv1.SectionName("dns-udp"), *parentRef.SectionName)

	// Verify rules and backendRef
	require.Len(t, route.Spec.Rules, 1)
	require.Len(t, route.Spec.Rules[0].BackendRefs, 1)
	backendRef := route.Spec.Rules[0].BackendRefs[0]
	assert.Equal(t, gatewayv1.ObjectName("test-coredns-dns-svc"), backendRef.Name)
	require.NotNil(t, backendRef.Port)
	assert.Equal(t, gatewayv1.PortNumber(53), *backendRef.Port)

	// Verify owner reference
	require.Len(t, route.OwnerReferences, 1)
	assert.Equal(t, "test-coredns", route.OwnerReferences[0].Name)
	assert.Equal(t, "NextDNSCoreDNS", route.OwnerReferences[0].Kind)
}

func TestCleanupGatewayResources(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
		},
	}

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns-dns",
			Namespace: "default",
		},
	}
	tcpRoute := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns-dns-tcp",
			Namespace: "default",
		},
	}
	udpRoute := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns-dns-udp",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS, gw, tcpRoute, udpRoute).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	err := reconciler.cleanupGatewayResources(ctx, coreDNS)
	require.NoError(t, err)

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, &gatewayv1.Gateway{})
	assert.True(t, apierrors.IsNotFound(err), "Gateway should be deleted")

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns-tcp", Namespace: "default"}, &gatewayv1alpha2.TCPRoute{})
	assert.True(t, apierrors.IsNotFound(err), "TCPRoute should be deleted")

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns-udp", Namespace: "default"}, &gatewayv1alpha2.UDPRoute{})
	assert.True(t, apierrors.IsNotFound(err), "UDPRoute should be deleted")
}

func TestCleanupGatewayResources_AlreadyGone(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	err := reconciler.cleanupGatewayResources(ctx, coreDNS)
	require.NoError(t, err)
}
