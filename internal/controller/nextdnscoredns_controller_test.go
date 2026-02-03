package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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
		name       string
		profile    *nextdnsv1alpha1.NextDNSProfile
		wantReady  bool
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
