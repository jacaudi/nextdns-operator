package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

func TestNextDNSTLDListReconciler_countActiveTLDs(t *testing.T) {
	tests := []struct {
		name     string
		tlds     []nextdnsv1alpha1.TLDEntry
		expected int
	}{
		{
			name: "all tlds active (nil means active)",
			tlds: []nextdnsv1alpha1.TLDEntry{
				{TLD: "example.com", Active: nil},
				{TLD: "test.com", Active: nil},
				{TLD: "demo.org", Active: nil},
			},
			expected: 3,
		},
		{
			name: "mixed active and inactive domains",
			tlds: []nextdnsv1alpha1.TLDEntry{
				{TLD: "example.com", Active: boolPtr(true)},
				{TLD: "test.com", Active: boolPtr(false)},
				{TLD: "demo.org", Active: nil},
				{TLD: "inactive.com", Active: boolPtr(false)},
			},
			expected: 2,
		},
		{
			name: "no active domains",
			tlds: []nextdnsv1alpha1.TLDEntry{
				{TLD: "example.com", Active: boolPtr(false)},
				{TLD: "test.com", Active: boolPtr(false)},
			},
			expected: 0,
		},
		{
			name:     "empty domain list",
			tlds:     []nextdnsv1alpha1.TLDEntry{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := &NextDNSTLDListReconciler{}
			result := reconciler.countActiveTLDs(tt.tlds)
			assert.Equal(t, tt.expected, result, "countActiveTLDs() returned unexpected count")
		})
	}
}

func TestNextDNSTLDListReconciler_findProfileReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nextdnsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name     string
		list     *nextdnsv1alpha1.NextDNSTLDList
		profiles []nextdnsv1alpha1.NextDNSProfile
		expected []nextdnsv1alpha1.ResourceReference
	}{
		{
			name: "single profile references list",
			list: &nextdnsv1alpha1.NextDNSTLDList{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tldlist",
					Namespace: "default",
				},
			},
			profiles: []nextdnsv1alpha1.NextDNSProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "default",
					},
					Spec: nextdnsv1alpha1.NextDNSProfileSpec{
						TLDListRefs: []nextdnsv1alpha1.ListReference{
							{Name: "test-tldlist"},
						},
					},
				},
			},
			expected: []nextdnsv1alpha1.ResourceReference{
				{Name: "profile1", Namespace: "default"},
			},
		},
		{
			name: "multiple profiles reference list",
			list: &nextdnsv1alpha1.NextDNSTLDList{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shared-list",
					Namespace: "default",
				},
			},
			profiles: []nextdnsv1alpha1.NextDNSProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "default",
					},
					Spec: nextdnsv1alpha1.NextDNSProfileSpec{
						TLDListRefs: []nextdnsv1alpha1.ListReference{
							{Name: "shared-list"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile2",
						Namespace: "default",
					},
					Spec: nextdnsv1alpha1.NextDNSProfileSpec{
						TLDListRefs: []nextdnsv1alpha1.ListReference{
							{Name: "shared-list"},
						},
					},
				},
			},
			expected: []nextdnsv1alpha1.ResourceReference{
				{Name: "profile1", Namespace: "default"},
				{Name: "profile2", Namespace: "default"},
			},
		},
		{
			name: "cross-namespace reference",
			list: &nextdnsv1alpha1.NextDNSTLDList{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "global-list",
					Namespace: "lists",
				},
			},
			profiles: []nextdnsv1alpha1.NextDNSProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "apps",
					},
					Spec: nextdnsv1alpha1.NextDNSProfileSpec{
						TLDListRefs: []nextdnsv1alpha1.ListReference{
							{Name: "global-list", Namespace: "lists"},
						},
					},
				},
			},
			expected: []nextdnsv1alpha1.ResourceReference{
				{Name: "profile1", Namespace: "apps"},
			},
		},
		{
			name: "no profiles reference list",
			list: &nextdnsv1alpha1.NextDNSTLDList{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unused-list",
					Namespace: "default",
				},
			},
			profiles: []nextdnsv1alpha1.NextDNSProfile{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "profile1",
						Namespace: "default",
					},
					Spec: nextdnsv1alpha1.NextDNSProfileSpec{
						TLDListRefs: []nextdnsv1alpha1.ListReference{
							{Name: "other-list"},
						},
					},
				},
			},
			expected: []nextdnsv1alpha1.ResourceReference{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []client.Object{tt.list}
			for i := range tt.profiles {
				objs = append(objs, &tt.profiles[i])
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()

			r := &NextDNSTLDListReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			refs, err := r.findProfileReferences(context.Background(), tt.list)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, refs)
		})
	}
}

func TestNextDNSTLDListReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nextdnsv1alpha1.AddToScheme(scheme)

	list := &nextdnsv1alpha1.NextDNSTLDList{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-list",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSTLDListSpec{
			TLDs: []nextdnsv1alpha1.TLDEntry{
				{TLD: "example.com"},
				{TLD: "test.com", Active: boolPtr(false)},
			},
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			TLDListRefs: []nextdnsv1alpha1.ListReference{
				{Name: "test-list"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(list, profile).
		WithStatusSubresource(&nextdnsv1alpha1.NextDNSTLDList{}).
		Build()

	r := &NextDNSTLDListReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-list",
			Namespace: "default",
		},
	}

	// First reconcile - should add finalizer
	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, result.Requeue)

	// Get updated list
	var updatedList nextdnsv1alpha1.NextDNSTLDList
	err = fakeClient.Get(context.Background(), req.NamespacedName, &updatedList)
	assert.NoError(t, err)
	assert.Contains(t, updatedList.Finalizers, TLDListFinalizerName)

	// Second reconcile - should update status
	result, err = r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, result.Requeue)

	// Verify status
	err = fakeClient.Get(context.Background(), req.NamespacedName, &updatedList)
	assert.NoError(t, err)
	assert.Equal(t, 1, updatedList.Status.TLDCount) // Only 1 active
	assert.Len(t, updatedList.Status.ProfileRefs, 1)
	assert.Equal(t, "test-profile", updatedList.Status.ProfileRefs[0].Name)

	// Check conditions
	validCond := meta.FindStatusCondition(updatedList.Status.Conditions, "Valid")
	assert.NotNil(t, validCond)
	assert.Equal(t, metav1.ConditionTrue, validCond.Status)

	inUseCond := meta.FindStatusCondition(updatedList.Status.Conditions, "InUse")
	assert.NotNil(t, inUseCond)
	assert.Equal(t, metav1.ConditionTrue, inUseCond.Status)
}

func TestNextDNSTLDListReconciler_HandleDeletion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nextdnsv1alpha1.AddToScheme(scheme)

	t.Run("deletion blocked when profiles reference list", func(t *testing.T) {
		now := metav1.Now()
		list := &nextdnsv1alpha1.NextDNSTLDList{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-list",
				Namespace:         "default",
				Finalizers:        []string{TLDListFinalizerName},
				DeletionTimestamp: &now,
			},
			Spec: nextdnsv1alpha1.NextDNSTLDListSpec{
				TLDs: []nextdnsv1alpha1.TLDEntry{
					{TLD: "example.com"},
				},
			},
			Status: nextdnsv1alpha1.NextDNSTLDListStatus{
				ProfileRefs: []nextdnsv1alpha1.ResourceReference{
					{Name: "profile1", Namespace: "default"},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(list).
			WithStatusSubresource(&nextdnsv1alpha1.NextDNSTLDList{}).
			Build()

		r := &NextDNSTLDListReconciler{
			Client: fakeClient,
			Scheme: scheme,
		}

		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-list",
				Namespace: "default",
			},
		}

		// Reconcile should block deletion
		result, err := r.Reconcile(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, 30*time.Second, result.RequeueAfter)

		// Verify DeletionBlocked condition is set
		var updatedList nextdnsv1alpha1.NextDNSTLDList
		err = fakeClient.Get(context.Background(), req.NamespacedName, &updatedList)
		assert.NoError(t, err)

		deletionBlockedCond := meta.FindStatusCondition(updatedList.Status.Conditions, "DeletionBlocked")
		assert.NotNil(t, deletionBlockedCond)
		assert.Equal(t, metav1.ConditionTrue, deletionBlockedCond.Status)
		assert.Equal(t, "InUseByProfiles", deletionBlockedCond.Reason)
		assert.Contains(t, deletionBlockedCond.Message, "profile1")

		// Finalizer should still be present
		assert.Contains(t, updatedList.Finalizers, TLDListFinalizerName)
	})

	t.Run("deletion allowed when no profiles reference list", func(t *testing.T) {
		now := metav1.Now()
		list := &nextdnsv1alpha1.NextDNSTLDList{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-list",
				Namespace:         "default",
				Finalizers:        []string{TLDListFinalizerName},
				DeletionTimestamp: &now,
			},
			Spec: nextdnsv1alpha1.NextDNSTLDListSpec{
				TLDs: []nextdnsv1alpha1.TLDEntry{
					{TLD: "example.com"},
				},
			},
			Status: nextdnsv1alpha1.NextDNSTLDListStatus{
				ProfileRefs: []nextdnsv1alpha1.ResourceReference{}, // No references
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(list).
			WithStatusSubresource(&nextdnsv1alpha1.NextDNSTLDList{}).
			Build()

		r := &NextDNSTLDListReconciler{
			Client: fakeClient,
			Scheme: scheme,
		}

		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-list",
				Namespace: "default",
			},
		}

		// Reconcile should allow deletion by removing finalizer
		result, err := r.Reconcile(context.Background(), req)
		assert.NoError(t, err)
		assert.False(t, result.Requeue)
		assert.Equal(t, time.Duration(0), result.RequeueAfter)

		// After finalizer is removed, the resource will be deleted
		// Verify we can't find it anymore (simulates successful deletion)
		var updatedList nextdnsv1alpha1.NextDNSTLDList
		err = fakeClient.Get(context.Background(), req.NamespacedName, &updatedList)
		assert.Error(t, err)                              // Resource should be gone
		assert.True(t, client.IgnoreNotFound(err) == nil) // Should be NotFound error
	})
}

func TestNextDNSTLDListReconciler_findTLDListsForProfile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nextdnsv1alpha1.AddToScheme(scheme)

	list1 := &nextdnsv1alpha1.NextDNSTLDList{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "list1",
			Namespace: "default",
		},
	}

	list2 := &nextdnsv1alpha1.NextDNSTLDList{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "list2",
			Namespace: "other",
		},
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			TLDListRefs: []nextdnsv1alpha1.ListReference{
				{Name: "list1"},                     // Same namespace
				{Name: "list2", Namespace: "other"}, // Cross-namespace
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(list1, list2, profile).
		Build()

	r := &NextDNSTLDListReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	requests := r.findTLDListsForProfile(context.Background(), profile)

	expected := []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "list1", Namespace: "default"}},
		{NamespacedName: types.NamespacedName{Name: "list2", Namespace: "other"}},
	}

	assert.ElementsMatch(t, expected, requests)
}
