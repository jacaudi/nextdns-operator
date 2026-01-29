package controller

import (
	"context"
	"fmt"
	"testing"

	sdknextdns "github.com/jacaudi/nextdns-go/nextdns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
	"github.com/jacaudi/nextdns-operator/internal/nextdns"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nextdnsv1alpha1.AddToScheme(scheme))
	return scheme
}

func TestBoolValue(t *testing.T) {
	tests := []struct {
		name         string
		ptr          *bool
		defaultValue bool
		expected     bool
	}{
		{
			name:         "nil pointer returns default true",
			ptr:          nil,
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "nil pointer returns default false",
			ptr:          nil,
			defaultValue: false,
			expected:     false,
		},
		{
			name:         "true pointer returns true",
			ptr:          boolPtr(true),
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false pointer returns false",
			ptr:          boolPtr(false),
			defaultValue: true,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := boolValue(tt.ptr, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseRetentionDays(t *testing.T) {
	tests := []struct {
		name      string
		retention string
		expected  int
	}{
		{
			name:      "empty string returns default",
			retention: "",
			expected:  7,
		},
		{
			name:      "1h returns 0",
			retention: "1h",
			expected:  0,
		},
		{
			name:      "6h returns 0",
			retention: "6h",
			expected:  0,
		},
		{
			name:      "1d returns 1",
			retention: "1d",
			expected:  1,
		},
		{
			name:      "7d returns 7",
			retention: "7d",
			expected:  7,
		},
		{
			name:      "30d returns 30",
			retention: "30d",
			expected:  30,
		},
		{
			name:      "90d returns 90",
			retention: "90d",
			expected:  90,
		},
		{
			name:      "1y returns 365",
			retention: "1y",
			expected:  365,
		},
		{
			name:      "2y returns 730",
			retention: "2y",
			expected:  730,
		},
		{
			name:      "uppercase 7D returns 7",
			retention: "7D",
			expected:  7,
		},
		{
			name:      "with whitespace",
			retention: "  30d  ",
			expected:  30,
		},
		{
			name:      "invalid string returns default",
			retention: "invalid",
			expected:  7,
		},
		{
			name:      "invalid number returns default",
			retention: "abcd",
			expected:  7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetentionDays(tt.retention)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolvedLists(t *testing.T) {
	resolved := &ResolvedLists{
		Allowlist: []string{"good.com", "allowed.com"},
		Denylist:  []string{"bad.com", "blocked.com"},
		TLDs:      []string{"xyz", "tk"},
		ResourceStatus: &nextdnsv1alpha1.ReferencedResources{
			Allowlists: []nextdnsv1alpha1.ReferencedResourceStatus{
				{Name: "allowlist-1", Namespace: "default", Ready: true, Count: 2},
			},
			Denylists: []nextdnsv1alpha1.ReferencedResourceStatus{
				{Name: "denylist-1", Namespace: "default", Ready: true, Count: 2},
			},
			TLDLists: []nextdnsv1alpha1.ReferencedResourceStatus{
				{Name: "tldlist-1", Namespace: "default", Ready: true, Count: 2},
			},
		},
	}

	assert.Equal(t, 2, len(resolved.Allowlist))
	assert.Equal(t, 2, len(resolved.Denylist))
	assert.Equal(t, 2, len(resolved.TLDs))
	assert.Equal(t, 1, len(resolved.ResourceStatus.Allowlists))
	assert.Equal(t, 1, len(resolved.ResourceStatus.Denylists))
	assert.Equal(t, 1, len(resolved.ResourceStatus.TLDLists))
}

func TestGetAPIKey(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	tests := []struct {
		name        string
		profile     *nextdnsv1alpha1.NextDNSProfile
		secret      *corev1.Secret
		expectError bool
		expectedKey string
	}{
		{
			name: "successful retrieval with default key",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile",
					Namespace: "default",
				},
				Spec: nextdnsv1alpha1.NextDNSProfileSpec{
					Name: "Test Profile",
					CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
						Name: "nextdns-secret",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nextdns-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"api-key": []byte("test-api-key-12345"),
				},
			},
			expectError: false,
			expectedKey: "test-api-key-12345",
		},
		{
			name: "successful retrieval with custom key",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile",
					Namespace: "default",
				},
				Spec: nextdnsv1alpha1.NextDNSProfileSpec{
					Name: "Test Profile",
					CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
						Name: "nextdns-secret",
						Key:  "custom-key",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nextdns-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"custom-key": []byte("custom-api-key"),
				},
			},
			expectError: false,
			expectedKey: "custom-api-key",
		},
		{
			name: "secret not found",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile",
					Namespace: "default",
				},
				Spec: nextdnsv1alpha1.NextDNSProfileSpec{
					Name: "Test Profile",
					CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
						Name: "missing-secret",
					},
				},
			},
			secret:      nil,
			expectError: true,
		},
		{
			name: "key not found in secret",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile",
					Namespace: "default",
				},
				Spec: nextdnsv1alpha1.NextDNSProfileSpec{
					Name: "Test Profile",
					CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
						Name: "nextdns-secret",
						Key:  "missing-key",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nextdns-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"other-key": []byte("value"),
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []client.Object
			if tt.secret != nil {
				objects = append(objects, tt.secret)
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()

			reconciler := &NextDNSProfileReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			apiKey, err := reconciler.getAPIKey(ctx, tt.profile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedKey, apiKey)
			}
		})
	}
}

