package coredns

import (
	"fmt"
	"net/url"
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
	endpoint := GetUpstreamEndpoint("abc123", ProtocolDoT, "", nil)
	assert.Equal(t, "tls://45.90.28.0, tls://45.90.30.0 (SNI: abc123.dns.nextdns.io)", endpoint)
}

func TestGetUpstreamEndpoint_DoH(t *testing.T) {
	endpoint := GetUpstreamEndpoint("def456", ProtocolDoH, "", nil)
	assert.Equal(t, "https://dns.nextdns.io/def456", endpoint)
}

func TestGetUpstreamEndpoint_DNS(t *testing.T) {
	endpoint := GetUpstreamEndpoint("ghi789", ProtocolDNS, "", nil)
	assert.Equal(t, "45.90.28.0, 45.90.30.0", endpoint)
}

func TestGetUpstreamEndpoint_UnknownProtocol(t *testing.T) {
	endpoint := GetUpstreamEndpoint("xyz", "UNKNOWN", "", nil)
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

func TestGenerateCorefile_DoTWithDeviceName(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		DeviceName:      "Home Router",
		CacheTTL:        3600,
		MetricsEnabled:  true,
	}
	corefile := GenerateCorefile(cfg)
	assert.Contains(t, corefile, "tls_servername Home--Router-abc123.dns.nextdns.io")
}

func TestGenerateCorefile_DoHWithDeviceName(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoH,
		DeviceName:      "Home Router",
		CacheTTL:        3600,
		MetricsEnabled:  true,
	}
	corefile := GenerateCorefile(cfg)
	expected := fmt.Sprintf("https://dns.nextdns.io/abc123/%s", url.PathEscape("Home Router"))
	assert.Contains(t, corefile, expected)
}

func TestGenerateCorefile_DNSWithDeviceName(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDNS,
		DeviceName:      "Home Router",
		CacheTTL:        3600,
		MetricsEnabled:  true,
	}
	corefile := GenerateCorefile(cfg)
	// Plain DNS should NOT contain device name — no mechanism for it
	assert.NotContains(t, corefile, "Home")
	assert.Contains(t, corefile, "45.90.28.0")
}

func TestGetUpstreamEndpoint_DoTWithDeviceName(t *testing.T) {
	endpoint := GetUpstreamEndpoint("abc123", ProtocolDoT, "Home Router", nil)
	assert.Contains(t, endpoint, "Home--Router-abc123.dns.nextdns.io")
}

func TestGetUpstreamEndpoint_DoHWithDeviceName(t *testing.T) {
	endpoint := GetUpstreamEndpoint("abc123", ProtocolDoH, "Home Router", nil)
	assert.Contains(t, endpoint, "/abc123/Home%20Router")
}

func TestGetUpstreamEndpoint_DNSWithDeviceName(t *testing.T) {
	endpoint := GetUpstreamEndpoint("abc123", ProtocolDNS, "Home Router", nil)
	// Plain DNS ignores device name
	assert.NotContains(t, endpoint, "Home")
	assert.Equal(t, "45.90.28.0, 45.90.30.0", endpoint)
}

func TestBuildDoTSNIHost(t *testing.T) {
	tests := []struct {
		name       string
		profileID  string
		deviceName string
		expected   string
	}{
		{"no device name", "abc123", "", "abc123"},
		{"simple name", "abc123", "router", "router-abc123"},
		{"name with spaces", "abc123", "Home Router", "Home--Router-abc123"},
		{"name with hyphens", "abc123", "my-device", "my-device-abc123"},
		{"name with multiple spaces", "abc123", "My Home Router", "My--Home--Router-abc123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildDoTSNIHost(tt.profileID, tt.deviceName))
		})
	}
}

func TestBuildDoHPath(t *testing.T) {
	tests := []struct {
		name       string
		profileID  string
		deviceName string
		expected   string
	}{
		{"no device name", "abc123", "", "abc123"},
		{"simple name", "abc123", "router", "abc123/router"},
		{"name with spaces", "abc123", "Home Router", "abc123/Home%20Router"},
		{"name with hyphens", "abc123", "my-device", "abc123/my-device"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildDoHPath(tt.profileID, tt.deviceName))
		})
	}
}

func TestGenerateCorefile_DoT_ProfileSpecificIPs(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		UpstreamIPv4:    []string{"45.90.28.198", "45.90.30.198"},
	}

	result := GenerateCorefile(cfg)

	assert.Contains(t, result, "tls://45.90.28.198 tls://45.90.30.198")
	assert.NotContains(t, result, "tls://45.90.28.0")
	assert.Contains(t, result, "tls_servername abc123.dns.nextdns.io")
}

func TestGenerateCorefile_DNS_ProfileSpecificIPs(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDNS,
		UpstreamIPv4:    []string{"45.90.28.198", "45.90.30.198"},
	}

	result := GenerateCorefile(cfg)

	assert.Contains(t, result, "forward . 45.90.28.198 45.90.30.198")
	assert.NotContains(t, result, "45.90.28.0")
}

