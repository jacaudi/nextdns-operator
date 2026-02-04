package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

// newCoreDNSTestScheme creates a scheme for CoreDNS controller tests
// which includes appsv1 for Deployment/DaemonSet operations
func newCoreDNSTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nextdnsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	return scheme
}

func TestNextDNSCoreDNSReconciler_ResolveProfile(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a ready profile
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
			Conditions: []metav1.Condition{
				{
					Type:   ConditionTypeReady,
					Status: metav1.ConditionTrue,
					Reason: "Ready",
				},
			},
		},
	}

	// Create a CoreDNS instance referencing the profile
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		Build()

	r := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Test resolveProfile
	resolvedProfile, err := r.resolveProfile(ctx, coreDNS)
	require.NoError(t, err)
	assert.Equal(t, "test-profile", resolvedProfile.Name)
	assert.Equal(t, "abc123", resolvedProfile.Status.ProfileID)
}

func TestNextDNSCoreDNSReconciler_ResolveProfile_NotFound(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a CoreDNS instance referencing a non-existent profile
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "missing-profile",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	r := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Test resolveProfile with missing profile
	resolvedProfile, err := r.resolveProfile(ctx, coreDNS)
	assert.Error(t, err)
	assert.Nil(t, resolvedProfile)
	assert.Contains(t, err.Error(), "failed to get NextDNSProfile")
}

func TestNextDNSCoreDNSReconciler_ResolveProfile_NotReady(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a profile without Ready condition or ProfileID
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unready-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Unready Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			// No ProfileID, no Ready condition
			Conditions: []metav1.Condition{
				{
					Type:   ConditionTypeReady,
					Status: metav1.ConditionFalse,
					Reason: "Syncing",
				},
			},
		},
	}

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "unready-profile",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		Build()

	r := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// resolveProfile should succeed (profile exists)
	resolvedProfile, err := r.resolveProfile(ctx, coreDNS)
	require.NoError(t, err)
	assert.NotNil(t, resolvedProfile)

	// But isProfileReady should return false
	isReady := r.isProfileReady(resolvedProfile)
	assert.False(t, isReady, "Profile without Ready=True condition should not be ready")
}

func TestNextDNSCoreDNSReconciler_ResolveProfile_CrossNamespace(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a profile in a different namespace
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-profile",
			Namespace: "shared",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Shared Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "shared123",
			Conditions: []metav1.Condition{
				{
					Type:   ConditionTypeReady,
					Status: metav1.ConditionTrue,
					Reason: "Ready",
				},
			},
		},
	}

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name:      "shared-profile",
				Namespace: "shared",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		Build()

	r := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	resolvedProfile, err := r.resolveProfile(ctx, coreDNS)
	require.NoError(t, err)
	assert.Equal(t, "shared-profile", resolvedProfile.Name)
	assert.Equal(t, "shared", resolvedProfile.Namespace)
	assert.Equal(t, "shared123", resolvedProfile.Status.ProfileID)
}

func TestNextDNSCoreDNSReconciler_GetResourceName(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	tests := []struct {
		name           string
		coreDNSName    string
		profileID      string
		expectedResult string
	}{
		{
			name:           "standard names",
			coreDNSName:    "my-dns",
			profileID:      "abc123",
			expectedResult: "my-dns-abc123-coredns",
		},
		{
			name:           "long profile ID",
			coreDNSName:    "coredns",
			profileID:      "longprofileid12345",
			expectedResult: "coredns-longprofileid12345-coredns",
		},
		{
			name:           "short names",
			coreDNSName:    "dns",
			profileID:      "x",
			expectedResult: "dns-x-coredns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.coreDNSName,
					Namespace: "default",
				},
			}

			profile := &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					ProfileID: tt.profileID,
				},
			}

			result := r.getResourceName(coreDNS, profile)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestNextDNSCoreDNSReconciler_GetServiceName_Default(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	// CoreDNS without service name override
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			// No Service config, no override
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "abc123",
		},
	}

	// Default service name should follow resource name pattern
	serviceName := r.getServiceName(coreDNS, profile)
	assert.Equal(t, "my-coredns-abc123-coredns", serviceName)
}

