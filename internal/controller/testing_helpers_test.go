package controller

import (
	"testing"

	"github.com/jacaudi/nextdns-operator/internal/nextdns"
	"github.com/stretchr/testify/assert"
)

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
