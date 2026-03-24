package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	sdknextdns "github.com/jacaudi/nextdns-go/nextdns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		Allowlist: []nextdns.DomainEntry{
			{Domain: "good.com", Active: true},
			{Domain: "allowed.com", Active: true},
		},
		Denylist: []nextdns.DomainEntry{
			{Domain: "bad.com", Active: true},
			{Domain: "blocked.com", Active: true},
		},
		TLDs: []string{"xyz", "tk"},
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

	// Check allowlist (3 from ref including inactive + 1 inline)
	assert.Equal(t, 4, len(resolved.Allowlist))
	assertContainsDomainEntry(t, resolved.Allowlist, "allowed1.com", true)
	assertContainsDomainEntry(t, resolved.Allowlist, "allowed2.com", true)
	assertContainsDomainEntry(t, resolved.Allowlist, "inactive.com", false)
	assertContainsDomainEntry(t, resolved.Allowlist, "inline-allowed.com", true)

	// Check denylist (2 from ref + 1 inline)
	assert.Equal(t, 3, len(resolved.Denylist))
	assertContainsDomainEntry(t, resolved.Denylist, "blocked1.com", true)
	assertContainsDomainEntry(t, resolved.Denylist, "blocked2.com", true)
	assertContainsDomainEntry(t, resolved.Denylist, "inline-blocked.com", true)

	// Check TLDs (TLDs stay as strings, only active ones are included since NextDNS API doesn't support active field for TLDs)
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
	assertContainsDomainEntry(t, resolved.Allowlist, "shared.com", true)
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
	assert.Greater(t, result.RequeueAfter, time.Duration(0))

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
	assert.Equal(t, "ConfigImported", ConditionTypeConfigImported)
	assert.Equal(t, "ObserveOnly", ConditionTypeObserveOnly)
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
		Allowlist: []nextdns.DomainEntry{{Domain: "allowed.com", Active: true}},
		Denylist:  []nextdns.DomainEntry{{Domain: "blocked.com", Active: true}},
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
		Allowlist: []nextdns.DomainEntry{},
		Denylist:  []nextdns.DomainEntry{},
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
	denylistEntries       []nextdns.DomainEntry

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
	m.denylistEntries = entries
	return nil
}