func TestNextDNSCoreDNSReconciler_GetServiceName_Override(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	// CoreDNS with service name override
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			Service: &nextdnsv1alpha1.CoreDNSServiceConfig{
				NameOverride: "custom-dns-service",
			},
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "abc123",
		},
	}

	// Service name should be the override
	serviceName := r.getServiceName(coreDNS, profile)
	assert.Equal(t, "custom-dns-service", serviceName)
}

func TestNextDNSCoreDNSReconciler_GetServiceName_EmptyOverride(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	// CoreDNS with empty service name override (should use default)
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			Service: &nextdnsv1alpha1.CoreDNSServiceConfig{
				NameOverride: "", // Empty string
			},
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "xyz789",
		},
	}

	// Empty override should fall back to default
	serviceName := r.getServiceName(coreDNS, profile)
	assert.Equal(t, "my-coredns-xyz789-coredns", serviceName)
}

func TestNextDNSCoreDNSReconciler_BuildLabels(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "abc123",
		},
	}

	labels := r.buildLabels(coreDNS, profile)

	// Check all standard labels are present and correct
	assert.Equal(t, "coredns", labels["app.kubernetes.io/name"])
	assert.Equal(t, "test-coredns", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "dns", labels["app.kubernetes.io/component"])
	assert.Equal(t, "nextdns-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "abc123", labels["nextdns.io/profile-id"])

	// Verify total number of labels
	assert.Len(t, labels, 5)
}

func TestNextDNSCoreDNSReconciler_IsProfileReady(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	tests := []struct {
		name      string
		profile   *nextdnsv1alpha1.NextDNSProfile
		wantReady bool
	}{
		{
			name: "profile with Ready=True",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					ProfileID: "abc123",
					Conditions: []metav1.Condition{
						{
							Type:   ConditionTypeReady,
							Status: metav1.ConditionTrue,
							Reason: "Ready",
						},
					},
				},
			},
			wantReady: true,
		},
		{
			name: "profile with Ready=False",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					Conditions: []metav1.Condition{
						{
							Type:   ConditionTypeReady,
							Status: metav1.ConditionFalse,
							Reason: "Syncing",
						},
					},
				},
			},
			wantReady: false,
		},
		{
			name: "profile with Ready=Unknown",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					Conditions: []metav1.Condition{
						{
							Type:   ConditionTypeReady,
							Status: metav1.ConditionUnknown,
							Reason: "Initializing",
						},
					},
				},
			},
			wantReady: false,
		},
		{
			name: "profile without Ready condition",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					ProfileID: "abc123",
					Conditions: []metav1.Condition{
						{
							Type:   ConditionTypeSynced,
							Status: metav1.ConditionTrue,
							Reason: "Synced",
						},
					},
				},
			},
			wantReady: false,
		},
		{
			name: "profile with no conditions",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					ProfileID:  "abc123",
					Conditions: []metav1.Condition{},
				},
			},
			wantReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isReady := r.isProfileReady(tt.profile)
			assert.Equal(t, tt.wantReady, isReady)
		})
	}
}

func TestNextDNSCoreDNSReconciler_Constants(t *testing.T) {
	// Verify important constants are defined correctly
	assert.Equal(t, "nextdns.io/coredns-finalizer", CoreDNSFinalizerName)
	assert.Equal(t, "ProfileResolved", ConditionTypeProfileResolved)
	assert.Equal(t, "Corefile", CorefileKey)
}

