package coredns

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCorefile_DoTPrimary(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		LoggingEnabled:  false,
		MetricsEnabled:  true,
	}

	corefile := GenerateCorefile(cfg)

	// Should contain forward with tls:// IPs and tls_servername with profile ID
	assert.Contains(t, corefile, "forward . tls://45.90.28.0 tls://45.90.30.0")
	assert.Contains(t, corefile, "tls_servername abc123.dns.nextdns.io")

	// Should contain cache with TTL
	assert.Contains(t, corefile, "cache 3600")

	// Should contain health and ready plugins
	assert.Contains(t, corefile, "health :8080")
	assert.Contains(t, corefile, "ready :8181")

	// Should contain prometheus (metrics enabled)
	assert.Contains(t, corefile, "prometheus :9153")

	// Should NOT contain log (logging disabled)
	// We need to check that "log" as a standalone plugin is not present
	// but "log" might appear in other contexts, so check more carefully
	lines := strings.Split(corefile, "\n")
	hasLogPlugin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "log" {
			hasLogPlugin = true
			break
		}
	}
	assert.False(t, hasLogPlugin, "log plugin should not be present when logging is disabled")

	// Should always contain errors plugin
	assert.Contains(t, corefile, "errors")
}

func TestGenerateCorefile_DoHPrimary(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "def456",
		PrimaryProtocol: ProtocolDoH,
		CacheTTL:        1800,
		LoggingEnabled:  true,
		MetricsEnabled:  true,
	}

	corefile := GenerateCorefile(cfg)

	// Should contain forward with https:// URL including profile ID
	assert.Contains(t, corefile, "forward . https://dns.nextdns.io/def456")

	// Should NOT contain tls_servername (not needed for DoH)
	assert.NotContains(t, corefile, "tls_servername")

	// Should contain cache with TTL
	assert.Contains(t, corefile, "cache 1800")

	// Should contain log (logging enabled)
	lines := strings.Split(corefile, "\n")
	hasLogPlugin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "log" {
			hasLogPlugin = true
			break
		}
	}
	assert.True(t, hasLogPlugin, "log plugin should be present when logging is enabled")

	// Should contain prometheus
	assert.Contains(t, corefile, "prometheus :9153")

	// Should always contain errors plugin
	assert.Contains(t, corefile, "errors")
}

func TestGenerateCorefile_DNSPrimary(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "ghi789",
		PrimaryProtocol: ProtocolDNS,
		CacheTTL:        600,
		LoggingEnabled:  false,
		MetricsEnabled:  true,
	}

	corefile := GenerateCorefile(cfg)

	// Should contain forward with NextDNS anycast IPs
	assert.Contains(t, corefile, "forward . 45.90.28.0 45.90.30.0")

	// Should NOT contain tls:// or https://
	assert.NotContains(t, corefile, "tls://")
	assert.NotContains(t, corefile, "https://")

	// Should NOT contain tls_servername
	assert.NotContains(t, corefile, "tls_servername")

	// Should contain cache with TTL
	assert.Contains(t, corefile, "cache 600")

	// Should contain health and ready plugins
	assert.Contains(t, corefile, "health :8080")
	assert.Contains(t, corefile, "ready :8181")

	// Should always contain errors plugin
	assert.Contains(t, corefile, "errors")
}

func TestGenerateCorefile_MetricsDisabled(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "pqr678",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		LoggingEnabled:  true,
		MetricsEnabled:  false,
	}

	corefile := GenerateCorefile(cfg)

	// Should NOT contain prometheus plugin
	assert.NotContains(t, corefile, "prometheus")

	// Should still contain other required plugins
	assert.Contains(t, corefile, "forward")
	assert.Contains(t, corefile, "cache 3600")
	assert.Contains(t, corefile, "health :8080")
	assert.Contains(t, corefile, "ready :8181")
	assert.Contains(t, corefile, "errors")

	// Should contain log since logging is enabled
	lines := strings.Split(corefile, "\n")
	hasLogPlugin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "log" {
			hasLogPlugin = true
			break
		}
	}
	assert.True(t, hasLogPlugin, "log plugin should be present when logging is enabled")
}

