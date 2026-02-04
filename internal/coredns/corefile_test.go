package coredns

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Should contain forward with tls:// and tls_servername
	assert.Contains(t, corefile, "forward . tls://dns.nextdns.io")
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

func TestGenerateCorefile_WithFallback(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:        "jkl012",
		PrimaryProtocol:  ProtocolDoT,
		FallbackProtocol: ProtocolDNS,
		CacheTTL:         3600,
		LoggingEnabled:   true,
		MetricsEnabled:   true,
	}

	corefile := GenerateCorefile(cfg)

	// Should contain primary DoT configuration
	assert.Contains(t, corefile, "tls://dns.nextdns.io")
	assert.Contains(t, corefile, "tls_servername jkl012.dns.nextdns.io")

	// Should contain fallback DNS IPs
	assert.Contains(t, corefile, "45.90.28.0")
	assert.Contains(t, corefile, "45.90.30.0")

	// Primary should appear before fallback in the forward directive
	dotIndex := strings.Index(corefile, "tls://dns.nextdns.io")
	dnsIndex := strings.Index(corefile, "45.90.28.0")
	require.Greater(t, dotIndex, -1, "DoT endpoint should be present")
	require.Greater(t, dnsIndex, -1, "DNS endpoint should be present")
	assert.Less(t, dotIndex, dnsIndex, "Primary (DoT) should appear before fallback (DNS)")

	// Should contain all standard plugins
	assert.Contains(t, corefile, "cache 3600")
	assert.Contains(t, corefile, "health :8080")
	assert.Contains(t, corefile, "ready :8181")
	assert.Contains(t, corefile, "prometheus :9153")
	assert.Contains(t, corefile, "errors")
}

func TestGenerateCorefile_DoHWithDoTFallback(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:        "mno345",
		PrimaryProtocol:  ProtocolDoH,
		FallbackProtocol: ProtocolDoT,
		CacheTTL:         1800,
		LoggingEnabled:   false,
		MetricsEnabled:   true,
	}

	corefile := GenerateCorefile(cfg)

	// Should contain primary DoH configuration
	assert.Contains(t, corefile, "https://dns.nextdns.io/mno345")

	// Should contain fallback DoT configuration
	assert.Contains(t, corefile, "tls://dns.nextdns.io")
	assert.Contains(t, corefile, "tls_servername mno345.dns.nextdns.io")

	// Primary should appear before fallback
	dohIndex := strings.Index(corefile, "https://dns.nextdns.io/mno345")
	dotIndex := strings.Index(corefile, "tls://dns.nextdns.io")
	require.Greater(t, dohIndex, -1, "DoH endpoint should be present")
	require.Greater(t, dotIndex, -1, "DoT endpoint should be present")
	assert.Less(t, dohIndex, dotIndex, "Primary (DoH) should appear before fallback (DoT)")
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
	assert.Equal(t, "tls://abc123.dns.nextdns.io", endpoint)
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
			name: "DoT with DNS fallback",
			cfg: &CorefileConfig{
				ProfileID:        "test4",
				PrimaryProtocol:  ProtocolDoT,
				FallbackProtocol: ProtocolDNS,
				CacheTTL:         3600,
				LoggingEnabled:   true,
				MetricsEnabled:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corefile := GenerateCorefile(tt.cfg)

			// Basic structural validation
			assert.True(t, strings.HasPrefix(strings.TrimSpace(corefile), ". {"), "Corefile should start with '. {'")
			assert.True(t, strings.HasSuffix(strings.TrimSpace(corefile), "}"), "Corefile should end with '}'")

			// Count braces to ensure they're balanced
			openBraces := strings.Count(corefile, "{")
			closeBraces := strings.Count(corefile, "}")
			assert.Equal(t, openBraces, closeBraces, "Braces should be balanced")
		})
	}
}