func TestNextDNSCoreDNSReconciler_Reconcile_HappyPath(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a ready NextDNSProfile with ProfileID "abc123"
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// Create a NextDNSCoreDNS with profileRef, upstream DoT, and 2 replicas
	replicas := int32(2)
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
			Upstream: &nextdnsv1alpha1.UpstreamConfig{
				Primary: nextdnsv1alpha1.DNSProtocolDoT,
			},
			Deployment: &nextdnsv1alpha1.CoreDNSDeploymentConfig{
				Replicas: &replicas,
			},
		},
	}

	// Create fake client with status subresource support
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		WithStatusSubresource(profile, coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	// First reconcile - should add finalizer
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.Greater(t, result.RequeueAfter, time.Duration(0), "Should requeue after adding finalizer")

	// Verify finalizer was added
	updatedCoreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{}
	err = fakeClient.Get(ctx, req.NamespacedName, updatedCoreDNS)
	require.NoError(t, err)
	assert.Contains(t, updatedCoreDNS.Finalizers, CoreDNSFinalizerName, "Finalizer should be added")

	// Second reconcile - should create ConfigMap, Deployment, Service
	result, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Expected resource name: test-coredns-abc123-coredns
	resourceName := "test-coredns-abc123-coredns"

	// Verify ConfigMap was created with "forward" in Corefile
	configMap := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}, configMap)
	require.NoError(t, err, "ConfigMap should be created")
	assert.Contains(t, configMap.Data[CorefileKey], "forward", "Corefile should contain forward directive")

	// Verify Deployment was created with 2 replicas
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}, deployment)
	require.NoError(t, err, "Deployment should be created")
	require.NotNil(t, deployment.Spec.Replicas, "Replicas should be set")
	assert.Equal(t, int32(2), *deployment.Spec.Replicas, "Deployment should have 2 replicas")

	// Verify Service was created with ClusterIP type
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}, service)
	require.NoError(t, err, "Service should be created")
	assert.Equal(t, corev1.ServiceTypeClusterIP, service.Spec.Type, "Service should be ClusterIP type")
}

func TestNextDNSCoreDNSReconciler_Reconcile_DaemonSetMode(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a ready NextDNSProfile with ProfileID "abc123"
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// Create a NextDNSCoreDNS with profileRef and DaemonSet mode
	// Include finalizer to skip the first reconcile that adds it
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-coredns",
			Namespace:  "default",
			Finalizers: []string{CoreDNSFinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
			Deployment: &nextdnsv1alpha1.CoreDNSDeploymentConfig{
				Mode: nextdnsv1alpha1.DeploymentModeDaemonSet,
			},
		},
	}

	// Create fake client with status subresource support
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		WithStatusSubresource(profile, coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	// Run reconcile - should create DaemonSet (finalizer already present)
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_ = result // Result may vary depending on implementation

	// Expected resource name: test-coredns-abc123-coredns
	resourceName := "test-coredns-abc123-coredns"

	// Verify DaemonSet was created with correct labels
	daemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}, daemonSet)
	require.NoError(t, err, "DaemonSet should be created")

	// Verify DaemonSet labels
	assert.Equal(t, "coredns", daemonSet.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "test-coredns", daemonSet.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, "dns", daemonSet.Labels["app.kubernetes.io/component"])
	assert.Equal(t, "nextdns-operator", daemonSet.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "abc123", daemonSet.Labels["nextdns.io/profile-id"])

	// Verify NO Deployment exists (should error on Get)
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}, deployment)
	assert.Error(t, err, "Deployment should NOT exist when mode is DaemonSet")
}

