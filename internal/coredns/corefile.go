// Package coredns provides utilities for generating CoreDNS Corefile configurations
// for use with NextDNS profiles.
package coredns

import (
	"fmt"
	"strings"
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

// DomainOverrideConfig represents a domain-specific upstream configuration
type DomainOverrideConfig struct {
	Domain    string
	Upstreams []string
	CacheTTL  int32 // 0 means use default (30 seconds)
}

// CorefileConfig holds the configuration for generating a CoreDNS Corefile.
type CorefileConfig struct {
	// ProfileID is the NextDNS profile ID to use for DNS resolution.
	ProfileID string

	// PrimaryProtocol specifies the primary DNS protocol (DoT, DoH, or DNS).
	PrimaryProtocol string

	// CacheTTL specifies the cache TTL in seconds.
	CacheTTL int32

	// LoggingEnabled controls whether the log plugin is enabled.
	LoggingEnabled bool

	// MetricsEnabled controls whether the prometheus plugin is enabled.
	MetricsEnabled bool

	// DomainOverrides specifies domain-specific upstream configurations
	DomainOverrides []DomainOverrideConfig
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

	// Generate forward plugin configuration
	writeForwardPlugin(&sb, cfg)

	// Cache plugin
	fmt.Fprintf(&sb, "    cache %d\n", cfg.CacheTTL)

	// Health plugin for liveness probes
	sb.WriteString("    health :8080\n")

	// Ready plugin for readiness probes
	sb.WriteString("    ready :8181\n")

	// Prometheus plugin for metrics (conditional)
	if cfg.MetricsEnabled {
		sb.WriteString("    prometheus :9153\n")
	}

	// Log plugin (conditional)
	if cfg.LoggingEnabled {
		sb.WriteString("    log\n")
	}

	// Errors plugin (always enabled)
	sb.WriteString("    errors\n")

	sb.WriteString("}")

	return sb.String()
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

// writeForwardPlugin writes the forward plugin configuration to the string builder.
// Note: Cross-protocol fallback (e.g., DoT→DoH) is not supported because CoreDNS's
// forward plugin cannot mix tls:// and https:// upstreams with a single tls_servername.
func writeForwardPlugin(sb *strings.Builder, cfg *CorefileConfig) {
	switch cfg.PrimaryProtocol {
	case ProtocolDoT:
		// DoT uses anycast IPs with TLS and tls_servername for SNI
		// The profile ID is embedded in the SNI hostname for NextDNS routing
		fmt.Fprintf(sb, "    forward . tls://%s tls://%s {\n", nextDNSAnycastIP1, nextDNSAnycastIP2)
		fmt.Fprintf(sb, "        tls_servername %s.%s\n", cfg.ProfileID, nextDNSDoTServer)
		sb.WriteString("    }\n")

	case ProtocolDoH:
		// DoH uses https:// URL directly
		upstream := fmt.Sprintf("https://%s/%s", nextDNSDoHServer, cfg.ProfileID)
		fmt.Fprintf(sb, "    forward . %s\n", upstream)

	case ProtocolDNS:
		// Plain DNS uses anycast IPs
		fmt.Fprintf(sb, "    forward . %s %s\n", nextDNSAnycastIP1, nextDNSAnycastIP2)
	}
}

// GetUpstreamEndpoint returns a human-readable endpoint string for the given
// protocol, suitable for use in status reporting.
func GetUpstreamEndpoint(profileID, protocol string) string {
	switch protocol {
	case ProtocolDoT:
		return fmt.Sprintf("tls://%s, tls://%s (SNI: %s.%s)", nextDNSAnycastIP1, nextDNSAnycastIP2, profileID, nextDNSDoTServer)
	case ProtocolDoH:
		return fmt.Sprintf("https://%s/%s", nextDNSDoHServer, profileID)
	case ProtocolDNS:
		return fmt.Sprintf("%s, %s", nextDNSAnycastIP1, nextDNSAnycastIP2)
	default:
		return ""
	}
}