func TestGenerateCorefile_ZeroCacheTTL(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "stu901",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        0,
		LoggingEnabled:  false,
		MetricsEnabled:  true,
	}

	corefile := GenerateCorefile(cfg)

	// With zero TTL, cache should still be present but with 0
	// or could be omitted - let's test that cache 0 is present
	assert.Contains(t, corefile, "cache 0")
}

func TestGetUpstreamEndpoint_DoT(t *testing.T) {
	endpoint := GetUpstreamEndpoint("abc123", ProtocolDoT)
	assert.Equal(t, "tls://45.90.28.0, tls://45.90.30.0 (SNI: abc123.dns.nextdns.io)", endpoint)
}

func TestGetUpstreamEndpoint_DoH(t *testing.T) {
	endpoint := GetUpstreamEndpoint("def456", ProtocolDoH)
	assert.Equal(t, "https://dns.nextdns.io/def456", endpoint)
}

func TestGetUpstreamEndpoint_DNS(t *testing.T) {
	endpoint := GetUpstreamEndpoint("ghi789", ProtocolDNS)
	assert.Equal(t, "45.90.28.0, 45.90.30.0", endpoint)
}

func TestGetUpstreamEndpoint_UnknownProtocol(t *testing.T) {
	endpoint := GetUpstreamEndpoint("xyz", "UNKNOWN")
	// Should return empty string or some default for unknown protocols
	assert.Empty(t, endpoint)
}

func TestDefaultCoreDNSImage(t *testing.T) {
	assert.Equal(t, "mirror.gcr.io/coredns/coredns:1.13.1", DefaultCoreDNSImage)
}

func TestCorefileConfig_Defaults(t *testing.T) {
	// Test that zero-value config doesn't panic
	cfg := &CorefileConfig{
		ProfileID:       "test",
		PrimaryProtocol: ProtocolDoT,
	}

	corefile := GenerateCorefile(cfg)
	assert.NotEmpty(t, corefile)
	assert.Contains(t, corefile, "forward")
}

func TestGenerateCorefile_WithDomainOverrides(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		LoggingEnabled:  false,
		MetricsEnabled:  true,
		DomainOverrides: []DomainOverrideConfig{
			{
				Domain:    "corp.example.com",
				Upstreams: []string{"10.0.0.1", "10.0.0.2"},
				CacheTTL:  60,
			},
		},
	}

	corefile := GenerateCorefile(cfg)

	assert.Contains(t, corefile, "corp.example.com {")
	assert.Contains(t, corefile, "forward . 10.0.0.1 10.0.0.2")
	assert.Contains(t, corefile, "cache 60")

	corpIndex := strings.Index(corefile, "corp.example.com {")
	catchAllIndex := strings.Index(corefile, ". {")
	assert.True(t, corpIndex < catchAllIndex, "Domain override should come before catch-all block")

	assert.Contains(t, corefile, "tls://45.90.28.0")
	assert.Contains(t, corefile, "tls_servername abc123.dns.nextdns.io")
}

func TestGenerateCorefile_MultipleDomainOverrides(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "xyz789",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		MetricsEnabled:  true,
		DomainOverrides: []DomainOverrideConfig{
			{
				Domain:    "internal.local",
				Upstreams: []string{"192.168.1.1"},
			},
			{
				Domain:    "corp.example.com",
				Upstreams: []string{"10.0.0.1", "10.0.0.2"},
				CacheTTL:  120,
			},
		},
	}

	corefile := GenerateCorefile(cfg)

	assert.Contains(t, corefile, "internal.local {")
	assert.Contains(t, corefile, "corp.example.com {")

	internalIndex := strings.Index(corefile, "internal.local {")
	internalEnd := strings.Index(corefile[internalIndex:], "}") + internalIndex
	internalBlock := corefile[internalIndex:internalEnd]
	assert.Contains(t, internalBlock, "cache 30")

	corpIndex := strings.Index(corefile, "corp.example.com {")
	corpEnd := strings.Index(corefile[corpIndex:], "}") + corpIndex
	corpBlock := corefile[corpIndex:corpEnd]
	assert.Contains(t, corpBlock, "cache 120")
}

