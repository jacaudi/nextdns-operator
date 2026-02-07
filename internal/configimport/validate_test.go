package configimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_ValidConfig(t *testing.T) {
	active := true
	cfg := &ProfileConfigJSON{
		Denylist:  []DomainEntryJSON{{Domain: "bad.com", Active: &active}},
		Allowlist: []DomainEntryJSON{{Domain: "good.com", Active: &active}},
		Rewrites: []RewriteEntryJSON{
			{From: "example.com", To: "1.2.3.4", Active: &active},
		},
	}
	assert.NoError(t, Validate(cfg))
}

func TestValidate_EmptyConfig(t *testing.T) {
	cfg := &ProfileConfigJSON{}
	assert.NoError(t, Validate(cfg))
}

func TestValidate_ValidDomains(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"simple", "example.com"},
		{"subdomain", "sub.example.com"},
		{"deep subdomain", "a.b.c.example.com"},
		{"wildcard", "*.example.com"},
		{"hyphen", "my-site.example.com"},
		{"numbers", "123.example.com"},
		{"long tld", "example.museum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ProfileConfigJSON{
				Denylist: []DomainEntryJSON{{Domain: tt.domain}},
			}
			assert.NoError(t, Validate(cfg))
		})
	}
}

func TestValidate_InvalidDomains(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"empty", ""},
		{"no tld", "localhost"},
		{"starts with hyphen", "-example.com"},
		{"ends with hyphen", "example-.com"},
		{"double dot", "example..com"},
		{"spaces", "example .com"},
		{"just a dot", "."},
		{"trailing dot only", "com."},
		{"single char tld", "example.c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ProfileConfigJSON{
				Denylist: []DomainEntryJSON{{Domain: tt.domain}},
			}
			err := Validate(cfg)
			require.Error(t, err, "expected error for domain %q", tt.domain)
			assert.Contains(t, err.Error(), "denylist[0]")
		})
	}
}

func TestValidate_DomainMaxLength(t *testing.T) {
	// Build a valid-looking domain that exceeds 253 characters.
	// Use short labels repeated many times.
	longDomain := ""
	for len(longDomain) < 250 {
		longDomain += "abcdefghij."
	}
	longDomain += "com"
	require.Greater(t, len(longDomain), 253, "test domain must exceed 253 chars, got %d", len(longDomain))

	cfg := &ProfileConfigJSON{
		Allowlist: []DomainEntryJSON{{Domain: longDomain}},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestValidate_RewriteFromInvalid(t *testing.T) {
	cfg := &ProfileConfigJSON{
		Rewrites: []RewriteEntryJSON{
			{From: "not valid!", To: "1.2.3.4"},
		},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rewrites[0].from")
}

func TestValidate_RewriteToEmpty(t *testing.T) {
	cfg := &ProfileConfigJSON{
		Rewrites: []RewriteEntryJSON{
			{From: "example.com", To: ""},
		},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rewrites[0].to")
	assert.Contains(t, err.Error(), "must not be empty")
}

func TestValidate_RewriteToIP(t *testing.T) {
	cfg := &ProfileConfigJSON{
		Rewrites: []RewriteEntryJSON{
			{From: "example.com", To: "192.168.1.1"},
		},
	}
	// To field accepts IPs, so this should pass.
	assert.NoError(t, Validate(cfg))
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &ProfileConfigJSON{
		Denylist: []DomainEntryJSON{
			{Domain: ""},
			{Domain: "also invalid!"},
		},
		Allowlist: []DomainEntryJSON{
			{Domain: ""},
		},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "denylist[0]")
	assert.Contains(t, err.Error(), "denylist[1]")
	assert.Contains(t, err.Error(), "allowlist[0]")
}

// --- Upper bounds tests (Task 6) ---

func TestValidate_DenylistExceedsMax(t *testing.T) {
	entries := make([]DomainEntryJSON, MaxImportDenylistEntries+1)
	for i := range entries {
		entries[i] = DomainEntryJSON{Domain: fmt.Sprintf("d%d.example.com", i)}
	}
	cfg := &ProfileConfigJSON{Denylist: entries}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("imported denylist contains %d entries, maximum allowed is %d", MaxImportDenylistEntries+1, MaxImportDenylistEntries))
}

func TestValidate_AllowlistExceedsMax(t *testing.T) {
	entries := make([]DomainEntryJSON, MaxImportAllowlistEntries+1)
	for i := range entries {
		entries[i] = DomainEntryJSON{Domain: fmt.Sprintf("a%d.example.com", i)}
	}
	cfg := &ProfileConfigJSON{Allowlist: entries}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("imported allowlist contains %d entries, maximum allowed is %d", MaxImportAllowlistEntries+1, MaxImportAllowlistEntries))
}

func TestValidate_RewritesExceedsMax(t *testing.T) {
	entries := make([]RewriteEntryJSON, MaxImportRewriteEntries+1)
	for i := range entries {
		entries[i] = RewriteEntryJSON{From: fmt.Sprintf("r%d.example.com", i), To: "1.2.3.4"}
	}
	cfg := &ProfileConfigJSON{Rewrites: entries}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("imported rewrites contains %d entries, maximum allowed is %d", MaxImportRewriteEntries+1, MaxImportRewriteEntries))
}

func TestValidate_BlocklistsExceedsMax(t *testing.T) {
	entries := make([]BlocklistEntryJSON, MaxImportBlocklistEntries+1)
	for i := range entries {
		entries[i] = BlocklistEntryJSON{ID: fmt.Sprintf("bl%d", i)}
	}
	cfg := &ProfileConfigJSON{
		Privacy: &PrivacyJSON{Blocklists: entries},
	}
	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("imported blocklists contains %d entries, maximum allowed is %d", MaxImportBlocklistEntries+1, MaxImportBlocklistEntries))
}

func TestValidate_AtMaxIsOK(t *testing.T) {
	denylist := make([]DomainEntryJSON, MaxImportDenylistEntries)
	for i := range denylist {
		denylist[i] = DomainEntryJSON{Domain: fmt.Sprintf("d%d.example.com", i)}
	}
	allowlist := make([]DomainEntryJSON, MaxImportAllowlistEntries)
	for i := range allowlist {
		allowlist[i] = DomainEntryJSON{Domain: fmt.Sprintf("a%d.example.com", i)}
	}
	rewrites := make([]RewriteEntryJSON, MaxImportRewriteEntries)
	for i := range rewrites {
		rewrites[i] = RewriteEntryJSON{From: fmt.Sprintf("r%d.example.com", i), To: "1.2.3.4"}
	}
	blocklists := make([]BlocklistEntryJSON, MaxImportBlocklistEntries)
	for i := range blocklists {
		blocklists[i] = BlocklistEntryJSON{ID: fmt.Sprintf("bl%d", i)}
	}
	cfg := &ProfileConfigJSON{
		Denylist:  denylist,
		Allowlist: allowlist,
		Rewrites:  rewrites,
		Privacy:   &PrivacyJSON{Blocklists: blocklists},
	}
	assert.NoError(t, Validate(cfg))
}
