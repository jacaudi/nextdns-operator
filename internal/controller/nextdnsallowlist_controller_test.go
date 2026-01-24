package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

func TestCountActiveDomains(t *testing.T) {
	tests := []struct {
		name     string
		domains  []nextdnsv1alpha1.DomainEntry
		expected int
	}{
		{
			name: "all domains active (nil means active)",
			domains: []nextdnsv1alpha1.DomainEntry{
				{Domain: "example.com", Active: nil},
				{Domain: "test.com", Active: nil},
				{Domain: "demo.org", Active: nil},
			},
			expected: 3,
		},
		{
			name: "mixed active and inactive domains",
			domains: []nextdnsv1alpha1.DomainEntry{
				{Domain: "example.com", Active: boolPtr(true)},
				{Domain: "test.com", Active: boolPtr(false)},
				{Domain: "demo.org", Active: nil},
				{Domain: "inactive.com", Active: boolPtr(false)},
			},
			expected: 2,
		},
		{
			name: "no active domains",
			domains: []nextdnsv1alpha1.DomainEntry{
				{Domain: "example.com", Active: boolPtr(false)},
				{Domain: "test.com", Active: boolPtr(false)},
			},
			expected: 0,
		},
		{
			name:     "empty domain list",
			domains:  []nextdnsv1alpha1.DomainEntry{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := &NextDNSAllowlistReconciler{}
			result := reconciler.countActiveDomains(tt.domains)
			assert.Equal(t, tt.expected, result, "countActiveDomains() returned unexpected count")
		})
	}
}

func TestNextDNSAllowlistReconciler_findProfileReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nextdnsv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name     string
		list     *nextdnsv1alpha1.NextDNSAllowlist
		profiles []nextdnsv1alpha1.NextDNSProfile
		expected []nextdnsv1alpha1.ResourceReference
	}{
		{
			name: "single profile references list",
			list: &nextdnsv1alpha1.NextDNSAllowlist{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-allowlist",
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
						AllowlistRefs: []nextdnsv1alpha1.ListReference{
							{Name: "test-allowlist"},
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
			list: &nextdnsv1alpha1.NextDNSAllowlist{
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
						AllowlistRefs: []nextdnsv1alpha1.ListReference{
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
						AllowlistRefs: []nextdnsv1alpha1.ListReference{
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
			list: &nextdnsv1alpha1.NextDNSAllowlist{
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
						AllowlistRefs: []nextdnsv1alpha1.ListReference{
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
			list: &nextdnsv1alpha1.NextDNSAllowlist{
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
						AllowlistRefs: []nextdnsv1alpha1.ListReference{
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

			r := &NextDNSAllowlistReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			refs, err := r.findProfileReferences(context.Background(), tt.list)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, refs)
		})
	}
}
