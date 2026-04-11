// Package coredns provides utilities for generating CoreDNS Corefile configurations
// for use with NextDNS profiles.
package coredns

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// DefaultCoreDNSImage is the default CoreDNS container image to use.
const DefaultCoreDNSImage = "mirror.gcr.io/coredns/coredns:1.13.1"

// Protocol constants for DNS resolution methods.
const (
	ProtocolDoT = "DoT" // DNS-over-TLS
	ProtocolDoH = "DoH" // DNS-over-HTTPS
	ProtocolDNS = "DNS" // Plain DNS (UDP/TCP)
)

// NextDNS server endpoints.
const (
	nextDNSDoTServer  = "dns.nextdns.io"
	nextDNSDoHServer  = "dns.nextdns.io"
	nextDNSAnycastIP1 = "45.90.28.0"
	nextDNSAnycastIP2 = "45.90.30.0"
)

// Default plugin listen ports. These preserve the pre-feature hardcoded
// behavior when the corresponding config pointer is nil or Port is 0.
const (
	defaultHealthPort  int32 = 8080
	defaultReadyPort   int32 = 8181
	defaultMetricsPort int32 = 9153
)

// ForwardTuningConfig holds per-deployment forward plugin tuning options.
// All fields optional; zero values mean "use CoreDNS default".
type ForwardTuningConfig struct {
	Policy        string // random, round_robin, sequential
	MaxConcurrent *int32
	HealthCheck   string // duration string (e.g. "5s")
	Expire        string // duration string
	MaxFails      *int32
}