func TestNextDNSCoreDNSReconciler_Reconcile_ProfileNotReady(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a NextDNSProfile with Ready=False condition (no ProfileID yet)
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			// No ProfileID - profile is not ready yet
			Conditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "Syncing",
					Message:            "Profile is syncing with NextDNS API",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// Create a NextDNSCoreDNS with profileRef and finalizer already added
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-coredns",
			Namespace:  "default",
			Finalizers: []string{CoreDNSFinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
		},
	}

	// Create fake client with status subresource support
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		WithStatusSubresource(profile, coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	// Run reconcile - should wait for profile to become ready
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify result.RequeueAfter > 0 (should requeue to wait for profile)
	assert.Greater(t, result.RequeueAfter, time.Duration(0), "Should requeue to wait for profile to become ready")

	// Fetch the updated NextDNSCoreDNS to check conditions
	updatedCoreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{}
	err = fakeClient.Get(ctx, req.NamespacedName, updatedCoreDNS)
	require.NoError(t, err)

	// Verify ProfileResolved condition is False with reason "ProfileNotReady"
	profileResolvedCondition := meta.FindStatusCondition(updatedCoreDNS.Status.Conditions, ConditionTypeProfileResolved)
	require.NotNil(t, profileResolvedCondition, "ProfileResolved condition should exist")
	assert.Equal(t, metav1.ConditionFalse, profileResolvedCondition.Status, "ProfileResolved should be False")
	assert.Equal(t, "ProfileNotReady", profileResolvedCondition.Reason, "ProfileResolved reason should be ProfileNotReady")

	// Verify Ready condition is also False
	readyCondition := meta.FindStatusCondition(updatedCoreDNS.Status.Conditions, ConditionTypeReady)
	require.NotNil(t, readyCondition, "Ready condition should exist")
	assert.Equal(t, metav1.ConditionFalse, readyCondition.Status, "Ready should be False")

	// Verify Status.Ready is false
	assert.False(t, updatedCoreDNS.Status.Ready, "Status.Ready should be false")
}

func TestNextDNSCoreDNSReconciler_HandleDeletion(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a ready NextDNSProfile (needed to compute resource names for the Deployment)
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// Create a NextDNSCoreDNS with:
	// - DeletionTimestamp set (marks it for deletion)
	// - Finalizer present (to be removed by handleDeletion)
	// - profileRef to "test-profile"
	deletionTime := metav1.Now()
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-coredns",
			Namespace:         "default",
			Finalizers:        []string{CoreDNSFinalizerName},
			DeletionTimestamp: &deletionTime,
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
		},
	}

	// Pre-create a Deployment that would have been created by the controller
	// In real scenarios, OwnerReferences would handle cleanup, but we verify the flow
	resourceName := "test-coredns-abc123-coredns"
	labels := map[string]string{
		"app.kubernetes.io/name":       "coredns",
		"app.kubernetes.io/instance":   "test-coredns",
		"app.kubernetes.io/component":  "dns",
		"app.kubernetes.io/managed-by": "nextdns-operator",
		"nextdns.io/profile-id":        "abc123",
	}
	replicas := int32(2)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: "default",
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "nextdns.io/v1alpha1",
					Kind:       "NextDNSCoreDNS",
					Name:       "test-coredns",
					UID:        coreDNS.UID,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "coredns",
							Image: "coredns/coredns:1.11.1",
						},
					},
				},
			},
		},
	}

	// Create fake client with status subresource support
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS, deployment).
		WithStatusSubresource(profile, coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	// Run reconcile - should handle deletion and remove finalizer
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verify result is empty (no requeue needed after deletion handling)
	assert.Equal(t, ctrl.Result{}, result, "Result should be empty after deletion handling")

	// Verify the resource is deleted (Get should fail with NotFound)
	// When the finalizer is removed from a resource with DeletionTimestamp,
	// the fake client simulates the real Kubernetes behavior and deletes the resource.
	updatedCoreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{}
	err = fakeClient.Get(ctx, req.NamespacedName, updatedCoreDNS)
	assert.True(t, apierrors.IsNotFound(err), "Resource should be deleted after finalizer removal, got error: %v", err)
}