func TestGenerateCorefile_DoT_FallbackToAnycast(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
	}

	result := GenerateCorefile(cfg)

	assert.Contains(t, result, "tls://45.90.28.0 tls://45.90.30.0")
}

func TestGetUpstreamEndpoint_ProfileSpecificIPs(t *testing.T) {
	result := GetUpstreamEndpoint("abc123", ProtocolDoT, "", []string{"45.90.28.198", "45.90.30.198"})
	assert.Contains(t, result, "45.90.28.198")
	assert.NotContains(t, result, "45.90.28.0")
}

func TestGenerateCorefile_WithRewriteRules(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		MetricsEnabled:  true,
		RewriteRules: []RewriteRuleConfig{
			{Type: "name", Match: "service.example.com", Replacement: "ingress.cluster.local"},
			{Type: "name", Match: "old.example.com", Replacement: "new.example.com", Matcher: "suffix"},
		},
	}

	out := GenerateCorefile(cfg)

	if !strings.Contains(out, "rewrite name service.example.com ingress.cluster.local") {
		t.Errorf("expected rewrite directive for service.example.com; got:\n%s", out)
	}
	if !strings.Contains(out, "rewrite name suffix old.example.com new.example.com") {
		t.Errorf("expected suffix rewrite directive; got:\n%s", out)
	}

	// Rewrite must precede forward in the catch-all block
	rewriteIdx := strings.Index(out, "rewrite name")
	forwardIdx := strings.Index(out, "forward .")
	if rewriteIdx == -1 || forwardIdx == -1 || rewriteIdx > forwardIdx {
		t.Errorf("rewrite must appear before forward; rewriteIdx=%d forwardIdx=%d", rewriteIdx, forwardIdx)
	}
}

func TestValidateRewriteRules(t *testing.T) {
	tests := []struct {
		name    string
		rules   []RewriteRuleConfig
		wantErr bool
	}{
		{"valid name rule", []RewriteRuleConfig{{Type: "name", Match: "a.com", Replacement: "b.com"}}, false},
		{"valid name rule with matcher", []RewriteRuleConfig{{Type: "name", Matcher: "suffix", Match: ".old", Replacement: ".new"}}, false},
		{"missing type", []RewriteRuleConfig{{Match: "a.com", Replacement: "b.com"}}, true},
		{"missing match", []RewriteRuleConfig{{Type: "name", Replacement: "b.com"}}, true},
		{"missing replacement", []RewriteRuleConfig{{Type: "name", Match: "a.com"}}, true},
		{"invalid matcher", []RewriteRuleConfig{{Type: "name", Matcher: "bogus", Match: "a.com", Replacement: "b.com"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRewriteRules(tt.rules)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateCorefile_WithForwardTuning(t *testing.T) {
	maxConcurrent := int32(1000)
	maxFails := int32(2)
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		MetricsEnabled:  true,
		ForwardTuning: &ForwardTuningConfig{
			Policy:        "round_robin",
			MaxConcurrent: &maxConcurrent,
			HealthCheck:   "5s",
			Expire:        "30s",
			MaxFails:      &maxFails,
		},
	}

	out := GenerateCorefile(cfg)

	// Each tuning directive must appear inside the forward block
	if !strings.Contains(out, "policy round_robin") {
		t.Errorf("expected policy directive; got:\n%s", out)
	}
	if !strings.Contains(out, "max_concurrent 1000") {
		t.Errorf("expected max_concurrent directive; got:\n%s", out)
	}
	if !strings.Contains(out, "health_check 5s") {
		t.Errorf("expected health_check directive; got:\n%s", out)
	}
	if !strings.Contains(out, "expire 30s") {
		t.Errorf("expected expire directive; got:\n%s", out)
	}
	if !strings.Contains(out, "max_fails 2") {
		t.Errorf("expected max_fails directive; got:\n%s", out)
	}
}

func TestGenerateCorefile_WithoutForwardTuning_Unchanged(t *testing.T) {
	// Sanity check: the existing forward block must not change when
	// ForwardTuning is nil. This is a regression test against the refactor.
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		MetricsEnabled:  true,
	}
	out := GenerateCorefile(cfg)
	for _, forbidden := range []string{"policy ", "max_concurrent", "health_check", "expire", "max_fails"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("unexpected %q in default forward block:\n%s", forbidden, out)
		}
	}
}

func TestGenerateCorefile_WithForwardTuning_DoH(t *testing.T) {
	maxConcurrent := int32(500)
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoH,
		CacheTTL:        3600,
		MetricsEnabled:  true,
		ForwardTuning: &ForwardTuningConfig{
			Policy:        "sequential",
			MaxConcurrent: &maxConcurrent,
			HealthCheck:   "10s",
		},
	}

	out := GenerateCorefile(cfg)

	if !strings.Contains(out, "policy sequential") {
		t.Errorf("expected policy directive for DoH; got:\n%s", out)
	}
	if !strings.Contains(out, "max_concurrent 500") {
		t.Errorf("expected max_concurrent directive for DoH; got:\n%s", out)
	}
	if !strings.Contains(out, "health_check 10s") {
		t.Errorf("expected health_check directive for DoH; got:\n%s", out)
	}
	// Must still have the DoH upstream URL
	if !strings.Contains(out, "https://dns.nextdns.io/abc123") {
		t.Errorf("expected DoH upstream URL; got:\n%s", out)
	}
}

func TestGenerateCorefile_WithForwardTuning_DNS(t *testing.T) {
	maxFails := int32(3)
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDNS,
		CacheTTL:        3600,
		MetricsEnabled:  true,
		ForwardTuning: &ForwardTuningConfig{
			MaxFails: &maxFails,
			Expire:   "60s",
		},
	}

	out := GenerateCorefile(cfg)

	if !strings.Contains(out, "max_fails 3") {
		t.Errorf("expected max_fails directive for DNS; got:\n%s", out)
	}
	if !strings.Contains(out, "expire 60s") {
		t.Errorf("expected expire directive for DNS; got:\n%s", out)
	}
	// Must still have the plain DNS IPs
	if !strings.Contains(out, "45.90.28.0") {
		t.Errorf("expected anycast IP for DNS; got:\n%s", out)
	}
}