func TestGenerateCorefile_NoDomainOverrides(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "test123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		MetricsEnabled:  true,
		DomainOverrides: nil,
	}

	corefile := GenerateCorefile(cfg)

	assert.True(t, strings.HasPrefix(strings.TrimSpace(corefile), ". {"),
		"Without overrides, corefile should start with catch-all block")
	assert.Contains(t, corefile, "forward . tls://45.90.28.0")

	catchAllIndex := strings.Index(corefile, ". {")
	preamble := corefile[:catchAllIndex]
	assert.Empty(t, strings.TrimSpace(preamble),
		"Nothing should appear before the catch-all block when there are no overrides")
}

func TestValidateDomainOverrides_DuplicateDomains(t *testing.T) {
	overrides := []DomainOverrideConfig{
		{Domain: "corp.example.com", Upstreams: []string{"10.0.0.1"}},
		{Domain: "corp.example.com", Upstreams: []string{"10.0.0.2"}},
	}

	err := ValidateDomainOverrides(overrides)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate domain override")
}

func TestValidateDomainOverrides_Valid(t *testing.T) {
	overrides := []DomainOverrideConfig{
		{Domain: "corp.example.com", Upstreams: []string{"10.0.0.1"}},
		{Domain: "internal.local", Upstreams: []string{"192.168.1.1"}},
	}

	err := ValidateDomainOverrides(overrides)
	assert.NoError(t, err)
}

func TestValidateDomainOverrides_Empty(t *testing.T) {
	err := ValidateDomainOverrides(nil)
	assert.NoError(t, err)
}

func TestGenerateCorefile_ValidCorefileSyntax(t *testing.T) {
	tests := []struct {
		name string
		cfg  *CorefileConfig
	}{
		{
			name: "DoT only",
			cfg: &CorefileConfig{
				ProfileID:       "test1",
				PrimaryProtocol: ProtocolDoT,
				CacheTTL:        3600,
				LoggingEnabled:  true,
				MetricsEnabled:  true,
			},
		},
		{
			name: "DoH only",
			cfg: &CorefileConfig{
				ProfileID:       "test2",
				PrimaryProtocol: ProtocolDoH,
				CacheTTL:        1800,
				LoggingEnabled:  false,
				MetricsEnabled:  true,
			},
		},
		{
			name: "DNS only",
			cfg: &CorefileConfig{
				ProfileID:       "test3",
				PrimaryProtocol: ProtocolDNS,
				CacheTTL:        600,
				LoggingEnabled:  true,
				MetricsEnabled:  false,
			},
		},
		{
			name: "with domain overrides",
			cfg: &CorefileConfig{
				ProfileID:       "test4",
				PrimaryProtocol: ProtocolDoT,
				CacheTTL:        3600,
				MetricsEnabled:  true,
				DomainOverrides: []DomainOverrideConfig{
					{Domain: "corp.example.com", Upstreams: []string{"10.0.0.1"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corefile := GenerateCorefile(tt.cfg)

			assert.Contains(t, corefile, ". {", "Corefile should contain catch-all block")
			assert.True(t, strings.HasSuffix(strings.TrimSpace(corefile), "}"), "Corefile should end with '}'")

			openBraces := strings.Count(corefile, "{")
			closeBraces := strings.Count(corefile, "}")
			assert.Equal(t, openBraces, closeBraces, "Braces should be balanced")
		})
	}
}
