package controller

import (
	"testing"

	"github.com/jacaudi/nextdns-operator/internal/nextdns"
	"github.com/stretchr/testify/assert"
)

// boolPtr is a test helper that returns a pointer to a bool value.
// This function is used across multiple test files to create bool pointers
// for testing optional boolean fields.
func boolPtr(b bool) *bool {
	return &b
}

// assertContainsDomainEntry is a test helper that asserts a slice of DomainEntry
// contains an entry with the given domain and active status.
func assertContainsDomainEntry(t *testing.T, entries []nextdns.DomainEntry, domain string, active bool) {
	t.Helper()
	for _, entry := range entries {
		if entry.Domain == domain {
			assert.Equal(t, active, entry.Active, "domain %s has unexpected active status", domain)
			return
		}
	}
	assert.Fail(t, "domain not found in entries", "domain %s not found in %v", domain, entries)
}