func (m *mockNextDNSClient) SyncAllowlist(ctx context.Context, profileID string, entries []nextdns.DomainEntry) error {
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

func (m *mockNextDNSClient) AddAllowlistEntry(ctx context.Context, profileID string, domain string, active bool) error {
	return nil
}

func (m *mockNextDNSClient) DeleteAllowlistEntry(ctx context.Context, profileID string, domain string) error {
	return nil
}

func (m *mockNextDNSClient) AddDenylistEntry(ctx context.Context, profileID string, domain string, active bool) error {
	return nil
}

func (m *mockNextDNSClient) DeleteDenylistEntry(ctx context.Context, profileID string, domain string) error {
	return nil
}

func (m *mockNextDNSClient) AddSecurityTLD(ctx context.Context, profileID string, tld string) error {
	return nil
}

func (m *mockNextDNSClient) DeleteSecurityTLD(ctx context.Context, profileID string, tld string) error {
	return nil
}

func (m *mockNextDNSClient) AddPrivacyNative(ctx context.Context, profileID string, nativeID string) error {
	return nil
}

func (m *mockNextDNSClient) DeletePrivacyNative(ctx context.Context, profileID string, nativeID string) error {
	return nil
}

func (m *mockNextDNSClient) GetSettings(ctx context.Context, profileID string) (*sdknextdns.Settings, error) {
	return &sdknextdns.Settings{}, nil
}

func (m *mockNextDNSClient) GetPrivacyBlocklists(ctx context.Context, profileID string) ([]*sdknextdns.PrivacyBlocklists, error) {
	return []*sdknextdns.PrivacyBlocklists{}, nil
}

func (m *mockNextDNSClient) GetPrivacyNatives(ctx context.Context, profileID string) ([]*sdknextdns.PrivacyNatives, error) {
	return []*sdknextdns.PrivacyNatives{}, nil
}

func (m *mockNextDNSClient) GetParentalControlCategories(ctx context.Context, profileID string) ([]*sdknextdns.ParentalControlCategories, error) {
	return []*sdknextdns.ParentalControlCategories{}, nil
}

func (m *mockNextDNSClient) GetParentalControlServices(ctx context.Context, profileID string) ([]*sdknextdns.ParentalControlServices, error) {
	return []*sdknextdns.ParentalControlServices{}, nil
}

func (m *mockNextDNSClient) GetRewrites(ctx context.Context, profileID string) ([]*sdknextdns.Rewrites, error) {
	return []*sdknextdns.Rewrites{}, nil
}

func TestReconcileConfigMap(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	// Create a profile with ConfigMapRef enabled
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-credentials",
			},
			ConfigMapRef: &nextdnsv1alpha1.ConfigMapRef{
				Enabled: true,
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-credentials",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
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
			mock := nextdns.NewMockClient()
			mock.SetProfile("abc123", "Test Profile", "abc123.dns.nextdns.io")
			return mock, nil
		},
	}

	// Reconcile
	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})
	require.NoError(t, err)

	// Verify ConfigMap was created
	configMap := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-profile-nextdns",
		Namespace: "default",
	}, configMap)
	require.NoError(t, err, "ConfigMap should be created")

	// Verify ConfigMap data
	assert.Equal(t, "abc123", configMap.Data["NEXTDNS_PROFILE_ID"])
	assert.Equal(t, "abc123.dns.nextdns.io", configMap.Data["NEXTDNS_DOT"])
	assert.Equal(t, "https://dns.nextdns.io/abc123", configMap.Data["NEXTDNS_DOH"])
	assert.Equal(t, "quic://abc123.dns.nextdns.io", configMap.Data["NEXTDNS_DOQ"])
	assert.Equal(t, "45.90.28.0", configMap.Data["NEXTDNS_IPV4_1"])
	assert.Equal(t, "45.90.30.0", configMap.Data["NEXTDNS_IPV4_2"])
	assert.Equal(t, "2a07:a8c0::", configMap.Data["NEXTDNS_IPV6_1"])
	assert.Equal(t, "2a07:a8c1::", configMap.Data["NEXTDNS_IPV6_2"])

	// Verify owner reference
	require.Len(t, configMap.OwnerReferences, 1)
	assert.Equal(t, "test-profile", configMap.OwnerReferences[0].Name)
	assert.Equal(t, "NextDNSProfile", configMap.OwnerReferences[0].Kind)
}

func TestReconcileConfigMapCustomName(t *testing.T) {
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
				Name: "nextdns-credentials",
			},
			ConfigMapRef: &nextdnsv1alpha1.ConfigMapRef{
				Enabled: true,
				Name:    "my-custom-config",
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "xyz789",
			Fingerprint: "xyz789.dns.nextdns.io",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-credentials",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
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
			mock := nextdns.NewMockClient()
			mock.SetProfile("xyz789", "Test Profile", "xyz789.dns.nextdns.io")
			return mock, nil
		},
	}

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})
	require.NoError(t, err)

	// Verify ConfigMap was created with custom name
	configMap := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "my-custom-config",
		Namespace: "default",
	}, configMap)
	require.NoError(t, err, "ConfigMap with custom name should be created")
	assert.Equal(t, "xyz789", configMap.Data["NEXTDNS_PROFILE_ID"])
}

func TestReconcileConfigMapDisabled(t *testing.T) {
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
				Name: "nextdns-credentials",
			},
			// ConfigMapRef not set - should not create ConfigMap
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "abc123",
			Fingerprint: "abc123.dns.nextdns.io",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-credentials",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
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
			mock := nextdns.NewMockClient()
			mock.SetProfile("abc123", "Test Profile", "abc123.dns.nextdns.io")
			return mock, nil
		},
	}

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})
	require.NoError(t, err)

	// Verify ConfigMap was NOT created
	configMap := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-profile-nextdns",
		Namespace: "default",
	}, configMap)
	assert.True(t, apierrors.IsNotFound(err), "ConfigMap should not be created when disabled")
}

func TestReconcileConfigMapUpdate(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-profile",
			Namespace:  "default",
			UID:        "test-uid",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-credentials",
			},
			ConfigMapRef: &nextdnsv1alpha1.ConfigMapRef{
				Enabled: true,
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID:   "newid456",
			Fingerprint: "newid456.dns.nextdns.io",
		},
	}

	// Pre-existing ConfigMap with old data
	existingConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile-nextdns",
			Namespace: "default",
		},
		Data: map[string]string{
			"NEXTDNS_PROFILE_ID": "oldid123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-credentials",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret, existingConfigMap).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			mock := nextdns.NewMockClient()
			mock.SetProfile("newid456", "Test Profile", "newid456.dns.nextdns.io")
			return mock, nil
		},
	}

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-profile",
			Namespace: "default",
		},
	})
	require.NoError(t, err)

	// Verify ConfigMap was updated
	configMap := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "test-profile-nextdns",
		Namespace: "default",
	}, configMap)
	require.NoError(t, err)
	assert.Equal(t, "newid456", configMap.Data["NEXTDNS_PROFILE_ID"])
}

