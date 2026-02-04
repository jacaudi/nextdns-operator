// Package coredns provides utilities for generating CoreDNS Corefile configurations
// for use with NextDNS profiles.
package coredns

import (
	"fmt"
	"strings"
)

// DefaultCoreDNSImage is the default CoreDNS container image to use.
const DefaultCoreDNSImage = "registry.k8s.io/coredns/coredns:1.11.1"

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

// CorefileConfig holds the configuration for generating a CoreDNS Corefile.
type CorefileConfig struct {
	// ProfileID is the NextDNS profile ID to use for DNS resolution.
	ProfileID string

	// PrimaryProtocol specifies the primary DNS protocol (DoT, DoH, or DNS).
	PrimaryProtocol string

	// FallbackProtocol specifies an optional fallback DNS protocol.
	// If empty, no fallback will be configured.
	FallbackProtocol string

	// CacheTTL specifies the cache TTL in seconds.
	CacheTTL int32

	// LoggingEnabled controls whether the log plugin is enabled.
	LoggingEnabled bool

	// MetricsEnabled controls whether the prometheus plugin is enabled.
	MetricsEnabled bool
}

// GenerateCorefile generates a CoreDNS Corefile configuration string
// based on the provided configuration.
func GenerateCorefile(cfg *CorefileConfig) string {
	var sb strings.Builder

	sb.WriteString(". {\n")

	// Generate forward plugin configuration
	writeForwardPlugin(&sb, cfg)

	// Cache plugin
	sb.WriteString(fmt.Sprintf("    cache %d\n", cfg.CacheTTL))

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

// writeForwardPlugin writes the forward plugin configuration to the string builder.
func writeForwardPlugin(sb *strings.Builder, cfg *CorefileConfig) {
	// Determine if we need a tls_servername block
	needsTLSServername := cfg.PrimaryProtocol == ProtocolDoT || cfg.FallbackProtocol == ProtocolDoT

	// Build the list of upstreams
	var upstreams []string

	// Add primary upstream
	primaryUpstream := getForwardUpstream(cfg.ProfileID, cfg.PrimaryProtocol)
	if primaryUpstream != "" {
		upstreams = append(upstreams, primaryUpstream)
	}

	// Add fallback upstream if specified
	if cfg.FallbackProtocol != "" {
		fallbackUpstream := getForwardUpstream(cfg.ProfileID, cfg.FallbackProtocol)
		if fallbackUpstream != "" {
			upstreams = append(upstreams, fallbackUpstream)
		}
	}

	// Write the forward directive
	if needsTLSServername {
		sb.WriteString(fmt.Sprintf("    forward . %s {\n", strings.Join(upstreams, " ")))
		sb.WriteString(fmt.Sprintf("        tls_servername %s.%s\n", cfg.ProfileID, nextDNSDoTServer))
		sb.WriteString("    }\n")
	} else {
		sb.WriteString(fmt.Sprintf("    forward . %s\n", strings.Join(upstreams, " ")))
	}
}

// getForwardUpstream returns the forward upstream string for a given protocol.
func getForwardUpstream(profileID, protocol string) string {
	switch protocol {
	case ProtocolDoT:
		return fmt.Sprintf("tls://%s", nextDNSDoTServer)
	case ProtocolDoH:
		return fmt.Sprintf("https://%s/%s", nextDNSDoHServer, profileID)
	case ProtocolDNS:
		return fmt.Sprintf("%s %s", nextDNSAnycastIP1, nextDNSAnycastIP2)
	default:
		return ""
	}
}

// GetUpstreamEndpoint returns a human-readable endpoint string for the given
// protocol, suitable for use in status reporting.
func GetUpstreamEndpoint(profileID, protocol string) string {
	switch protocol {
	case ProtocolDoT:
		return fmt.Sprintf("tls://%s.%s", profileID, nextDNSDoTServer)
	case ProtocolDoH:
		return fmt.Sprintf("https://%s/%s", nextDNSDoHServer, profileID)
	case ProtocolDNS:
		return fmt.Sprintf("%s, %s", nextDNSAnycastIP1, nextDNSAnycastIP2)
	default:
		return ""
	}
}
