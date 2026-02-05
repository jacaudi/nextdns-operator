# Implementation Plan: Domain Override Support in NextDNSCoreDNS

**Issue:** [#41](https://github.com/jacaudi/nextdns-operator/issues/41) - feat: Add domain override support in NextDNSCoreDNS
**Author:** @jacaudi
**Status:** Planning

## Overview

This feature adds domain-specific DNS forwarding capabilities to the NextDNSCoreDNS resource. Users will be able to forward specific domains to designated upstream servers independently of the primary NextDNS configuration, enabling split-horizon DNS architectures.

---

## Use Cases

1. **Internal DNS Resolution**: Forward `corp.example.com` to internal DNS servers
2. **Split-Horizon DNS**: Different DNS servers for internal vs. external domains
3. **Local Development**: Override specific domains for local services
4. **Multi-Cloud Environments**: Route cloud-specific domains to respective DNS services

---

## Phase 1: API Type Changes

**File: `api/v1alpha1/nextdnscoredns_types.go`**

### 1. Add `DomainOverride` type:

```go
// DomainOverride defines DNS forwarding for specific domains
type DomainOverride struct {
    // Domain is the domain to override (e.g., "corp.example.com")
    // Supports wildcard subdomains when domain starts with "."
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    Domain string `json:"domain"`

    // Upstreams specifies the DNS servers to forward queries to
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinItems=1
    Upstreams []DomainUpstream `json:"upstreams"`

    // Cache configures caching for this domain override
    // +optional
    Cache *DomainCacheConfig `json:"cache,omitempty"`
}

// DomainUpstream defines an upstream DNS server for domain overrides
type DomainUpstream struct {
    // Address is the DNS server address (IP or hostname)
    // For DoT, use the format "tls://ip" or "tls://hostname"
    // For plain DNS, use just the IP or hostname
    // +kubebuilder:validation:Required
    Address string `json:"address"`

    // Port is the DNS server port (default: 53 for DNS, 853 for DoT)
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=65535
    // +optional
    Port *int32 `json:"port,omitempty"`

    // TLS enables DNS over TLS for this upstream
    // +optional
    TLS *DomainUpstreamTLS `json:"tls,omitempty"`
}

// DomainUpstreamTLS configures TLS for domain override upstreams
type DomainUpstreamTLS struct {
    // Enabled enables TLS for this upstream
    // +kubebuilder:default=false
    // +optional
    Enabled bool `json:"enabled,omitempty"`

    // ServerName is the TLS server name for certificate validation
    // Required when using TLS with IP addresses
    // +optional
    ServerName string `json:"serverName,omitempty"`

    // InsecureSkipVerify disables certificate verification (not recommended)
    // +kubebuilder:default=false
    // +optional
    InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// DomainCacheConfig configures caching for domain overrides
type DomainCacheConfig struct {
    // Enabled enables caching for this domain
    // +kubebuilder:default=true
    // +optional
    Enabled *bool `json:"enabled,omitempty"`

    // TTL specifies the cache TTL in seconds
    // +kubebuilder:validation:Minimum=0
    // +kubebuilder:default=300
    // +optional
    TTL *int32 `json:"ttl,omitempty"`
}
```

### 2. Add field to `NextDNSCoreDNSSpec`:

```go
type NextDNSCoreDNSSpec struct {
    // ... existing fields ...

    // DomainOverrides specifies domain-specific DNS forwarding rules
    // These domains will be forwarded to specified upstreams instead of NextDNS
    // +optional
    DomainOverrides []DomainOverride `json:"domainOverrides,omitempty"`
}
```

### 3. Add status field for override tracking:

```go
// DomainOverrideStatus tracks the status of a domain override
type DomainOverrideStatus struct {
    // Domain is the overridden domain
    Domain string `json:"domain"`

    // Healthy indicates if the upstream servers are responding
    Healthy bool `json:"healthy"`

    // LastChecked is when health was last verified
    LastChecked *metav1.Time `json:"lastChecked,omitempty"`
}

// Add to NextDNSCoreDNSStatus:
type NextDNSCoreDNSStatus struct {
    // ... existing fields ...

    // DomainOverrides tracks the status of each domain override
    // +optional
    DomainOverrides []DomainOverrideStatus `json:"domainOverrides,omitempty"`
}
```

---

## Phase 2: Corefile Generation Updates

**File: `internal/coredns/corefile.go`**

### Update CorefileConfig:

```go
// CorefileConfig holds the configuration for generating a CoreDNS Corefile
type CorefileConfig struct {
    // ... existing fields ...

    // DomainOverrides specifies domain-specific forwarding rules
    DomainOverrides []DomainOverrideConfig `json:"domainOverrides,omitempty"`
}

// DomainOverrideConfig holds configuration for a single domain override
type DomainOverrideConfig struct {
    Domain    string
    Upstreams []UpstreamConfig
    CacheTTL  int32
}

// UpstreamConfig holds configuration for an upstream server
type UpstreamConfig struct {
    Address    string
    Port       int32
    TLS        bool
    ServerName string
}
```

### Update GenerateCorefile function:

```go
// GenerateCorefile generates a CoreDNS Corefile configuration string
func GenerateCorefile(cfg *CorefileConfig) string {
    var sb strings.Builder

    // Generate domain override server blocks FIRST (before catch-all)
    for _, override := range cfg.DomainOverrides {
        writeDomainOverrideBlock(&sb, &override, cfg)
    }

    // Generate main server block for all other domains (catch-all)
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

    sb.WriteString("}\n")

    return sb.String()
}

// writeDomainOverrideBlock writes a server block for a domain override
func writeDomainOverrideBlock(sb *strings.Builder, override *DomainOverrideConfig, cfg *CorefileConfig) {
    // Server block for the specific domain
    fmt.Fprintf(sb, "%s {\n", override.Domain)

    // Forward plugin with override upstreams
    writeOverrideForwardPlugin(sb, override)

    // Cache plugin with domain-specific TTL
    if override.CacheTTL > 0 {
        fmt.Fprintf(sb, "    cache %d\n", override.CacheTTL)
    }

    // Prometheus plugin (if enabled globally)
    if cfg.MetricsEnabled {
        sb.WriteString("    prometheus :9153\n")
    }

    // Log plugin (if enabled globally)
    if cfg.LoggingEnabled {
        sb.WriteString("    log\n")
    }

    // Errors plugin
    sb.WriteString("    errors\n")

    sb.WriteString("}\n\n")
}

// writeOverrideForwardPlugin writes the forward plugin for domain overrides
func writeOverrideForwardPlugin(sb *strings.Builder, override *DomainOverrideConfig) {
    // Build upstream list
    var upstreams []string
    var tlsServerName string

    for _, upstream := range override.Upstreams {
        addr := upstream.Address
        if upstream.Port != 0 && upstream.Port != 53 {
            addr = fmt.Sprintf("%s:%d", addr, upstream.Port)
        }

        if upstream.TLS {
            upstreams = append(upstreams, fmt.Sprintf("tls://%s", addr))
            if upstream.ServerName != "" {
                tlsServerName = upstream.ServerName
            }
        } else {
            upstreams = append(upstreams, addr)
        }
    }

    // Write forward plugin
    fmt.Fprintf(sb, "    forward . %s", strings.Join(upstreams, " "))

    // Add TLS configuration if needed
    if tlsServerName != "" {
        fmt.Fprintf(sb, " {\n        tls_servername %s\n    }\n", tlsServerName)
    } else {
        sb.WriteString("\n")
    }
}
```

### Example Generated Corefile:

```
# Domain override for corp.example.com
corp.example.com {
    forward . 10.0.0.53 10.0.0.54
    cache 300
    prometheus :9153
    errors
}

# Domain override for internal.local (with DoT)
internal.local {
    forward . tls://10.1.0.53 {
        tls_servername internal-dns.local
    }
    cache 600
    prometheus :9153
    errors
}

# Catch-all for NextDNS
. {
    forward . tls://45.90.28.0 tls://45.90.30.0 {
        tls_servername abc123.dns.nextdns.io
    }
    cache 3600
    health :8080
    ready :8181
    prometheus :9153
    errors
}
```

---

## Phase 3: Controller Updates

**File: `internal/controller/nextdnscoredns_controller.go`**

### Update reconcileCorefile function:

```go
func (r *NextDNSCoreDNSReconciler) reconcileCorefile(ctx context.Context, coredns *nextdnsv1alpha1.NextDNSCoreDNS, profile *nextdnsv1alpha1.NextDNSProfile) error {
    // ... existing code ...

    // Build domain overrides config
    var domainOverrides []coredns.DomainOverrideConfig
    for _, override := range coredns.Spec.DomainOverrides {
        overrideConfig := coredns.DomainOverrideConfig{
            Domain:   override.Domain,
            CacheTTL: 300, // default
        }

        if override.Cache != nil && override.Cache.TTL != nil {
            overrideConfig.CacheTTL = *override.Cache.TTL
        }

        for _, upstream := range override.Upstreams {
            upstreamConfig := coredns.UpstreamConfig{
                Address: upstream.Address,
                Port:    53, // default
            }
            if upstream.Port != nil {
                upstreamConfig.Port = *upstream.Port
            }
            if upstream.TLS != nil {
                upstreamConfig.TLS = upstream.TLS.Enabled
                upstreamConfig.ServerName = upstream.TLS.ServerName
            }
            overrideConfig.Upstreams = append(overrideConfig.Upstreams, upstreamConfig)
        }

        domainOverrides = append(domainOverrides, overrideConfig)
    }

    cfg := &coredns.CorefileConfig{
        ProfileID:       profile.Status.ProfileID,
        PrimaryProtocol: string(coredns.Spec.Upstream.Primary),
        CacheTTL:        cacheTTL,
        LoggingEnabled:  loggingEnabled,
        MetricsEnabled:  metricsEnabled,
        DomainOverrides: domainOverrides,
    }

    // ... rest of existing code ...
}
```

---

## Phase 4: Validation

**File: `api/v1alpha1/nextdnscoredns_webhook.go` (or validation function)**

```go
// ValidateDomainOverrides validates the domain override configuration
func ValidateDomainOverrides(overrides []DomainOverride) error {
    seen := make(map[string]bool)

    for i, override := range overrides {
        // Check for duplicate domains
        normalizedDomain := strings.ToLower(override.Domain)
        if seen[normalizedDomain] {
            return fmt.Errorf("duplicate domain override for %q", override.Domain)
        }
        seen[normalizedDomain] = true

        // Validate domain format
        if !isValidDomain(override.Domain) {
            return fmt.Errorf("invalid domain format at index %d: %q", i, override.Domain)
        }

        // Validate upstreams
        if len(override.Upstreams) == 0 {
            return fmt.Errorf("domain override %q must have at least one upstream", override.Domain)
        }

        for j, upstream := range override.Upstreams {
            if upstream.Address == "" {
                return fmt.Errorf("upstream %d for domain %q has empty address", j, override.Domain)
            }

            // Validate TLS configuration
            if upstream.TLS != nil && upstream.TLS.Enabled {
                // If using IP address, ServerName is recommended
                if net.ParseIP(upstream.Address) != nil && upstream.TLS.ServerName == "" {
                    // Warning: TLS with IP address and no ServerName may fail verification
                }
            }
        }
    }

    return nil
}

// isValidDomain checks if the domain format is valid
func isValidDomain(domain string) bool {
    // Allow wildcard domains starting with "."
    if strings.HasPrefix(domain, ".") {
        domain = domain[1:]
    }

    // Basic domain validation
    if len(domain) == 0 || len(domain) > 253 {
        return false
    }

    // Check each label
    labels := strings.Split(domain, ".")
    for _, label := range labels {
        if len(label) == 0 || len(label) > 63 {
            return false
        }
    }

    return true
}
```

---

## Phase 5: Tests

### Unit Tests for Corefile Generation

**File: `internal/coredns/corefile_test.go`**

```go
func TestGenerateCorefileWithDomainOverrides(t *testing.T) {
    tests := []struct {
        name     string
        config   *CorefileConfig
        contains []string
    }{
        {
            name: "single domain override with plain DNS",
            config: &CorefileConfig{
                ProfileID:       "abc123",
                PrimaryProtocol: ProtocolDoT,
                CacheTTL:        3600,
                DomainOverrides: []DomainOverrideConfig{
                    {
                        Domain: "corp.example.com",
                        Upstreams: []UpstreamConfig{
                            {Address: "10.0.0.53", Port: 53},
                        },
                        CacheTTL: 300,
                    },
                },
            },
            contains: []string{
                "corp.example.com {",
                "forward . 10.0.0.53",
                "cache 300",
            },
        },
        {
            name: "domain override with DoT",
            config: &CorefileConfig{
                ProfileID:       "abc123",
                PrimaryProtocol: ProtocolDoT,
                CacheTTL:        3600,
                DomainOverrides: []DomainOverrideConfig{
                    {
                        Domain: "secure.internal",
                        Upstreams: []UpstreamConfig{
                            {
                                Address:    "10.1.0.53",
                                Port:       853,
                                TLS:        true,
                                ServerName: "dns.internal",
                            },
                        },
                        CacheTTL: 600,
                    },
                },
            },
            contains: []string{
                "secure.internal {",
                "forward . tls://10.1.0.53:853",
                "tls_servername dns.internal",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := GenerateCorefile(tt.config)
            for _, expected := range tt.contains {
                if !strings.Contains(result, expected) {
                    t.Errorf("expected Corefile to contain %q, got:\n%s", expected, result)
                }
            }
        })
    }
}
```

---

## Tasks Checklist

- [ ] **API Types**: Add `DomainOverride`, `DomainUpstream`, `DomainCacheConfig` to CRD types
- [ ] **Run `make generate manifests`**: Regenerate CRDs
- [ ] **Corefile Package**: Update `internal/coredns/corefile.go` with domain override support
- [ ] **Controller Integration**: Update reconciler to process domain overrides
- [ ] **Validation**: Add validation for domain override configuration
- [ ] **Unit Tests**: Test Corefile generation with domain overrides
- [ ] **Integration Tests**: Test end-to-end domain override flow
- [ ] **Documentation**: Update README with domain override examples
- [ ] **Sample Resources**: Add sample NextDNSCoreDNS with domain overrides

---

## Example Usage

### Basic Internal Domain Override:

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: home-dns
spec:
  profileRef:
    name: my-profile
  upstream:
    primary: DoT
  domainOverrides:
    - domain: corp.example.com
      upstreams:
        - address: 10.0.0.53
        - address: 10.0.0.54
      cache:
        enabled: true
        ttl: 300
  deployment:
    replicas: 2
  service:
    type: LoadBalancer
```

### Multiple Overrides with TLS:

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: enterprise-dns
spec:
  profileRef:
    name: enterprise-profile
  upstream:
    primary: DoT
  domainOverrides:
    # Internal corporate domains
    - domain: corp.example.com
      upstreams:
        - address: 10.0.0.53
          tls:
            enabled: true
            serverName: internal-dns.corp.example.com
        - address: 10.0.0.54
          tls:
            enabled: true
            serverName: internal-dns.corp.example.com
      cache:
        ttl: 300

    # AWS internal DNS
    - domain: aws.internal
      upstreams:
        - address: 10.10.0.2
      cache:
        ttl: 60

    # Local development
    - domain: dev.local
      upstreams:
        - address: 127.0.0.1
          port: 5353
      cache:
        enabled: false
  deployment:
    replicas: 3
  service:
    type: LoadBalancer
    loadBalancerIP: "192.168.1.53"
```

---

## Open Questions to Address

1. **Should TLS upstream options be supported for overrides?**
   - Recommendation: Yes, included in the design above

2. **Can cache TTL be made configurable per override?**
   - Recommendation: Yes, included in the design above

3. **What validation prevents conflicting domain definitions?**
   - Recommendation: Validation checks for duplicate domains
   - More specific domains take precedence (e.g., `sub.corp.example.com` before `corp.example.com`)

4. **Should we support health checking for override upstreams?**
   - Recommendation: Future enhancement - track in status but don't implement health probes initially

---

## Estimated Effort

| Phase | Complexity | Files Modified |
|-------|------------|----------------|
| API Types | Medium | 1 file |
| Corefile Generation | Medium | 1 file |
| Controller Integration | Low | 1 file |
| Validation | Low | 1 file |
| Tests | Medium | 2-3 files |
| Documentation | Low | 1-2 files |

---

## Future Enhancements

1. **Health Checking**: Implement active health checks for override upstreams
2. **Failover**: Support failover between override upstreams
3. **Metrics**: Per-domain query metrics
4. **Negative Caching**: Configure negative response caching per domain
5. **Policy**: Integration with network policies for upstream access