func TestResolveListReferences(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	// Create test allowlist
	allowlist := &nextdnsv1alpha1.NextDNSAllowlist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-allowlist",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSAllowlistSpec{
			Domains: []nextdnsv1alpha1.DomainEntry{
				{Domain: "allowed1.com"},
				{Domain: "allowed2.com", Active: boolPtr(true)},
				{Domain: "inactive.com", Active: boolPtr(false)},
			},
		},
	}

	// Create test denylist
	denylist := &nextdnsv1alpha1.NextDNSDenylist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-denylist",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSDenylistSpec{
			Domains: []nextdnsv1alpha1.DomainEntry{
				{Domain: "blocked1.com"},
				{Domain: "blocked2.com"},
			},
		},
	}

	// Create test TLD list
	tldList := &nextdnsv1alpha1.NextDNSTLDList{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tldlist",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSTLDListSpec{
			TLDs: []nextdnsv1alpha1.TLDEntry{
				{TLD: "xyz"},
				{TLD: "tk", Active: boolPtr(true)},
				{TLD: "ml", Active: boolPtr(false)},
			},
		},
	}

	// Create test profile with references
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			AllowlistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "test-allowlist"},
			},
			DenylistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "test-denylist"},
			},
			TLDListRefs: []nextdnsv1alpha1.ListReference{
				{Name: "test-tldlist"},
			},
			Allowlist: []nextdnsv1alpha1.DomainEntry{
				{Domain: "inline-allowed.com"},
			},
			Denylist: []nextdnsv1alpha1.DomainEntry{
				{Domain: "inline-blocked.com"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(allowlist, denylist, tldList, profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	resolved, err := reconciler.resolveListReferences(ctx, profile)
	require.NoError(t, err)

	// Check allowlist (2 active from ref + 1 inline)
	assert.Equal(t, 3, len(resolved.Allowlist))
	assert.Contains(t, resolved.Allowlist, "allowed1.com")
	assert.Contains(t, resolved.Allowlist, "allowed2.com")
	assert.Contains(t, resolved.Allowlist, "inline-allowed.com")
	assert.NotContains(t, resolved.Allowlist, "inactive.com")

	// Check denylist (2 from ref + 1 inline)
	assert.Equal(t, 3, len(resolved.Denylist))
	assert.Contains(t, resolved.Denylist, "blocked1.com")
	assert.Contains(t, resolved.Denylist, "blocked2.com")
	assert.Contains(t, resolved.Denylist, "inline-blocked.com")

	// Check TLDs (2 active from ref)
	assert.Equal(t, 2, len(resolved.TLDs))
	assert.Contains(t, resolved.TLDs, "xyz")
	assert.Contains(t, resolved.TLDs, "tk")
	assert.NotContains(t, resolved.TLDs, "ml")

	// Check resource status
	assert.Equal(t, 1, len(resolved.ResourceStatus.Allowlists))
	assert.Equal(t, "test-allowlist", resolved.ResourceStatus.Allowlists[0].Name)
	assert.Equal(t, 2, resolved.ResourceStatus.Allowlists[0].Count)

	assert.Equal(t, 1, len(resolved.ResourceStatus.Denylists))
	assert.Equal(t, "test-denylist", resolved.ResourceStatus.Denylists[0].Name)
	assert.Equal(t, 2, resolved.ResourceStatus.Denylists[0].Count)

	assert.Equal(t, 1, len(resolved.ResourceStatus.TLDLists))
	assert.Equal(t, "test-tldlist", resolved.ResourceStatus.TLDLists[0].Name)
	assert.Equal(t, 2, resolved.ResourceStatus.TLDLists[0].Count)
}

func TestResolveListReferences_MissingResource(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			AllowlistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "missing-allowlist"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	_, err := reconciler.resolveListReferences(ctx, profile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get allowlist")
}

func TestResolveListReferences_CrossNamespace(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	// Create allowlist in different namespace
	allowlist := &nextdnsv1alpha1.NextDNSAllowlist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-allowlist",
			Namespace: "shared",
		},
		Spec: nextdnsv1alpha1.NextDNSAllowlistSpec{
			Domains: []nextdnsv1alpha1.DomainEntry{
				{Domain: "shared.com"},
			},
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			AllowlistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "shared-allowlist", Namespace: "shared"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(allowlist, profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	resolved, err := reconciler.resolveListReferences(ctx, profile)
	require.NoError(t, err)

	assert.Equal(t, 1, len(resolved.Allowlist))
	assert.Contains(t, resolved.Allowlist, "shared.com")
	assert.Equal(t, "shared", resolved.ResourceStatus.Allowlists[0].Namespace)
}

func TestFindProfilesForAllowlist(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	allowlist := &nextdnsv1alpha1.NextDNSAllowlist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-allowlist",
			Namespace: "default",
		},
	}

	profile1 := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile-1",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Profile 1",
			AllowlistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "test-allowlist"},
			},
		},
	}

	profile2 := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile-2",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Profile 2",
			AllowlistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "other-allowlist"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(allowlist, profile1, profile2).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	requests := reconciler.findProfilesForAllowlist(ctx, allowlist)

	assert.Equal(t, 1, len(requests))
	assert.Equal(t, "profile-1", requests[0].Name)
	assert.Equal(t, "default", requests[0].Namespace)
}