func TestNextDNSCoreDNSReconciler_Reconcile_LoadBalancerService(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a ready NextDNSProfile
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// Create a NextDNSCoreDNS with LoadBalancer service type, loadBalancerIP, and annotations
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-coredns",
			Namespace:  "default",
			Finalizers: []string{CoreDNSFinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
			Service: &nextdnsv1alpha1.CoreDNSServiceConfig{
				Type:           nextdnsv1alpha1.ServiceTypeLoadBalancer,
				LoadBalancerIP: "192.168.1.53",
				Annotations: map[string]string{
					"metallb.universe.tf/address-pool": "dns-pool",
					"external-dns.alpha.kubernetes.io/hostname": "dns.example.com",
				},
			},
		},
	}

	// Create fake client with status subresource support
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		WithStatusSubresource(profile, coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	// Run reconcile - finalizer already present, should create resources
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_ = result

	// Expected resource name: test-coredns-abc123-coredns
	resourceName := "test-coredns-abc123-coredns"

	// Verify Service was created with LoadBalancer type
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}, service)
	require.NoError(t, err, "Service should be created")

	// Verify service type is LoadBalancer
	assert.Equal(t, corev1.ServiceTypeLoadBalancer, service.Spec.Type, "Service should be LoadBalancer type")

	// Verify loadBalancerIP is set
	assert.Equal(t, "192.168.1.53", service.Spec.LoadBalancerIP, "Service should have loadBalancerIP set")

	// Verify annotations are present
	require.NotNil(t, service.Annotations, "Service should have annotations")
	assert.Equal(t, "dns-pool", service.Annotations["metallb.universe.tf/address-pool"], "MetalLB annotation should be present")
	assert.Equal(t, "dns.example.com", service.Annotations["external-dns.alpha.kubernetes.io/hostname"], "External DNS annotation should be present")
}

func TestNextDNSCoreDNSReconciler_Reconcile_NodePortService(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Create a ready NextDNSProfile
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// Create a NextDNSCoreDNS with NodePort service type
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-coredns",
			Namespace:  "default",
			Finalizers: []string{CoreDNSFinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
			Service: &nextdnsv1alpha1.CoreDNSServiceConfig{
				Type: nextdnsv1alpha1.ServiceTypeNodePort,
			},
		},
	}

	// Create fake client with status subresource support
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		WithStatusSubresource(profile, coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}

	// Run reconcile - finalizer already present, should create resources
	result, err := reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	_ = result

	// Expected resource name: test-coredns-abc123-coredns
	resourceName := "test-coredns-abc123-coredns"

	// Verify Service was created with NodePort type
	service := &corev1.Service{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}, service)
	require.NoError(t, err, "Service should be created")

	// Verify service type is NodePort
	assert.Equal(t, corev1.ServiceTypeNodePort, service.Spec.Type, "Service should be NodePort type")
}