func TestFindProfilesForConfigMap(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
			UID:       "profile-uid-123",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-credentials",
			},
		},
	}

	// ConfigMap owned by the profile
	ownedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile-nextdns",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: nextdnsv1alpha1.GroupVersion.String(),
					Kind:       "NextDNSProfile",
					Name:       "test-profile",
					UID:        "profile-uid-123",
				},
			},
		},
	}

	// ConfigMap not owned by any profile
	unownedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-configmap",
			Namespace: "default",
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

	t.Run("owned ConfigMap triggers reconcile", func(t *testing.T) {
		requests := reconciler.findProfilesForConfigMap(ctx, ownedConfigMap)
		require.Len(t, requests, 1)
		assert.Equal(t, "test-profile", requests[0].Name)
		assert.Equal(t, "default", requests[0].Namespace)
	})

	t.Run("unowned ConfigMap does not trigger reconcile", func(t *testing.T) {
		requests := reconciler.findProfilesForConfigMap(ctx, unownedConfigMap)
		assert.Empty(t, requests)
	})
}

func TestReconcile_ConfigImport(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	mockClient := newMockNextDNSClient()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-creds",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	importCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "profile-import",
			Namespace:       "default",
			ResourceVersion: "42",
		},
		Data: map[string]string{
			"config.json": `{
				"security": {
					"nrd": true,
					"ddns": true
				},
				"denylist": [
					{"domain": "imported.example.com", "active": true}
				]
			}`,
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
				Name: "nextdns-creds",
				Key:  "api-key",
			},
			ConfigImportRef: &nextdnsv1alpha1.ConfigImportRef{
				Name: "profile-import",
			},
			Security: &nextdnsv1alpha1.SecuritySpec{
				NRD: boolPtr(false), // Spec overrides import's NRD=true
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret, importCM, profile).
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
		NamespacedName: types.NamespacedName{Name: "test-profile", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify security was called with spec overriding import
	require.NotNil(t, mockClient.securityConfig)
	assert.False(t, mockClient.securityConfig.NRD, "Spec NRD=false should override import NRD=true")
	assert.True(t, mockClient.securityConfig.DDNS, "Import DDNS=true should be applied (not in spec)")

	// Verify denylist includes imported domain
	assert.True(t, mockClient.syncDenylistCalled)
	require.Len(t, mockClient.denylistEntries, 1)
	assert.Equal(t, "imported.example.com", mockClient.denylistEntries[0].Domain)

	// Verify status tracks ConfigMap ResourceVersion
	updatedProfile := &nextdnsv1alpha1.NextDNSProfile{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-profile", Namespace: "default"}, updatedProfile)
	require.NoError(t, err)
	assert.Equal(t, "42", updatedProfile.Status.ConfigImportResourceVersion)
}

func TestReconcile_ConfigImport_MissingConfigMap(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-creds",
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
				Name: "nextdns-creds",
				Key:  "api-key",
			},
			ConfigImportRef: &nextdnsv1alpha1.ConfigImportRef{
				Name: "nonexistent-config",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret, profile).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-profile", Namespace: "default"},
	})

	// Should requeue, not return error
	assert.NoError(t, err)
	assert.NotZero(t, result.RequeueAfter)
}

func TestFindProfilesForImportConfigMap(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "import-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Import Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "creds",
			},
			ConfigImportRef: &nextdnsv1alpha1.ConfigImportRef{
				Name: "my-import-config",
			},
		},
	}

	profileNoImport := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-import-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "No Import Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "creds",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, profileNoImport).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// ConfigMap matching configImportRef should trigger reconcile
	importCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-import-config",
			Namespace: "default",
		},
	}

	requests := reconciler.findProfilesForConfigMap(ctx, importCM)
	require.Len(t, requests, 1)
	assert.Equal(t, "import-profile", requests[0].Name)

	// Unrelated ConfigMap should not trigger
	unrelatedCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-other-config",
			Namespace: "default",
		},
	}

	requests = reconciler.findProfilesForConfigMap(ctx, unrelatedCM)
	assert.Empty(t, requests)
}