func TestFindProfilesForDenylist(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	denylist := &nextdnsv1alpha1.NextDNSDenylist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-denylist",
			Namespace: "default",
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile-1",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Profile 1",
			DenylistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "test-denylist"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(denylist, profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	requests := reconciler.findProfilesForDenylist(ctx, denylist)

	assert.Equal(t, 1, len(requests))
	assert.Equal(t, "profile-1", requests[0].Name)
}

func TestFindProfilesForTLDList(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	tldList := &nextdnsv1alpha1.NextDNSTLDList{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tldlist",
			Namespace: "default",
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile-1",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Profile 1",
			TLDListRefs: []nextdnsv1alpha1.ListReference{
				{Name: "test-tldlist"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tldList, profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	requests := reconciler.findProfilesForTLDList(ctx, tldList)

	assert.Equal(t, 1, len(requests))
	assert.Equal(t, "profile-1", requests[0].Name)
}

func TestFindProfilesForSecret(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-key"),
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile-1",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Profile 1",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret, profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	requests := reconciler.findProfilesForSecret(ctx, secret)

	assert.Equal(t, 1, len(requests))
	assert.Equal(t, "profile-1", requests[0].Name)
}

func TestFindProfilesForWrongType(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Test with wrong object type
	wrongObj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	// These should return nil for wrong types
	assert.Nil(t, reconciler.findProfilesForAllowlist(ctx, wrongObj))
	assert.Nil(t, reconciler.findProfilesForDenylist(ctx, wrongObj))
	assert.Nil(t, reconciler.findProfilesForTLDList(ctx, wrongObj))
	assert.Nil(t, reconciler.findProfilesForSecret(ctx, wrongObj))
}

func TestReconcile_ProfileNotFound(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent",
			Namespace: "default",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestReconcile_AddFinalizer(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})

	assert.NoError(t, err)
	assert.True(t, result.Requeue)

	// Verify finalizer was added
	updatedProfile := &nextdnsv1alpha1.NextDNSProfile{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-profile", Namespace: "default"}, updatedProfile)
	require.NoError(t, err)
	assert.Contains(t, updatedProfile.Finalizers, FinalizerName)
}

func TestReconcile_MissingCredentials(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "missing-secret",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})

	assert.NoError(t, err)
	assert.NotZero(t, result.RequeueAfter)
}

func TestSetCondition(t *testing.T) {
	scheme := newTestScheme()

	reconciler := &NextDNSProfileReconciler{
		Scheme: scheme,
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Generation: 1,
		},
	}

	reconciler.setCondition(profile, ConditionTypeReady, metav1.ConditionTrue, "TestReason", "Test message")

	assert.Equal(t, 1, len(profile.Status.Conditions))
	assert.Equal(t, ConditionTypeReady, profile.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, profile.Status.Conditions[0].Status)
	assert.Equal(t, "TestReason", profile.Status.Conditions[0].Reason)
	assert.Equal(t, "Test message", profile.Status.Conditions[0].Message)

	// Update the same condition
	reconciler.setCondition(profile, ConditionTypeReady, metav1.ConditionFalse, "NewReason", "New message")

	assert.Equal(t, 1, len(profile.Status.Conditions))
	assert.Equal(t, metav1.ConditionFalse, profile.Status.Conditions[0].Status)
	assert.Equal(t, "NewReason", profile.Status.Conditions[0].Reason)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "nextdns.io/finalizer", FinalizerName)
	assert.Equal(t, "Ready", ConditionTypeReady)
	assert.Equal(t, "Synced", ConditionTypeSynced)
	assert.Equal(t, "ReferencesResolved", ConditionTypeReferencesResolved)
}