func TestValidateForwardTuning(t *testing.T) {
	mc := int32(1000)
	tests := []struct {
		name    string
		t       *ForwardTuningConfig
		wantErr bool
	}{
		{"nil", nil, false},
		{"empty", &ForwardTuningConfig{}, false},
		{"valid", &ForwardTuningConfig{Policy: "round_robin", HealthCheck: "5s", Expire: "30s", MaxConcurrent: &mc}, false},
		{"invalid policy", &ForwardTuningConfig{Policy: "bogus"}, true},
		{"bad healthCheck", &ForwardTuningConfig{HealthCheck: "5xs"}, true},
		{"bad expire", &ForwardTuningConfig{Expire: "thirty"}, true},
		{"maxConcurrent zero", &ForwardTuningConfig{MaxConcurrent: func() *int32 { v := int32(0); return &v }()}, true},
		{"maxFails negative", &ForwardTuningConfig{MaxFails: func() *int32 { v := int32(-1); return &v }()}, true},
		{"maxFails zero ok", &ForwardTuningConfig{MaxFails: func() *int32 { v := int32(0); return &v }()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateForwardTuning(tt.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateCorefile_WithHostsBlock(t *testing.T) {
	cfg := &CorefileConfig{
		ProfileID:       "abc123",
		PrimaryProtocol: ProtocolDoT,
		CacheTTL:        3600,
		MetricsEnabled:  true,
		Hosts: &HostsPluginConfig{
			Entries: []HostsEntryConfig{
				{IP: "192.168.1.100", Hostnames: []string{"grafana.internal", "grafana.example.com"}},
				{IP: "192.168.1.101", Hostnames: []string{"prometheus.internal"}},
			},
			Fallthrough: true,
			TTL:         3600,
		},
	}

	out := GenerateCorefile(cfg)

	// Hosts block must exist and contain entries
	if !strings.Contains(out, "hosts {") {
		t.Errorf("expected hosts block; got:\n%s", out)
	}
	if !strings.Contains(out, "192.168.1.100 grafana.internal grafana.example.com") {
		t.Errorf("expected first hosts entry; got:\n%s", out)
	}
	if !strings.Contains(out, "192.168.1.101 prometheus.internal") {
		t.Errorf("expected second hosts entry; got:\n%s", out)
	}
	if !strings.Contains(out, "fallthrough") {
		t.Errorf("expected fallthrough directive; got:\n%s", out)
	}
	if !strings.Contains(out, "ttl 3600") {
		t.Errorf("expected ttl directive; got:\n%s", out)
	}

	// Hosts block must precede forward
	hostsIdx := strings.Index(out, "hosts {")
	forwardIdx := strings.Index(out, "forward .")
	if hostsIdx == -1 || forwardIdx == -1 || hostsIdx > forwardIdx {
		t.Errorf("hosts block must precede forward; hostsIdx=%d forwardIdx=%d", hostsIdx, forwardIdx)
	}
}

func TestValidateHostsEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []HostsEntryConfig
		wantErr bool
	}{
		{"valid", []HostsEntryConfig{{IP: "192.168.1.1", Hostnames: []string{"a.local"}}}, false},
		{"valid IPv6", []HostsEntryConfig{{IP: "fe80::1", Hostnames: []string{"a.local"}}}, false},
		{"missing IP", []HostsEntryConfig{{Hostnames: []string{"a.local"}}}, true},
		{"invalid IP", []HostsEntryConfig{{IP: "not-an-ip", Hostnames: []string{"a.local"}}}, true},
		{"no hostnames", []HostsEntryConfig{{IP: "192.168.1.1"}}, true},
		{"empty hostname", []HostsEntryConfig{{IP: "192.168.1.1", Hostnames: []string{""}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostsEntries(tt.entries)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
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