func TestProfileModeConstants(t *testing.T) {
	assert.Equal(t, nextdnsv1alpha1.ProfileMode("observe"), nextdnsv1alpha1.ProfileModeObserve)
	assert.Equal(t, nextdnsv1alpha1.ProfileMode("managed"), nextdnsv1alpha1.ProfileModeManaged)
}

func TestProfileSpecModeField(t *testing.T) {
	profile := &nextdnsv1alpha1.NextDNSProfile{
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Mode: nextdnsv1alpha1.ProfileModeObserve,
		},
	}
	assert.Equal(t, nextdnsv1alpha1.ProfileModeObserve, profile.Spec.Mode)
}

func TestObservedConfigTypes(t *testing.T) {
	observed := &nextdnsv1alpha1.ObservedConfig{
		Name: "Test Profile",
		Security: &nextdnsv1alpha1.ObservedSecurity{
			AIThreatDetection:  true,
			GoogleSafeBrowsing: true,
		},
		Privacy: &nextdnsv1alpha1.ObservedPrivacy{
			DisguisedTrackers: true,
			Blocklists: []nextdnsv1alpha1.ObservedBlocklistEntry{
				{ID: "nextdns-recommended"},
			},
		},
		Denylist: []nextdnsv1alpha1.ObservedDomainEntry{
			{Domain: "bad.com", Active: true},
		},
		Allowlist: []nextdnsv1alpha1.ObservedDomainEntry{
			{Domain: "good.com", Active: true},
		},
		Settings: &nextdnsv1alpha1.ObservedSettings{
			Web3: true,
		},
		Rewrites: []nextdnsv1alpha1.ObservedRewriteEntry{
			{Name: "example.com", Content: "1.2.3.4"},
		},
	}

	assert.Equal(t, "Test Profile", observed.Name)
	assert.True(t, observed.Security.AIThreatDetection)
	assert.Equal(t, 1, len(observed.Privacy.Blocklists))
	assert.Equal(t, 1, len(observed.Denylist))
}

func TestObservedConfigInStatus(t *testing.T) {
	profile := &nextdnsv1alpha1.NextDNSProfile{}
	profile.Status.ObservedConfig = &nextdnsv1alpha1.ObservedConfig{
		Name: "Observed Profile",
	}
	assert.Equal(t, "Observed Profile", profile.Status.ObservedConfig.Name)
}

func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func TestReconcile_ObserveMode_Success(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-observe",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Mode:      nextdnsv1alpha1.ProfileModeObserve,
			ProfileID: "abc123",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		WithStatusSubresource(profile).
		Build()

	mockNDS := nextdns.NewMockClient()
	mockNDS.SetProfile("abc123", "Remote Profile", "abc123.dns.nextdns.io")
	mockNDS.Security["abc123"] = &sdknextdns.Security{
		AiThreatDetection:  true,
		GoogleSafeBrowsing: true,
	}
	mockNDS.Privacy["abc123"] = &sdknextdns.Privacy{
		DisguisedTrackers: true,
	}
	mockNDS.ParentalControl["abc123"] = &sdknextdns.ParentalControl{
		SafeSearch: true,
	}
	mockNDS.Denylists["abc123"] = []*sdknextdns.Denylist{
		{ID: "bad.com", Active: true},
	}
	mockNDS.Allowlists["abc123"] = []*sdknextdns.Allowlist{
		{ID: "good.com", Active: true},
	}
	mockNDS.Settings["abc123"] = &sdknextdns.Settings{
		Logs:      &sdknextdns.SettingsLogs{Enabled: true, Retention: 7},
		BlockPage: &sdknextdns.SettingsBlockPage{Enabled: true},
		Performance: &sdknextdns.SettingsPerformance{
			Ecs:             true,
			CacheBoost:      true,
			CnameFlattening: true,
		},
		Web3: false,
	}

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		SyncPeriod: 5 * time.Minute,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockNDS, nil
		},
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-observe", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.True(t, result.RequeueAfter > 0)

	// Verify status was updated
	updated := &nextdnsv1alpha1.NextDNSProfile{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-observe", Namespace: "default"}, updated)
	require.NoError(t, err)

	assert.Equal(t, "abc123", updated.Status.ProfileID)
	assert.NotNil(t, updated.Status.ObservedConfig)
	assert.Equal(t, "Remote Profile", updated.Status.ObservedConfig.Name)
	assert.True(t, updated.Status.ObservedConfig.Security.AIThreatDetection)
	assert.True(t, updated.Status.ObservedConfig.Privacy.DisguisedTrackers)
	assert.Equal(t, 1, len(updated.Status.ObservedConfig.Denylist))
	assert.Equal(t, 1, len(updated.Status.ObservedConfig.Allowlist))
	assert.True(t, updated.Status.ObservedConfig.Settings.Logs.Enabled)

	// Verify conditions
	readyCondition := findCondition(updated.Status.Conditions, ConditionTypeReady)
	require.NotNil(t, readyCondition)
	assert.Equal(t, metav1.ConditionTrue, readyCondition.Status)
	assert.Equal(t, "Observed", readyCondition.Reason)

	observeCondition := findCondition(updated.Status.Conditions, ConditionTypeObserveOnly)
	require.NotNil(t, observeCondition)
	assert.Equal(t, metav1.ConditionTrue, observeCondition.Status)

	// Verify no write methods were called
	assert.False(t, mockNDS.WasMethodCalled("UpdateSecurity"))
	assert.False(t, mockNDS.WasMethodCalled("UpdatePrivacy"))
	assert.False(t, mockNDS.WasMethodCalled("UpdateProfile"))
	assert.False(t, mockNDS.WasMethodCalled("CreateProfile"))
}