func TestFindProfilesForAllowlist_InvalidType(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Pass a different object type
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	requests := reconciler.findProfilesForAllowlist(ctx, secret)
	assert.Nil(t, requests)
}

func TestFindProfilesForDenylist_InvalidType(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Pass a different object type
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	requests := reconciler.findProfilesForDenylist(ctx, secret)
	assert.Nil(t, requests)
}

func TestFindProfilesForTLDList_InvalidType(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Pass a different object type
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	requests := reconciler.findProfilesForTLDList(ctx, secret)
	assert.Nil(t, requests)
}

func TestFindProfilesForSecret_InvalidType(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Pass a different object type
	allowlist := &nextdnsv1alpha1.NextDNSAllowlist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	requests := reconciler.findProfilesForSecret(ctx, allowlist)
	assert.Nil(t, requests)
}

func TestFindProfilesForAllowlist_MultipleProfiles(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	allowlist := &nextdnsv1alpha1.NextDNSAllowlist{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-allowlist",
			Namespace: "default",
		},
	}

	profile1 := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile-1",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Profile 1",
			AllowlistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "shared-allowlist"},
			},
		},
	}

	profile2 := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "profile-2",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Profile 2",
			AllowlistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "shared-allowlist"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(allowlist, profile1, profile2).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	requests := reconciler.findProfilesForAllowlist(ctx, allowlist)

	assert.Equal(t, 2, len(requests))
	names := []string{requests[0].Name, requests[1].Name}
	assert.Contains(t, names, "profile-1")
	assert.Contains(t, names, "profile-2")
}