func TestNextDNSCoreDNSReconciler_BuildCorefileConfig(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	// Helper to create a pointer to DNSProtocol
	protocolPtr := func(p nextdnsv1alpha1.DNSProtocol) *nextdnsv1alpha1.DNSProtocol {
		return &p
	}

	// Helper to create a pointer to bool
	boolPtr := func(b bool) *bool {
		return &b
	}

	// Helper to create a pointer to int32
	int32Ptr := func(i int32) *int32 {
		return &i
	}

	tests := []struct {
		name             string
		coreDNS          *nextdnsv1alpha1.NextDNSCoreDNS
		profile          *nextdnsv1alpha1.NextDNSProfile
		wantProfileID    string
		wantPrimary      string
		wantFallback     string
		wantCacheTTL     int32
		wantLogging      bool
		wantMetrics      bool
	}{
		{
			name: "DoT primary with DoH fallback",
			coreDNS: &nextdnsv1alpha1.NextDNSCoreDNS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-coredns",
					Namespace: "default",
				},
				Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
					Upstream: &nextdnsv1alpha1.UpstreamConfig{
						Primary:  nextdnsv1alpha1.DNSProtocolDoT,
						Fallback: protocolPtr(nextdnsv1alpha1.DNSProtocolDoH),
					},
					Cache: &nextdnsv1alpha1.CoreDNSCacheConfig{
						SuccessTTL: int32Ptr(60),
					},
					Logging: &nextdnsv1alpha1.CoreDNSLoggingConfig{
						Enabled: boolPtr(true),
					},
					Metrics: &nextdnsv1alpha1.CoreDNSMetricsConfig{
						Enabled: boolPtr(true),
					},
				},
			},
			profile: &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					ProfileID: "abc123",
				},
			},
			wantProfileID: "abc123",
			wantPrimary:   "DoT",
			wantFallback:  "DoH",
			wantCacheTTL:  60,
			wantLogging:   true,
			wantMetrics:   true,
		},
		{
			name: "DNS primary only",
			coreDNS: &nextdnsv1alpha1.NextDNSCoreDNS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-coredns",
					Namespace: "default",
				},
				Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
					Upstream: &nextdnsv1alpha1.UpstreamConfig{
						Primary: nextdnsv1alpha1.DNSProtocolDNS,
						// No fallback
					},
					// Use defaults for cache, logging, metrics
				},
			},
			profile: &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					ProfileID: "def456",
				},
			},
			wantProfileID: "def456",
			wantPrimary:   "DNS",
			wantFallback:  "", // No fallback
			wantCacheTTL:  3600, // Default
			wantLogging:   false, // Default
			wantMetrics:   true, // Default
		},
		{
			name: "defaults when spec is minimal",
			coreDNS: &nextdnsv1alpha1.NextDNSCoreDNS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-coredns",
					Namespace: "default",
				},
				Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
					// Empty spec - all defaults
				},
			},
			profile: &nextdnsv1alpha1.NextDNSProfile{
				Status: nextdnsv1alpha1.NextDNSProfileStatus{
					ProfileID: "ghi789",
				},
			},
			wantProfileID: "ghi789",
			wantPrimary:   "DoT", // Default
			wantFallback:  "", // No fallback by default
			wantCacheTTL:  3600, // Default
			wantLogging:   false, // Default
			wantMetrics:   true, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := r.buildCorefileConfig(tt.coreDNS, tt.profile)

			assert.Equal(t, tt.wantProfileID, cfg.ProfileID, "ProfileID should match")
			assert.Equal(t, tt.wantPrimary, cfg.PrimaryProtocol, "PrimaryProtocol should match")
			assert.Equal(t, tt.wantFallback, cfg.FallbackProtocol, "FallbackProtocol should match")
			assert.Equal(t, tt.wantCacheTTL, cfg.CacheTTL, "CacheTTL should match")
			assert.Equal(t, tt.wantLogging, cfg.LoggingEnabled, "LoggingEnabled should match")
			assert.Equal(t, tt.wantMetrics, cfg.MetricsEnabled, "MetricsEnabled should match")
		})
	}
}