func TestReconcile_ObserveMode_MissingProfileID(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-observe-no-id",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Mode: nextdnsv1alpha1.ProfileModeObserve,
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
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
		NamespacedName: types.NamespacedName{Name: "test-observe-no-id", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.True(t, result.RequeueAfter > 0)

	updated := &nextdnsv1alpha1.NextDNSProfile{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-observe-no-id", Namespace: "default"}, updated)
	require.NoError(t, err)

	readyCondition := findCondition(updated.Status.Conditions, ConditionTypeReady)
	require.NotNil(t, readyCondition)
	assert.Equal(t, metav1.ConditionFalse, readyCondition.Status)
	assert.Equal(t, "ProfileIDRequired", readyCondition.Reason)
}

func TestReconcile_TransitionBlocked(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-transition",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Mode:      nextdnsv1alpha1.ProfileModeManaged,
			Name:      "Test Profile",
			ProfileID: "abc123",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ObservedConfig: &nextdnsv1alpha1.ObservedConfig{
				Name: "Remote Profile",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
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
		NamespacedName: types.NamespacedName{Name: "test-transition", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.True(t, result.RequeueAfter > 0)

	updated := &nextdnsv1alpha1.NextDNSProfile{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-transition", Namespace: "default"}, updated)
	require.NoError(t, err)

	readyCondition := findCondition(updated.Status.Conditions, ConditionTypeReady)
	require.NotNil(t, readyCondition)
	assert.Equal(t, metav1.ConditionFalse, readyCondition.Status)
	assert.Equal(t, "TransitionBlocked", readyCondition.Reason)
}

func TestReconcile_ManagedMode_MissingName(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-no-name",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Mode: nextdnsv1alpha1.ProfileModeManaged,
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
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
		NamespacedName: types.NamespacedName{Name: "test-no-name", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.True(t, result.RequeueAfter > 0)

	updated := &nextdnsv1alpha1.NextDNSProfile{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-no-name", Namespace: "default"}, updated)
	require.NoError(t, err)

	readyCondition := findCondition(updated.Status.Conditions, ConditionTypeReady)
	require.NotNil(t, readyCondition)
	assert.Equal(t, metav1.ConditionFalse, readyCondition.Status)
	assert.Equal(t, "NameRequired", readyCondition.Reason)
}

func TestReconcile_ObserveMode_APIError(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-observe-error",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Mode:      nextdnsv1alpha1.ProfileModeObserve,
			ProfileID: "abc123",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		WithStatusSubresource(profile).
		Build()

	mockNDS := nextdns.NewMockClient()
	mockNDS.SetProfile("abc123", "Remote Profile", "abc123.dns.nextdns.io")
	mockNDS.GetSecurityError = fmt.Errorf("API rate limited")

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockNDS, nil
		},
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-observe-error", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.True(t, result.RequeueAfter > 0)

	updated := &nextdnsv1alpha1.NextDNSProfile{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-observe-error", Namespace: "default"}, updated)
	require.NoError(t, err)

	readyCondition := findCondition(updated.Status.Conditions, ConditionTypeReady)
	require.NotNil(t, readyCondition)
	assert.Equal(t, metav1.ConditionFalse, readyCondition.Status)
	assert.Equal(t, "ObserveFailed", readyCondition.Reason)
}

func TestHandleDeletion_ObserveMode(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()
	now := metav1.Now()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-observe-delete",
			Namespace:         "default",
			Finalizers:        []string{FinalizerName},
			DeletionTimestamp: &now,
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Mode:      nextdnsv1alpha1.ProfileModeObserve,
			ProfileID: "abc123",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			ProfileID: "abc123",
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		WithStatusSubresource(profile).
		Build()

	mockNDS := nextdns.NewMockClient()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockNDS, nil
		},
	}

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-observe-delete", Namespace: "default"},
	})
	require.NoError(t, err)

	// DeleteProfile should NOT have been called — observe mode doesn't own the profile
	assert.False(t, mockNDS.WasMethodCalled("DeleteProfile"))
}

func TestSpecHasConfig(t *testing.T) {
	tests := []struct {
		name     string
		spec     nextdnsv1alpha1.NextDNSProfileSpec
		expected bool
	}{
		{
			name:     "empty spec",
			spec:     nextdnsv1alpha1.NextDNSProfileSpec{},
			expected: false,
		},
		{
			name:     "only name and mode (no config)",
			spec:     nextdnsv1alpha1.NextDNSProfileSpec{Name: "test", Mode: nextdnsv1alpha1.ProfileModeManaged},
			expected: false,
		},
		{
			name:     "security set",
			spec:     nextdnsv1alpha1.NextDNSProfileSpec{Security: &nextdnsv1alpha1.SecuritySpec{}},
			expected: true,
		},
		{
			name:     "privacy set",
			spec:     nextdnsv1alpha1.NextDNSProfileSpec{Privacy: &nextdnsv1alpha1.PrivacySpec{}},
			expected: true,
		},
		{
			name:     "parental control set",
			spec:     nextdnsv1alpha1.NextDNSProfileSpec{ParentalControl: &nextdnsv1alpha1.ParentalControlSpec{}},
			expected: true,
		},
		{
			name:     "settings set",
			spec:     nextdnsv1alpha1.NextDNSProfileSpec{Settings: &nextdnsv1alpha1.SettingsSpec{}},
			expected: true,
		},
		{
			name: "denylist set",
			spec: nextdnsv1alpha1.NextDNSProfileSpec{Denylist: []nextdnsv1alpha1.DomainEntry{{Domain: "bad.com"}}},
			expected: true,
		},
		{
			name: "allowlist set",
			spec: nextdnsv1alpha1.NextDNSProfileSpec{Allowlist: []nextdnsv1alpha1.DomainEntry{{Domain: "good.com"}}},
			expected: true,
		},
		{
			name: "rewrites set",
			spec: nextdnsv1alpha1.NextDNSProfileSpec{Rewrites: []nextdnsv1alpha1.RewriteEntry{{From: "a", To: "b"}}},
			expected: true,
		},
		{
			name: "denylist refs set",
			spec: nextdnsv1alpha1.NextDNSProfileSpec{DenylistRefs: []nextdnsv1alpha1.ListReference{{Name: "ref"}}},
			expected: true,
		},
		{
			name: "allowlist refs set",
			spec: nextdnsv1alpha1.NextDNSProfileSpec{AllowlistRefs: []nextdnsv1alpha1.ListReference{{Name: "ref"}}},
			expected: true,
		},
		{
			name: "tld list refs set",
			spec: nextdnsv1alpha1.NextDNSProfileSpec{TLDListRefs: []nextdnsv1alpha1.ListReference{{Name: "ref"}}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, specHasConfig(&tt.spec))
		})
	}
}

func TestFormatRetentionString(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		expected string
	}{
		{name: "zero returns empty", days: 0, expected: ""},
		{name: "1 day", days: 1, expected: "1d"},
		{name: "7 days", days: 7, expected: "7d"},
		{name: "30 days", days: 30, expected: "30d"},
		{name: "90 days", days: 90, expected: "90d"},
		{name: "365 days is 1y", days: 365, expected: "1y"},
		{name: "730 days is 2y", days: 730, expected: "2y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRetentionString(tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}