// ValidateForwardTuning checks that policy is one of the supported
// values and durations parse cleanly. Empty / nil fields are skipped.
func ValidateForwardTuning(t *ForwardTuningConfig) error {
	if t == nil {
		return nil
	}
	var errs []string
	validPolicies := map[string]bool{
		"random": true, "round_robin": true, "sequential": true,
	}
	if t.Policy != "" && !validPolicies[t.Policy] {
		errs = append(errs, fmt.Sprintf("invalid forward policy %q", t.Policy))
	}
	if t.HealthCheck != "" {
		if _, err := time.ParseDuration(t.HealthCheck); err != nil {
			errs = append(errs, fmt.Sprintf("invalid healthCheck duration %q: %v", t.HealthCheck, err))
		}
	}
	if t.Expire != "" {
		if _, err := time.ParseDuration(t.Expire); err != nil {
			errs = append(errs, fmt.Sprintf("invalid expire duration %q: %v", t.Expire, err))
		}
	}
	if t.MaxConcurrent != nil && *t.MaxConcurrent < 1 {
		errs = append(errs, fmt.Sprintf("maxConcurrent must be >= 1, got %d", *t.MaxConcurrent))
	}
	if t.MaxFails != nil && *t.MaxFails < 0 {
		errs = append(errs, fmt.Sprintf("maxFails must be >= 0, got %d", *t.MaxFails))
	}
	if len(errs) > 0 {
		return fmt.Errorf("forward tuning validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// HostsEntryConfig is a single IP-to-hostnames mapping for the hosts plugin.
type HostsEntryConfig struct {
	IP        string
	Hostnames []string
}

// HostsPluginConfig configures the CoreDNS hosts plugin block.
type HostsPluginConfig struct {
	Entries     []HostsEntryConfig
	Fallthrough bool  // emit fallthrough directive
	TTL         int32 // 0 means omit (use CoreDNS default)
}

// ValidateHostsEntries checks that each entry has a parseable IP and at
// least one hostname. Returns an error describing all validation failures.
func ValidateHostsEntries(entries []HostsEntryConfig) error {
	var errs []string
	for i, e := range entries {
		if e.IP == "" {
			errs = append(errs, fmt.Sprintf("hosts entry %d: ip is required", i))
		} else if net.ParseIP(e.IP) == nil {
			errs = append(errs, fmt.Sprintf("hosts entry %d: invalid ip %q", i, e.IP))
		}
		if len(e.Hostnames) == 0 {
			errs = append(errs, fmt.Sprintf("hosts entry %d: at least one hostname required", i))
		}
		for j, h := range e.Hostnames {
			if h == "" {
				errs = append(errs, fmt.Sprintf("hosts entry %d hostname %d: empty hostname", i, j))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("hosts validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// DomainOverrideConfig represents a domain-specific upstream configuration
type DomainOverrideConfig struct {
	Domain    string
	Upstreams []string
	CacheTTL  int32 // 0 means use default (30 seconds)
}

// RewriteRuleConfig represents a single CoreDNS rewrite plugin rule.
type RewriteRuleConfig struct {
	Type        string // name, class, type, ttl, edns0
	Match       string
	Replacement string
	Matcher     string // optional: exact, prefix, suffix, substring, regex (only for type=name)
}

// ValidateRewriteRules checks that name rewrites have non-empty match
// and replacement, and that the matcher (if set) is one of the supported
// values.
func ValidateRewriteRules(rules []RewriteRuleConfig) error {
	var errs []string
	validMatchers := map[string]bool{
		"exact": true, "prefix": true, "suffix": true, "substring": true, "regex": true,
	}
	for i, r := range rules {
		if r.Type == "" {
			errs = append(errs, fmt.Sprintf("rewrite rule %d: type is required", i))
			continue
		}
		if r.Match == "" {
			errs = append(errs, fmt.Sprintf("rewrite rule %d: match is required", i))
		}
		if r.Replacement == "" {
			errs = append(errs, fmt.Sprintf("rewrite rule %d: replacement is required", i))
		}
		if r.Matcher != "" && !validMatchers[r.Matcher] {
			errs = append(errs, fmt.Sprintf("rewrite rule %d: invalid matcher %q", i, r.Matcher))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("rewrite rule validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// HealthPluginConfig configures the CoreDNS health plugin.
// A nil *HealthPluginConfig means "use defaults (enabled on port 8080, no lameduck)".
type HealthPluginConfig struct {
	Enabled  bool
	Port     int32  // 0 means use default 8080
	Lameduck string // empty means omit the lameduck directive
}

// ReadyPluginConfig configures the CoreDNS ready plugin.
// A nil *ReadyPluginConfig means "use defaults (enabled on port 8181)".
type ReadyPluginConfig struct {
	Enabled bool
	Port    int32 // 0 means use default 8181
}

// ConsolidateRuleConfig is a single consolidate directive for the errors plugin.
type ConsolidateRuleConfig struct {
	Interval string
	Pattern  string
}

// ErrorsPluginConfig configures the CoreDNS errors plugin.
// A nil *ErrorsPluginConfig means "use defaults (enabled, no consolidate rules)".
type ErrorsPluginConfig struct {
	Enabled     bool
	Consolidate []ConsolidateRuleConfig
}

// CorefileConfig holds the configuration for generating a CoreDNS Corefile.
type CorefileConfig struct {
	// ProfileID is the NextDNS profile ID to use for DNS resolution.
	ProfileID string

	// PrimaryProtocol specifies the primary DNS protocol (DoT, DoH, or DNS).
	PrimaryProtocol string

	// DeviceName is an optional device identifier for NextDNS analytics.
	DeviceName string

	// CacheTTL specifies the cache TTL in seconds.
	CacheTTL int32

	// LoggingEnabled controls whether the log plugin is enabled.
	LoggingEnabled bool

	// MetricsEnabled controls whether the prometheus plugin is enabled.
	MetricsEnabled bool

	// DomainOverrides specifies domain-specific upstream configurations
	DomainOverrides []DomainOverrideConfig

	// UpstreamIPv4 contains profile-specific IPv4 addresses for DoT/DNS forwarding.
	// Falls back to anycast IPs (45.90.28.0, 45.90.30.0) if empty.
	UpstreamIPv4 []string

	// RewriteRules specifies CoreDNS rewrite plugin rules to emit before the
	// forward directive in the catch-all server block.
	RewriteRules []RewriteRuleConfig

	// ForwardTuning optionally configures forward plugin tuning options.
	// When nil, CoreDNS defaults apply and forward block shape is unchanged.
	ForwardTuning *ForwardTuningConfig

	// Hosts configures the CoreDNS hosts plugin for inline static overrides.
	// When set, a hosts block is emitted before the forward plugin.
	Hosts *HostsPluginConfig

	// Health configures the CoreDNS health plugin. nil means "use defaults
	// (enabled on port 8080, no lameduck)" so the generated output is
	// byte-identical to the pre-feature behavior.
	Health *HealthPluginConfig

	// Ready configures the CoreDNS ready plugin. nil means "use defaults
	// (enabled on port 8181)".
	Ready *ReadyPluginConfig

	// Errors configures the CoreDNS errors plugin. nil means "use defaults
	// (enabled, no consolidate rules)".
	Errors *ErrorsPluginConfig

	// MetricsPort overrides the prometheus plugin port. 0 means default 9153.
	// Only honored when MetricsEnabled is true.
	MetricsPort int32
}

// ValidateDomainOverrides checks for duplicate domains and invalid upstream values.
// Returns an error describing all validation failures.
func ValidateDomainOverrides(overrides []DomainOverrideConfig) error {
	seen := make(map[string]bool, len(overrides))
	var errs []string
	for _, o := range overrides {
		if seen[o.Domain] {
			errs = append(errs, fmt.Sprintf("duplicate domain override: %s", o.Domain))
		}
		seen[o.Domain] = true
		for _, u := range o.Upstreams {
			if u == "" {
				errs = append(errs, fmt.Sprintf("empty upstream for domain %s", o.Domain))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("domain override validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// GenerateCorefile generates a CoreDNS Corefile configuration string
// based on the provided configuration.
func GenerateCorefile(cfg *CorefileConfig) string {
	var sb strings.Builder

	// Generate domain override blocks first (order matters in CoreDNS)
	for _, override := range cfg.DomainOverrides {
		writeDomainOverrideBlock(&sb, &override)
	}

	// Generate the catch-all block for NextDNS
	sb.WriteString(". {\n")

	// Rewrite directives fire first so the (possibly rewritten) query is
	// matched by hosts and then forwarded (CoreDNS plugin order matters).
	writeRewriteRules(&sb, cfg.RewriteRules)

	// Hosts block (before forward, so static entries resolve without hitting NextDNS)
	writeHostsBlock(&sb, cfg.Hosts)

	// Generate forward plugin configuration
	writeForwardPlugin(&sb, cfg)

	// Cache plugin
	fmt.Fprintf(&sb, "    cache %d\n", cfg.CacheTTL)

	// Health plugin for liveness probes (configurable port + optional lameduck)
	writeHealthBlock(&sb, cfg.Health)

	// Ready plugin for readiness probes (configurable port, can be disabled)
	writeReadyBlock(&sb, cfg.Ready)

	// Prometheus plugin for metrics (conditional, configurable port)
	if cfg.MetricsEnabled {
		mPort := cfg.MetricsPort
		if mPort == 0 {
			mPort = defaultMetricsPort
		}
		fmt.Fprintf(&sb, "    prometheus :%d\n", mPort)
	}

	// Log plugin (conditional)
	if cfg.LoggingEnabled {
		sb.WriteString("    log\n")
	}

	// Errors plugin (configurable, may include consolidate rules)
	writeErrorsBlock(&sb, cfg.Errors)

	sb.WriteString("}")

	return sb.String()
}

// writeRewriteRules writes rewrite directives to the string builder.
// Rules are emitted in order; those with a matcher use the four-argument form.
func writeRewriteRules(sb *strings.Builder, rules []RewriteRuleConfig) {
	for _, r := range rules {
		if r.Matcher != "" {
			fmt.Fprintf(sb, "    rewrite %s %s %s %s\n", r.Type, r.Matcher, r.Match, r.Replacement)
		} else {
			fmt.Fprintf(sb, "    rewrite %s %s %s\n", r.Type, r.Match, r.Replacement)
		}
	}
}

// writeDomainOverrideBlock writes a domain-specific server block.
// Override blocks intentionally only include forward, cache, and errors.
// Plugins like health, ready, prometheus, and log are omitted because they
// only need to be configured once in the catch-all block — CoreDNS applies
// them process-wide from there.
func writeDomainOverrideBlock(sb *strings.Builder, override *DomainOverrideConfig) {
	fmt.Fprintf(sb, "%s {\n", override.Domain)

	// Build upstream list
	upstreams := strings.Join(override.Upstreams, " ")
	fmt.Fprintf(sb, "    forward . %s\n", upstreams)

	// Cache with override-specific TTL or default
	cacheTTL := override.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 30 // default for overrides
	}
	fmt.Fprintf(sb, "    cache %d\n", cacheTTL)

	sb.WriteString("    errors\n")
	sb.WriteString("}\n\n")
}

// writeHostsBlock writes a CoreDNS hosts plugin block if hosts is non-nil and
// has at least one entry. The block is written before the forward plugin so
// static entries resolve without hitting NextDNS.
func writeHostsBlock(sb *strings.Builder, hosts *HostsPluginConfig) {
	if hosts == nil || len(hosts.Entries) == 0 {
		return
	}
	sb.WriteString("    hosts {\n")
	for _, e := range hosts.Entries {
		fmt.Fprintf(sb, "        %s %s\n", e.IP, strings.Join(e.Hostnames, " "))
	}
	if hosts.TTL > 0 {
		fmt.Fprintf(sb, "        ttl %d\n", hosts.TTL)
	}
	if hosts.Fallthrough {
		sb.WriteString("        fallthrough\n")
	}
	sb.WriteString("    }\n")
}

// writeHealthBlock writes the health plugin directive. A nil config or
// Enabled=false omits the directive entirely. The lameduck directive is
// emitted inside a block when set; otherwise the directive is a single line.
//
// Backward compatibility: cfg.Health == nil produces "    health :8080\n"
// which exactly matches the pre-feature output.
func writeHealthBlock(sb *strings.Builder, h *HealthPluginConfig) {
	enabled := true
	port := defaultHealthPort
	lameduck := ""
	if h != nil {
		enabled = h.Enabled
		if h.Port != 0 {
			port = h.Port
		}
		lameduck = h.Lameduck
	}
	if !enabled {
		return
	}
	if lameduck != "" {
		fmt.Fprintf(sb, "    health :%d {\n", port)
		fmt.Fprintf(sb, "        lameduck %s\n", lameduck)
		sb.WriteString("    }\n")
		return
	}
	fmt.Fprintf(sb, "    health :%d\n", port)
}

// writeReadyBlock writes the ready plugin directive. A nil config or
// Enabled=false omits the directive. nil produces "    ready :8181\n" —
// the pre-feature default.
func writeReadyBlock(sb *strings.Builder, r *ReadyPluginConfig) {
	enabled := true
	port := defaultReadyPort
	if r != nil {
		enabled = r.Enabled
		if r.Port != 0 {
			port = r.Port
		}
	}
	if !enabled {
		return
	}
	fmt.Fprintf(sb, "    ready :%d\n", port)
}

// writeErrorsBlock writes the errors plugin directive. A nil config produces
// a bare "    errors\n" line (pre-feature default). Enabled=false omits the
// directive entirely. When consolidate rules are present, the directive is
// emitted as a block with one consolidate line per rule.
func writeErrorsBlock(sb *strings.Builder, e *ErrorsPluginConfig) {
	enabled := true
	var consolidate []ConsolidateRuleConfig
	if e != nil {
		enabled = e.Enabled
		consolidate = e.Consolidate
	}
	if !enabled {
		return
	}
	if len(consolidate) == 0 {
		sb.WriteString("    errors\n")
		return
	}
	sb.WriteString("    errors {\n")
	for _, c := range consolidate {
		fmt.Fprintf(sb, "        consolidate %s %q\n", c.Interval, c.Pattern)
	}
	sb.WriteString("    }\n")
}

// ValidatePluginConfig checks that configured plugin ports are distinct,
// within the 1-65535 TCP range, and that durations parse cleanly. Pass
// metricsPort=0 to mean "use the 9153 default".
//
// Collision checks only apply to plugins that are actually enabled — a
// disabled plugin with a colliding port is allowed (the directive is
// never emitted).
func ValidatePluginConfig(health *HealthPluginConfig, ready *ReadyPluginConfig, errors *ErrorsPluginConfig, metricsPort int32) error {
	var errs []string

	healthPort := defaultHealthPort
	if health != nil && health.Port != 0 {
		healthPort = health.Port
	}
	readyPort := defaultReadyPort
	if ready != nil && ready.Port != 0 {
		readyPort = ready.Port
	}
	mPort := defaultMetricsPort
	if metricsPort != 0 {
		mPort = metricsPort
	}

	// Range checks (kubebuilder enforces at the API boundary; this is
	// defensive for internal callers that bypass the API).
	// Iterate in a stable order so error messages are deterministic.
	portLabels := []struct {
		label string
		port  int32
	}{
		{"health", healthPort},
		{"ready", readyPort},
		{"metrics", mPort},
	}
	for _, pl := range portLabels {
		if pl.port < 1 || pl.port > 65535 {
			errs = append(errs, fmt.Sprintf("%s port %d out of range 1-65535", pl.label, pl.port))
		}
	}

	// Collision checks (only meaningful for enabled plugins).
	healthOn := health == nil || health.Enabled
	readyOn := ready == nil || ready.Enabled
	if healthOn && readyOn && healthPort == readyPort {
		errs = append(errs, fmt.Sprintf("health and ready ports must differ (both %d)", healthPort))
	}
	if healthOn && healthPort == mPort {
		errs = append(errs, fmt.Sprintf("health and metrics ports must differ (both %d)", healthPort))
	}
	if readyOn && readyPort == mPort {
		errs = append(errs, fmt.Sprintf("ready and metrics ports must differ (both %d)", readyPort))
	}

	// Duration parsing.
	if health != nil && health.Lameduck != "" {
		if _, err := time.ParseDuration(health.Lameduck); err != nil {
			errs = append(errs, fmt.Sprintf("invalid health.lameduck duration %q: %v", health.Lameduck, err))
		}
	}
	if errors != nil {
		for i, c := range errors.Consolidate {
			if c.Interval == "" || c.Pattern == "" {
				errs = append(errs, fmt.Sprintf("errors.consolidate[%d]: interval and pattern are both required", i))
				continue
			}
			if _, err := time.ParseDuration(c.Interval); err != nil {
				errs = append(errs, fmt.Sprintf("errors.consolidate[%d]: invalid interval %q: %v", i, c.Interval, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("plugin config validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// formatDeviceNameDoT converts a device name for DoT SNI (spaces become --)
func formatDeviceNameDoT(name string) string {
	return strings.ReplaceAll(name, " ", "--")
}

// buildDoTSNIHost returns the SNI hostname for DoT, with optional device name prefix.
func buildDoTSNIHost(profileID, deviceName string) string {
	if deviceName != "" {
		return formatDeviceNameDoT(deviceName) + "-" + profileID
	}
	return profileID
}

// buildDoHPath returns the URL path segment for DoH, with optional device name suffix.
func buildDoHPath(profileID, deviceName string) string {
	if deviceName != "" {
		return profileID + "/" + url.PathEscape(deviceName)
	}
	return profileID
}

// writeForwardTuning writes forward plugin tuning directives inside a forward block.
// It is a no-op when t is nil.
func writeForwardTuning(sb *strings.Builder, t *ForwardTuningConfig) {
	if t == nil {
		return
	}
	if t.Policy != "" {
		fmt.Fprintf(sb, "        policy %s\n", t.Policy)
	}
	if t.MaxConcurrent != nil {
		fmt.Fprintf(sb, "        max_concurrent %d\n", *t.MaxConcurrent)
	}
	if t.HealthCheck != "" {
		fmt.Fprintf(sb, "        health_check %s\n", t.HealthCheck)
	}
	if t.Expire != "" {
		fmt.Fprintf(sb, "        expire %s\n", t.Expire)
	}
	if t.MaxFails != nil {
		fmt.Fprintf(sb, "        max_fails %d\n", *t.MaxFails)
	}
}

// writeForwardPlugin writes the forward plugin configuration to the string builder.
// Note: Cross-protocol fallback (e.g., DoT→DoH) is not supported because CoreDNS's
// forward plugin cannot mix tls:// and https:// upstreams with a single tls_servername.
func writeForwardPlugin(sb *strings.Builder, cfg *CorefileConfig) {
	ip1, ip2 := resolveUpstreamIPs(cfg.UpstreamIPv4)

	switch cfg.PrimaryProtocol {
	case ProtocolDoT:
		// DoT uses IPs with TLS and tls_servername for SNI
		// The profile ID is embedded in the SNI hostname for NextDNS routing
		fmt.Fprintf(sb, "    forward . tls://%s tls://%s {\n", ip1, ip2)
		fmt.Fprintf(sb, "        tls_servername %s.%s\n", buildDoTSNIHost(cfg.ProfileID, cfg.DeviceName), nextDNSDoTServer)
		writeForwardTuning(sb, cfg.ForwardTuning)
		sb.WriteString("    }\n")

	case ProtocolDoH:
		// DoH uses https:// URL directly
		upstream := fmt.Sprintf("https://%s/%s", nextDNSDoHServer, buildDoHPath(cfg.ProfileID, cfg.DeviceName))
		if cfg.ForwardTuning != nil {
			fmt.Fprintf(sb, "    forward . %s {\n", upstream)
			writeForwardTuning(sb, cfg.ForwardTuning)
			sb.WriteString("    }\n")
		} else {
			fmt.Fprintf(sb, "    forward . %s\n", upstream)
		}

	case ProtocolDNS:
		// Plain DNS uses upstream IPs
		if cfg.ForwardTuning != nil {
			fmt.Fprintf(sb, "    forward . %s %s {\n", ip1, ip2)
			writeForwardTuning(sb, cfg.ForwardTuning)
			sb.WriteString("    }\n")
		} else {
			fmt.Fprintf(sb, "    forward . %s %s\n", ip1, ip2)
		}
	}
}

// resolveUpstreamIPs returns two upstream IPs. Uses profile-specific IPs if
// available (at least 2), otherwise falls back to NextDNS anycast IPs.
func resolveUpstreamIPs(profileIPs []string) (string, string) {
	if len(profileIPs) >= 2 {
		return profileIPs[0], profileIPs[1]
	}
	return nextDNSAnycastIP1, nextDNSAnycastIP2
}

// GetUpstreamEndpoint returns a human-readable endpoint string for the given
// protocol, suitable for use in status reporting.
func GetUpstreamEndpoint(profileID, protocol, deviceName string, upstreamIPv4 []string) string {
	ip1, ip2 := resolveUpstreamIPs(upstreamIPv4)

	switch protocol {
	case ProtocolDoT:
		return fmt.Sprintf("tls://%s, tls://%s (SNI: %s.%s)", ip1, ip2, buildDoTSNIHost(profileID, deviceName), nextDNSDoTServer)
	case ProtocolDoH:
		return fmt.Sprintf("https://%s/%s", nextDNSDoHServer, buildDoHPath(profileID, deviceName))
	case ProtocolDNS:
		return fmt.Sprintf("%s, %s", ip1, ip2)
	default:
		return ""
	}
}