func TestNextDNSCoreDNSReconciler_BuildPodSpec(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	// Create a NextDNSCoreDNS with all placement and resource settings
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
			Deployment: &nextdnsv1alpha1.CoreDNSDeploymentConfig{
				Image: "custom-coredns:v1.0.0",
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
				NodeSelector: map[string]string{
					"kubernetes.io/os": "linux",
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "node-role.kubernetes.io/control-plane",
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
					},
				},
			},
		},
	}

	configMapName := "test-coredns-abc123-coredns"
	podSpec := r.buildPodSpec(coreDNS, configMapName)

	// Verify container image matches custom image
	require.Len(t, podSpec.Containers, 1, "Should have exactly one container")
	assert.Equal(t, "custom-coredns:v1.0.0", podSpec.Containers[0].Image, "Container image should match custom image")
	assert.Equal(t, "coredns", podSpec.Containers[0].Name, "Container name should be coredns")

	// Verify resource requests are set correctly
	cpuRequest := podSpec.Containers[0].Resources.Requests[corev1.ResourceCPU]
	assert.Equal(t, "100m", cpuRequest.String(), "CPU request should be 100m")

	memRequest := podSpec.Containers[0].Resources.Requests[corev1.ResourceMemory]
	assert.Equal(t, "128Mi", memRequest.String(), "Memory request should be 128Mi")

	// Verify resource limits are set correctly
	memLimit := podSpec.Containers[0].Resources.Limits[corev1.ResourceMemory]
	assert.Equal(t, "256Mi", memLimit.String(), "Memory limit should be 256Mi")

	// Verify NodeSelector is applied
	require.NotNil(t, podSpec.NodeSelector, "NodeSelector should be set")
	assert.Equal(t, "linux", podSpec.NodeSelector["kubernetes.io/os"], "NodeSelector should have kubernetes.io/os=linux")

	// Verify Tolerations are present
	require.Len(t, podSpec.Tolerations, 1, "Should have exactly one toleration")
	assert.Equal(t, "node-role.kubernetes.io/control-plane", podSpec.Tolerations[0].Key, "Toleration key should match")
	assert.Equal(t, corev1.TolerationOpExists, podSpec.Tolerations[0].Operator, "Toleration operator should be Exists")
	assert.Equal(t, corev1.TaintEffectNoSchedule, podSpec.Tolerations[0].Effect, "Toleration effect should be NoSchedule")

	// Verify volume mount for /etc/coredns exists
	require.Len(t, podSpec.Containers[0].VolumeMounts, 1, "Should have exactly one volume mount")
	assert.Equal(t, "/etc/coredns", podSpec.Containers[0].VolumeMounts[0].MountPath, "Volume mount path should be /etc/coredns")
	assert.True(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly, "Volume mount should be read-only")

	// Verify volume is configured correctly
	require.Len(t, podSpec.Volumes, 1, "Should have exactly one volume")
	assert.Equal(t, "config-volume", podSpec.Volumes[0].Name, "Volume name should be config-volume")
	require.NotNil(t, podSpec.Volumes[0].ConfigMap, "Volume should be a ConfigMap volume")
	assert.Equal(t, configMapName, podSpec.Volumes[0].ConfigMap.Name, "ConfigMap name should match")

	// Verify security context
	require.NotNil(t, podSpec.Containers[0].SecurityContext, "Container security context should be set")
	assert.False(t, *podSpec.Containers[0].SecurityContext.AllowPrivilegeEscalation, "AllowPrivilegeEscalation should be false")
	assert.True(t, *podSpec.Containers[0].SecurityContext.ReadOnlyRootFilesystem, "ReadOnlyRootFilesystem should be true")

	// Verify pod-level security context
	require.NotNil(t, podSpec.SecurityContext, "Pod security context should be set")
	assert.True(t, *podSpec.SecurityContext.RunAsNonRoot, "RunAsNonRoot should be true")
}

func TestNextDNSCoreDNSReconciler_BuildPodSpec_DefaultImage(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	// Create a NextDNSCoreDNS without custom image (should use default)
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
			// No Deployment spec - use defaults
		},
	}

	configMapName := "test-coredns-abc123-coredns"
	podSpec := r.buildPodSpec(coreDNS, configMapName)

	// Verify default image is used
	require.Len(t, podSpec.Containers, 1, "Should have exactly one container")
	assert.Equal(t, "registry.k8s.io/coredns/coredns:1.11.1", podSpec.Containers[0].Image, "Container image should be default coredns image")

	// Verify no custom NodeSelector, Tolerations, or Resources
	assert.Nil(t, podSpec.NodeSelector, "NodeSelector should be nil when not specified")
	assert.Nil(t, podSpec.Tolerations, "Tolerations should be nil when not specified")
	assert.Empty(t, podSpec.Containers[0].Resources.Requests, "Resource requests should be empty when not specified")
	assert.Empty(t, podSpec.Containers[0].Resources.Limits, "Resource limits should be empty when not specified")
}

func TestNextDNSCoreDNSReconciler_BuildPodSpec_NoHardcodedServiceAccount(t *testing.T) {
	scheme := newCoreDNSTestScheme()

	r := &NextDNSCoreDNSReconciler{
		Scheme: scheme,
	}

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
		},
	}

	configMapName := "test-coredns-abc123-coredns"
	podSpec := r.buildPodSpec(coreDNS, configMapName)

	// ServiceAccountName should be empty (use namespace default)
	assert.Empty(t, podSpec.ServiceAccountName, "ServiceAccountName should be empty to use namespace default")
}