// Test for handling deletion scenarios
func TestHandleDeletion_NoFinalizer(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
			// No finalizer
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := reconciler.handleDeletion(ctx, profile)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestSyncWithNextDNS_CreateNewProfile(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	lists := &ResolvedLists{
		Allowlist: []string{"allowed.com"},
		Denylist:  []string{"blocked.com"},
		TLDs:      []string{"xyz"},
	}

	err := reconciler.syncWithNextDNS(ctx, profile, "test-api-key", lists)
	require.NoError(t, err)

	// Verify profile was created
	assert.NotEmpty(t, profile.Status.ProfileID)
	assert.Contains(t, profile.Status.Fingerprint, ".dns.nextdns.io")

	// Verify mock was called
	assert.True(t, mockClient.createProfileCalled)
	assert.True(t, mockClient.updateProfileCalled)
	assert.True(t, mockClient.syncDenylistCalled)
	assert.True(t, mockClient.syncAllowlistCalled)
	assert.True(t, mockClient.syncSecurityTLDsCalled)
}

func TestSyncWithNextDNS_AdoptExistingProfile(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name:      "Adopted Profile",
			ProfileID: "existing-profile-123",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	lists := &ResolvedLists{
		Allowlist: []string{},
		Denylist:  []string{},
		TLDs:      []string{},
	}

	err := reconciler.syncWithNextDNS(ctx, profile, "test-api-key", lists)
	require.NoError(t, err)

	// Verify profile was adopted (not created)
	assert.Equal(t, "existing-profile-123", profile.Status.ProfileID)
	assert.False(t, mockClient.createProfileCalled)
	assert.True(t, mockClient.getProfileCalled)
}

func TestSyncWithNextDNS_WithSecuritySettings(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Security Profile",
			Security: &nextdnsv1alpha1.SecuritySpec{
				AIThreatDetection:  boolPtr(true),
				GoogleSafeBrowsing: boolPtr(true),
				Cryptojacking:      boolPtr(false),
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "existing-id",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	lists := &ResolvedLists{}

	err := reconciler.syncWithNextDNS(ctx, profile, "test-api-key", lists)
	require.NoError(t, err)

	assert.True(t, mockClient.updateSecurityCalled)
	assert.NotNil(t, mockClient.securityConfig)
	assert.True(t, mockClient.securityConfig.AIThreatDetection)
	assert.True(t, mockClient.securityConfig.GoogleSafeBrowsing)
	assert.False(t, mockClient.securityConfig.Cryptojacking)
}

func TestSyncWithNextDNS_WithPrivacySettings(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Privacy Profile",
			Privacy: &nextdnsv1alpha1.PrivacySpec{
				DisguisedTrackers: boolPtr(true),
				AllowAffiliate:    boolPtr(false),
				Blocklists: []nextdnsv1alpha1.BlocklistEntry{
					{ID: "blocklist-1"},
					{ID: "blocklist-2", Active: boolPtr(false)},
				},
				Natives: []nextdnsv1alpha1.NativeEntry{
					{ID: "apple"},
					{ID: "windows"},
				},
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "existing-id",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	lists := &ResolvedLists{}

	err := reconciler.syncWithNextDNS(ctx, profile, "test-api-key", lists)
	require.NoError(t, err)

	assert.True(t, mockClient.updatePrivacyCalled)
	assert.True(t, mockClient.syncPrivacyBlocklistsCalled)
	assert.True(t, mockClient.syncPrivacyNativesCalled)
	assert.Equal(t, 1, len(mockClient.blocklists)) // Only active one
	assert.Equal(t, 2, len(mockClient.natives))
}

func TestSyncWithNextDNS_WithParentalControl(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Parental Profile",
			ParentalControl: &nextdnsv1alpha1.ParentalControlSpec{
				SafeSearch:            boolPtr(true),
				YouTubeRestrictedMode: boolPtr(true),
				Categories: []nextdnsv1alpha1.CategoryEntry{
					{ID: "adult"},
					{ID: "gambling", Active: boolPtr(true)},
					{ID: "drugs", Active: boolPtr(false)},
				},
				Services: []nextdnsv1alpha1.ServiceEntry{
					{ID: "tiktok"},
					{ID: "instagram"},
				},
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "existing-id",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	lists := &ResolvedLists{}

	err := reconciler.syncWithNextDNS(ctx, profile, "test-api-key", lists)
	require.NoError(t, err)

	assert.True(t, mockClient.updateParentalControlCalled)
	assert.NotNil(t, mockClient.parentalControlConfig)
	assert.True(t, mockClient.parentalControlConfig.SafeSearch)
	assert.True(t, mockClient.parentalControlConfig.YouTubeRestrictedMode)
	assert.Equal(t, 2, len(mockClient.parentalControlConfig.Categories)) // Only active ones
	assert.Equal(t, 2, len(mockClient.parentalControlConfig.Services))
}

func TestSyncWithNextDNS_WithSettings(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Settings Profile",
			Settings: &nextdnsv1alpha1.SettingsSpec{
				Logs: &nextdnsv1alpha1.LogsSpec{
					Enabled:   boolPtr(true),
					Retention: "30d",
				},
				BlockPage: &nextdnsv1alpha1.BlockPageSpec{
					Enabled: boolPtr(true),
				},
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "existing-id",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	lists := &ResolvedLists{}

	err := reconciler.syncWithNextDNS(ctx, profile, "test-api-key", lists)
	require.NoError(t, err)

	assert.True(t, mockClient.updateSettingsCalled)
	assert.NotNil(t, mockClient.settingsConfig)
	assert.True(t, mockClient.settingsConfig.LogsEnabled)
	assert.Equal(t, 30, mockClient.settingsConfig.LogRetention)
	assert.True(t, mockClient.settingsConfig.BlockPageEnable)
}

func TestSyncWithNextDNS_ClientFactoryError(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return nil, assert.AnError
		},
	}

	lists := &ResolvedLists{}

	err := reconciler.syncWithNextDNS(ctx, profile, "test-api-key", lists)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create NextDNS client")
}

func TestHandleDeletion_WithFinalizer_CreatedProfile(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			// No ProfileID means it was created by the operator
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "created-profile-123",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	result, err := reconciler.handleDeletion(ctx, profile)
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	// Verify profile was deleted from NextDNS
	assert.True(t, mockClient.deleteProfileCalled)
	assert.Equal(t, "created-profile-123", mockClient.deletedProfileID)

	// Verify finalizer was removed
	assert.NotContains(t, profile.Finalizers, FinalizerName)
}

func TestHandleDeletion_WithFinalizer_AdoptedProfile(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name:      "Adopted Profile",
			ProfileID: "adopted-profile-123", // Has ProfileID means it was adopted
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "adopted-profile-123",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	result, err := reconciler.handleDeletion(ctx, profile)
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	// Verify profile was NOT deleted from NextDNS (because it was adopted)
	assert.False(t, mockClient.deleteProfileCalled)

	// Verify finalizer was still removed
	assert.NotContains(t, profile.Finalizers, FinalizerName)
}

func TestHandleDeletion_MissingCredentials(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "missing-secret",
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "profile-123",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := reconciler.handleDeletion(ctx, profile)
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	// Finalizer should still be removed even if credentials are missing
	assert.NotContains(t, profile.Finalizers, FinalizerName)
}

func TestReconcile_FullFlow_WithMock(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Full Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
			Security: &nextdnsv1alpha1.SecuritySpec{
				AIThreatDetection: boolPtr(true),
			},
			Privacy: &nextdnsv1alpha1.PrivacySpec{
				DisguisedTrackers: boolPtr(true),
			},
			Allowlist: []nextdnsv1alpha1.DomainEntry{
				{Domain: "allowed.com"},
			},
			Denylist: []nextdnsv1alpha1.DomainEntry{
				{Domain: "blocked.com"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify mock was called
	assert.True(t, mockClient.createProfileCalled)
	assert.True(t, mockClient.updateProfileCalled)
	assert.True(t, mockClient.updateSecurityCalled)
	assert.True(t, mockClient.updatePrivacyCalled)
	assert.True(t, mockClient.syncDenylistCalled)
	assert.True(t, mockClient.syncAllowlistCalled)

	// Verify status was updated
	updatedProfile := &nextdnsv1alpha1.NextDNSProfile{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-profile", Namespace: "default"}, updatedProfile)
	require.NoError(t, err)
	assert.NotEmpty(t, updatedProfile.Status.ProfileID)
	assert.NotNil(t, updatedProfile.Status.LastSyncTime)
}

func TestReconcile_FailedSync(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()
	mockClient.createProfileError = assert.AnError

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockClient, nil
		},
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})

	// Should not return error but requeue
	assert.NoError(t, err)
	assert.NotZero(t, result.RequeueAfter)
}

func TestReconcile_FailedListResolution(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
			AllowlistRefs: []nextdnsv1alpha1.ListReference{
				{Name: "missing-allowlist"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})

	// Should not return error but requeue
	assert.NoError(t, err)
	assert.NotZero(t, result.RequeueAfter)
}

func TestDefaultClientFactory(t *testing.T) {
	// Test that DefaultClientFactory returns error for empty API key
	client, err := DefaultClientFactory("")
	assert.Error(t, err)
	assert.Nil(t, client)
}

// mockNextDNSClient is a test mock for the NextDNS client
type mockNextDNSClient struct {
	// Call tracking
	createProfileCalled         bool
	getProfileCalled            bool
	updateProfileCalled         bool
	deleteProfileCalled         bool
	updateSecurityCalled        bool
	updatePrivacyCalled         bool
	updateParentalControlCalled bool
	updateSettingsCalled        bool
	syncDenylistCalled          bool
	syncAllowlistCalled         bool
	syncSecurityTLDsCalled      bool
	syncPrivacyBlocklistsCalled bool
	syncPrivacyNativesCalled    bool

	// Captured values
	createdProfileName    string
	deletedProfileID      string
	securityConfig        *nextdns.SecurityConfig
	privacyConfig         *nextdns.PrivacyConfig
	parentalControlConfig *nextdns.ParentalControlConfig
	settingsConfig        *nextdns.SettingsConfig
	blocklists            []string
	natives               []string

	// Error injection
	createProfileError error
	getProfileError    error

	// Profile counter for generating IDs
	profileCounter int
}

func newMockNextDNSClient() *mockNextDNSClient {
	return &mockNextDNSClient{
		profileCounter: 0,
	}
}

func (m *mockNextDNSClient) CreateProfile(ctx context.Context, name string) (string, error) {
	m.createProfileCalled = true
	m.createdProfileName = name
	if m.createProfileError != nil {
		return "", m.createProfileError
	}
	m.profileCounter++
	return fmt.Sprintf("mock-profile-%d", m.profileCounter), nil
}

func (m *mockNextDNSClient) GetProfile(ctx context.Context, profileID string) (*sdknextdns.Profile, error) {
	m.getProfileCalled = true
	if m.getProfileError != nil {
		return nil, m.getProfileError
	}
	return &sdknextdns.Profile{
		Name: "Mock Profile",
	}, nil
}

func (m *mockNextDNSClient) UpdateProfile(ctx context.Context, profileID, name string) error {
	m.updateProfileCalled = true
	return nil
}

func (m *mockNextDNSClient) DeleteProfile(ctx context.Context, profileID string) error {
	m.deleteProfileCalled = true
	m.deletedProfileID = profileID
	return nil
}

func (m *mockNextDNSClient) UpdateSecurity(ctx context.Context, profileID string, config *nextdns.SecurityConfig) error {
	m.updateSecurityCalled = true
	m.securityConfig = config
	return nil
}

func (m *mockNextDNSClient) GetSecurity(ctx context.Context, profileID string) (*sdknextdns.Security, error) {
	return &sdknextdns.Security{}, nil
}

func (m *mockNextDNSClient) UpdatePrivacy(ctx context.Context, profileID string, config *nextdns.PrivacyConfig) error {
	m.updatePrivacyCalled = true
	m.privacyConfig = config
	return nil
}

func (m *mockNextDNSClient) GetPrivacy(ctx context.Context, profileID string) (*sdknextdns.Privacy, error) {
	return &sdknextdns.Privacy{}, nil
}

func (m *mockNextDNSClient) SyncPrivacyBlocklists(ctx context.Context, profileID string, blocklists []string) error {
	m.syncPrivacyBlocklistsCalled = true
	m.blocklists = blocklists
	return nil
}

func (m *mockNextDNSClient) SyncPrivacyNatives(ctx context.Context, profileID string, natives []string) error {
	m.syncPrivacyNativesCalled = true
	m.natives = natives
	return nil
}

func (m *mockNextDNSClient) UpdateParentalControl(ctx context.Context, profileID string, config *nextdns.ParentalControlConfig) error {
	m.updateParentalControlCalled = true
	m.parentalControlConfig = config
	return nil
}

func (m *mockNextDNSClient) GetParentalControl(ctx context.Context, profileID string) (*sdknextdns.ParentalControl, error) {
	return &sdknextdns.ParentalControl{}, nil
}

func (m *mockNextDNSClient) SyncDenylist(ctx context.Context, profileID string, entries []nextdns.DomainEntry) error {
	m.syncDenylistCalled = true
	return nil
}

func (m *mockNextDNSClient) SyncAllowlist(ctx context.Context, profileID string, domains []string) error {
	m.syncAllowlistCalled = true
	return nil
}

func (m *mockNextDNSClient) SyncSecurityTLDs(ctx context.Context, profileID string, tlds []string) error {
	m.syncSecurityTLDsCalled = true
	return nil
}

func (m *mockNextDNSClient) GetDenylist(ctx context.Context, profileID string) ([]*sdknextdns.Denylist, error) {
	return []*sdknextdns.Denylist{}, nil
}

func (m *mockNextDNSClient) GetAllowlist(ctx context.Context, profileID string) ([]*sdknextdns.Allowlist, error) {
	return []*sdknextdns.Allowlist{}, nil
}

func (m *mockNextDNSClient) GetSecurityTLDs(ctx context.Context, profileID string) ([]*sdknextdns.SecurityTlds, error) {
	return []*sdknextdns.SecurityTlds{}, nil
}

func (m *mockNextDNSClient) UpdateSettings(ctx context.Context, profileID string, config *nextdns.SettingsConfig) error {
	m.updateSettingsCalled = true
	m.settingsConfig = config
	return nil
}
