package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