func TestNextDNSCoreDNSReconciler_UpdateStatus(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// 1. Create a ready NextDNSProfile with ProfileID "abc123"
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
			Conditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Ready",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// 2. Create a NextDNSCoreDNS with profileRef and upstream.primary="DoT"
	// 3. Add finalizer to skip first reconcile phase
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-coredns",
			Namespace:  "default",
			Finalizers: []string{CoreDNSFinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
			Upstream: &nextdnsv1alpha1.UpstreamConfig{
				Primary: nextdnsv1alpha1.DNSProtocolDoT,
			},
		},
	}

	// Pre-create Service with ClusterIP already assigned (simulating Kubernetes behavior)
	// The reconciler's reconcileService will update labels/ports but preserve ClusterIP
	resourceName := "test-coredns-abc123-coredns"
	labels := map[string]string{
		"app.kubernetes.io/name":       "coredns",
		"app.kubernetes.io/instance":   "test-coredns",
		"app.kubernetes.io/component":  "dns",
		"app.kubernetes.io/managed-by": "nextdns-operator",
		"nextdns.io/profile-id":        "abc123",
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: "default",
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.53",
			Selector:  labels,
			Ports: []corev1.ServicePort{
				{Name: "dns", Port: 53, Protocol: corev1.ProtocolUDP},
				{Name: "dns-tcp", Port: 53, Protocol: corev1.ProtocolTCP},
				{Name: "metrics", Port: 9153, Protocol: corev1.ProtocolTCP},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS, service).
		WithStatusSubresource(profile, coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Call updateStatus directly to test status update logic
	// First set ProfileResolved condition (normally done in Reconcile before updateStatus)
	reconciler.setCondition(coreDNS, ConditionTypeProfileResolved, metav1.ConditionTrue, "ProfileResolved", "Referenced profile found and ready")
	coreDNS.Status.ProfileID = profile.Status.ProfileID
	coreDNS.Status.Fingerprint = profile.Status.Fingerprint

	// 4. Run updateStatus
	err := reconciler.updateStatus(ctx, coreDNS, profile)
	require.NoError(t, err)

	// Fetch updated NextDNSCoreDNS
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-coredns",
			Namespace: "default",
		},
	}
	updatedCoreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{}
	err = fakeClient.Get(ctx, req.NamespacedName, updatedCoreDNS)
	require.NoError(t, err)

	// 5. Verify status fields
	// - ProfileID equals "abc123"
	assert.Equal(t, "abc123", updatedCoreDNS.Status.ProfileID, "ProfileID should be set from profile")

	// - UpstreamEndpoint contains "tls://" (for DoT)
	require.NotNil(t, updatedCoreDNS.Status.Upstream, "Upstream status should be set")
	assert.Contains(t, updatedCoreDNS.Status.Upstream.URL, "tls://", "UpstreamEndpoint should contain tls:// for DoT protocol")

	// - DNSEndpoint is not empty
	assert.NotEmpty(t, updatedCoreDNS.Status.Endpoints, "DNSEndpoints should not be empty")

	// Verify DNS IP is set
	assert.Equal(t, "10.96.0.53", updatedCoreDNS.Status.DNSIP, "DNSIP should be set from Service ClusterIP")

	// 6. Verify conditions
	// - ProfileResolved condition is True
	profileResolvedCondition := meta.FindStatusCondition(updatedCoreDNS.Status.Conditions, ConditionTypeProfileResolved)
	require.NotNil(t, profileResolvedCondition, "ProfileResolved condition should exist")
	assert.Equal(t, metav1.ConditionTrue, profileResolvedCondition.Status, "ProfileResolved should be True")

	// - Ready condition exists
	readyCondition := meta.FindStatusCondition(updatedCoreDNS.Status.Conditions, ConditionTypeReady)
	assert.NotNil(t, readyCondition, "Ready condition should exist")
}
