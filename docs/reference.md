# CRD Reference

Complete field reference for all 5 NextDNS Operator custom resources, including spec fields, status fields, and conditions.

> For the full documentation index, see the [main docs page](README.md).

---

## NextDNSProfile

The primary resource for managing a NextDNS profile. Each `NextDNSProfile` maps to one profile in the NextDNS dashboard.

### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | No | | Human-readable name shown in NextDNS dashboard (1-100 chars) |
| `mode` | string | No | `managed` | Operational mode: `observe` (read-only) or `managed` (sync spec to remote) |
| `credentialsRef.name` | string | Yes | | Name of the Secret containing the API key |
| `credentialsRef.namespace` | string | No | CR's namespace | Namespace of the Secret (for cross-namespace references) |
| `credentialsRef.key` | string | No | `api-key` | Key within the Secret |
| `profileID` | string | No | | Existing NextDNS profile ID to adopt. If unset, a new profile is created |
| `allowlistRefs` | ListReference[] | No | | References to NextDNSAllowlist resources |
| `denylistRefs` | ListReference[] | No | | References to NextDNSDenylist resources |
| `tldListRefs` | ListReference[] | No | | References to NextDNSTLDList resources |
| `allowlist` | DomainEntry[] | No | | Inline domains to allow (merged with allowlistRefs) |
| `denylist` | DomainEntry[] | No | | Inline domains to block (merged with denylistRefs) |
| `security` | SecuritySpec | No | | Threat protection settings (see below) |
| `privacy` | PrivacySpec | No | | Tracker and ad blocking settings (see below) |
| `parentalControl` | ParentalControlSpec | No | | Content filtering settings (see below) |
| `rewrites` | RewriteEntry[] | No | | DNS rewrite rules |
| `settings` | SettingsSpec | No | | Logging, performance, and other options (see below) |
| `configMapRef` | ConfigMapRef | No | | Enable ConfigMap creation with connection details |

**SecuritySpec:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `aiThreatDetection` | *bool | `true` | AI-based threat detection |
| `threatIntelligenceFeeds` | *bool | `true` | Enable threat intelligence feeds |
| `googleSafeBrowsing` | *bool | `true` | Google Safe Browsing protection |
| `cryptojacking` | *bool | `true` | Block cryptomining scripts |
| `dnsRebinding` | *bool | `true` | DNS rebinding attack protection |
| `idnHomographs` | *bool | `true` | Block IDN homograph attacks |
| `typosquatting` | *bool | `true` | Block typosquatting domains |
| `dga` | *bool | `true` | Block algorithmically-generated domains |
| `nrd` | *bool | `false` | Block newly registered domains |
| `ddns` | *bool | `false` | Block dynamic DNS hostnames |
| `parking` | *bool | `true` | Block parked domains |
| `csam` | *bool | `true` | Block child sexual abuse material |

**PrivacySpec:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `blocklists` | BlocklistEntry[] | | Ad/tracker blocklists (`id` required, `active` defaults to `true`) |
| `natives` | NativeEntry[] | | Native tracking protection per vendor (`id` required, `active` defaults to `true`) |
| `disguisedTrackers` | *bool | `true` | Block CNAME-cloaked trackers |
| `allowAffiliate` | *bool | `false` | Allow affiliate & tracking links |

**ParentalControlSpec:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `categories` | CategoryEntry[] | | Content categories to block (`id` required, `active` defaults to `true`) |
| `services` | ServiceEntry[] | | Specific services to block (`id` required, `active` defaults to `true`) |
| `safeSearch` | *bool | `false` | Enforce safe search on search engines |
| `youtubeRestrictedMode` | *bool | `false` | Enforce YouTube restricted mode |
| `blockBypass` | *bool | `false` | Prevent bypassing parental controls |

Each `CategoryEntry` has:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id` | string | | Category identifier |
| `active` | *bool | `true` | Whether this category is blocked |
| `recreation` | *bool | `false` | Allow recreation time exceptions for this category |

**SettingsSpec:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `logs.enabled` | *bool | `true` | Enable query logging |
| `logs.logClientsIPs` | *bool | `false` | Log client IP addresses |
| `logs.logDomains` | *bool | `true` | Log queried domains |
| `logs.retention` | string | `7d` | Log retention (`1h`, `6h`, `1d`, `7d`, `30d`, `90d`, `1y`, `2y`) |
| `logs.location` | string | | Log storage location (e.g., `eu`, `us`, `ch`) |
| `blockPage.enabled` | *bool | `true` | Show block page instead of failing silently |
| `performance.ecs` | *bool | `true` | EDNS Client Subnet for geo-aware responses |
| `performance.cacheBoost` | *bool | `true` | Extended caching at NextDNS edge |
| `performance.cnameFlattening` | *bool | `true` | CNAME flattening |
| `web3` | *bool | `false` | Web3 domain resolution |
| `bav` | *bool | `false` | Bypass Age Verification |

**Shared types:**

| Type | Fields | Description |
|------|--------|-------------|
| `ListReference` | `name` (required), `namespace` (optional) | Reference to a list CRD; namespace defaults to profile's namespace |
| `DomainEntry` | `domain` (required), `active` (default: true), `reason` (optional) | Domain entry for allow/deny lists; supports wildcards (`*.example.com`) |
| `RewriteEntry` | `from` (required), `to` (required), `active` (default: true) | DNS rewrite rule |
| `ConfigMapRef` | `enabled` (default: false), `name` (optional) | ConfigMap export config; name defaults to `<profile-name>-nextdns` |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `profileID` | string | NextDNS-assigned profile identifier |
| `fingerprint` | string | Profile fingerprint from the NextDNS API, used for DNS endpoint construction |
| `aggregatedCounts.allowlistDomains` | int | Total allowlisted domains from all sources |
| `aggregatedCounts.denylistDomains` | int | Total denylisted domains from all sources |
| `aggregatedCounts.blockedTLDs` | int | Total blocked TLDs from all sources |
| `referencedResources.allowlists` | []ReferencedResourceStatus | Status of each referenced allowlist |
| `referencedResources.denylists` | []ReferencedResourceStatus | Status of each referenced denylist |
| `referencedResources.tldLists` | []ReferencedResourceStatus | Status of each referenced TLD list |
| `setup.ipv4` | []string | Profile-specific IPv4 upstream addresses |
| `setup.ipv6` | []string | Profile-specific IPv6 upstream addresses |
| `setup.linkedIP.servers` | []string | Linked-IP upstream servers |
| `setup.linkedIP.ip` | string | Linked-IP assigned IP address |
| `setup.linkedIP.ddns` | string | Linked-IP DDNS hostname |
| `setup.dnsCrypt` | DNSCryptConfig | DNSCrypt relay configuration |
| `setup.dotHostname` | string | DNS-over-TLS hostname (e.g., `abc123.dns.nextdns.io`) |
| `setup.dohURL` | string | DNS-over-HTTPS URL (e.g., `https://dns.nextdns.io/abc123`) |
| `conditions` | []Condition | Standard Kubernetes conditions (see Conditions below) |
| `lastSyncTime` | Time | Last time the profile was synced with NextDNS API |
| `observedGeneration` | int64 | Generation last processed by the controller |
| `observedConfig` | ObservedConfig | Full observed state of remote profile (observe mode only) |
| `suggestedSpec` | SuggestedSpec | Spec-compatible translation of observed config for easy transition |

### Conditions

| Type | True | False |
|------|------|-------|
| **Ready** | Profile is fully synced and operational | One or more subsystems have issues |
| **Synced** | Spec successfully applied to NextDNS API | API sync failed (check `message` for details) |
| **ReferencesResolved** | All referenced lists exist and are ready | One or more list references are missing or not ready |
| **ObserveOnly** | Profile is in observe-only mode (reading remote, not writing) | Profile is in managed mode |

---

## NextDNSAllowlist

A reusable list of domains to allow. Can be referenced by multiple `NextDNSProfile` resources via `allowlistRefs`.

### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `description` | string | No | | Human-readable description of this allowlist |
| `domains` | DomainEntry[] | Yes (min 1) | | Domains to allow |

Each `DomainEntry` has:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `domain` | string | Yes | | Domain name (supports wildcards like `*.example.com`, max 253 chars) |
| `active` | *bool | No | `true` | Whether this entry is enabled |
| `reason` | string | No | | Why this domain is allowlisted |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `domainCount` | int | Number of active domains in this list |
| `profileRefs` | ResourceReference[] | Profiles currently using this allowlist |
| `conditions` | []Condition | Standard Kubernetes conditions |

---

## NextDNSDenylist

A reusable list of domains to block. Can be referenced by multiple `NextDNSProfile` resources via `denylistRefs`.

### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `description` | string | No | | Human-readable description of this denylist |
| `domains` | DomainEntry[] | Yes (min 1) | | Domains to block |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `domainCount` | int | Number of active domains in this list |
| `profileRefs` | ResourceReference[] | Profiles currently using this denylist |
| `conditions` | []Condition | Standard Kubernetes conditions |

---

## NextDNSTLDList

A reusable list of top-level domains to block. Can be referenced by multiple `NextDNSProfile` resources via `tldListRefs`.

### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `description` | string | No | | Human-readable description of this TLD list |
| `tlds` | TLDEntry[] | Yes (min 1) | | TLDs to block |

Each `TLDEntry` has:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `tld` | string | Yes | | Top-level domain without the dot (e.g., `com`, `net`, `co.uk`; max 63 chars) |
| `active` | *bool | No | `true` | Whether this TLD is blocked |
| `reason` | string | No | | Why this TLD is blocked |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `tldCount` | int | Number of active TLDs in this list |
| `profileRefs` | ResourceReference[] | Profiles currently using this TLD list |
| `conditions` | []Condition | Standard Kubernetes conditions |

---

## NextDNSCoreDNS

Deploys a CoreDNS instance configured to forward DNS queries to a NextDNS profile.

### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `profileRef.name` | string | Yes | | Name of the NextDNSProfile to use |
| `profileRef.namespace` | string | No | | Namespace (defaults to same namespace) |
| `corefile.upstream.primary` | DNSProtocol | Yes (if `upstream` set) | `DoT` | Upstream protocol: `DoT`, `DoH`, or `DNS` |
| `corefile.upstream.deviceName` | string | No | | Device name for NextDNS Analytics (max 63 chars, alphanumeric/hyphens/spaces) |
| `corefile.upstream.forward.policy` | ForwardPolicy | No | `random` (CoreDNS default) | Failover policy: `random`, `round_robin`, or `sequential` |
| `corefile.upstream.forward.maxConcurrent` | *int32 | No | unlimited | Cap on concurrent upstream queries (min 1) |
| `corefile.upstream.forward.healthCheck` | string | No | `500ms` (CoreDNS default) | Interval between upstream health checks (Go duration) |
| `corefile.upstream.forward.expire` | string | No | `10s` (CoreDNS default) | Idle upstream connection expiration (Go duration) |
| `corefile.upstream.forward.maxFails` | *int32 | No | `2` (CoreDNS default) | Failed health checks before marking upstream down |
| `deployment.mode` | DeploymentMode | No | `Deployment` | `Deployment` or `DaemonSet` |
| `deployment.replicas` | *int32 | No | `2` | Replicas (Deployment mode only, min: 1) |
| `deployment.image` | string | No | `mirror.gcr.io/coredns/coredns:1.13.1` | CoreDNS container image |
| `deployment.nodeSelector` | map[string]string | No | | Node label selector |
| `deployment.affinity` | Affinity | No | | Pod scheduling constraints |
| `deployment.tolerations` | Toleration[] | No | | Pod tolerations |
| `deployment.resources` | ResourceRequirements | No | | CPU/memory requests and limits |
| `deployment.podAnnotations` | map[string]string | No | | Additional pod annotations (prefer `spec.multus` for Multus) |
| `deployment.podDisruptionBudget.minAvailable` | IntOrString | No | | Min pods available (mutually exclusive with maxUnavailable) |
| `deployment.podDisruptionBudget.maxUnavailable` | IntOrString | No | — | Max pods unavailable (mutually exclusive with minAvailable). Defaults to 1 in the generated PDB if neither minAvailable nor maxUnavailable is set. |
| `service.type` | CoreDNSServiceType | No | `ClusterIP` | `ClusterIP` or `LoadBalancer` |
| `service.loadBalancerIP` | string | No | | Static IP for LoadBalancer (valid IPv4) |
| `service.annotations` | map[string]string | No | | Additional service annotations |
| `service.nameOverride` | string | No | | Custom service name |
| `corefile.cache.enabled` | *bool | No | `true` | Enable DNS response caching |
| `corefile.cache.successTTL` | *int32 | No | `3600` | Cache TTL for successful responses (seconds) |
| `corefile.metrics.enabled` | *bool | No | `true` | Enable Prometheus metrics endpoint |
| `corefile.metrics.port` | *int32 | No | `9153` | Prometheus plugin listen port |
| `corefile.health.enabled` | *bool | No | `true` | Enable health plugin and the deployment's liveness probe |
| `corefile.health.port` | *int32 | No | `8080` | Health plugin listen port (also used for the liveness probe) |
| `corefile.health.lameduck` | string | No | | Delay shutdown to drain load-balancer traffic (Go duration string) |
| `corefile.ready.enabled` | *bool | No | `true` | Enable ready plugin and the deployment's readiness probe |
| `corefile.ready.port` | *int32 | No | `8181` | Ready plugin listen port (also used for the readiness probe) |
| `corefile.errors.enabled` | *bool | No | `true` | Enable the errors plugin |
| `corefile.errors.consolidate` | ConsolidateRule[] | No | | Log-spam consolidation rules (see below) |
| `corefile.logging.enabled` | *bool | No | `false` | Enable DNS query logging |
| `corefile.domainOverrides` | DomainOverride[] | No | | Domain-specific upstream overrides |
| `corefile.rewrite` | RewriteRule[] | No | | Query rewrite rules (rewrite plugin) |
| `corefile.hosts.entries` | HostsEntry[] | Yes (if `hosts` set) | | Static IP-to-hostname mappings |
| `corefile.hosts.fallthrough` | *bool | No | `true` | Pass unmatched names to next plugin |
| `corefile.hosts.ttl` | *int32 | No | `3600` (CoreDNS default) | TTL for static entries (seconds) |
| `multus.networkAttachmentDefinition` | string | Yes (if `multus` set) | | Name of the NetworkAttachmentDefinition CR |
| `multus.namespace` | string | No | CR namespace | Namespace of the NetworkAttachmentDefinition |
| `multus.ips` | string[] | No | | Static IPs to request from IPAM (one per pod) |
| `gateway.gatewayClassName` | *string | No | Operator default | GatewayClass to reference (e.g., `envoy-gateway`, `cilium`) |
| `gateway.addresses` | GatewayAddress[] | Yes (if `gateway` set) | | IP addresses or hostnames requested from the gateway implementation |
| `gateway.replicas` | *int32 | No | | Replica count for gateway proxy pods (Envoy Gateway only, min 1). Mutually exclusive with `gateway.infrastructure.parametersRef`. |
| `gateway.annotations` | map[string]string | No | | Additional annotations for the Gateway resource |
| `gateway.infrastructure.annotations` | map[string]string | No | | Annotations propagated to gateway-generated resources (e.g., LB Service) |
| `gateway.infrastructure.labels` | map[string]string | No | | Labels propagated to gateway-generated resources |
| `gateway.infrastructure.parametersRef.group` | string | Yes (if `parametersRef` set) | | API group of the implementation-specific config resource |
| `gateway.infrastructure.parametersRef.kind` | string | Yes (if `parametersRef` set) | | Kind of the implementation-specific config resource |
| `gateway.infrastructure.parametersRef.name` | string | Yes (if `parametersRef` set) | | Name of the implementation-specific config resource |

**GatewayAddress sub-fields:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `value` | string | Yes | | IP address or hostname requested from the gateway implementation |
| `type` | string | No | `IPAddress` | Address type: `IPAddress` or `Hostname` |

Each `DomainOverride` has:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `domain` | string | Yes | | DNS domain to override (e.g., `corp.example.com`) |
| `upstreams` | string[] | Yes (min 1) | | Upstream DNS server IPs (IPv4 or IPv4:port) |
| `cacheTTL` | *int32 | No | | Cache TTL for this domain (seconds) |

Each `RewriteRule` has:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | Yes | | Rewrite type: `name`, `class`, `type`, `ttl`, `edns0` |
| `match` | string | Yes | | Pattern to match (query name, class, etc.) |
| `replacement` | string | Yes | | Value to rewrite to |
| `matcher` | string | No | `exact` | Sub-type for `name` rewrites: `exact`, `prefix`, `suffix`, `substring`, `regex` |

Each `ConsolidateRule` (used in `corefile.errors.consolidate`) has:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `interval` | string | Yes | | Consolidation window (Go duration string, e.g., `5m`) |
| `pattern` | string | Yes | | Regular expression matched against error log lines |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `profileID` | string | NextDNS profile ID from the referenced profile |
| `fingerprint` | string | DNS fingerprint from the referenced profile |
| `endpoints` | DNSEndpoint[] | DNS endpoints exposed by the service (`ip`, `port`, `protocol`) |
| `dnsIP` | string | Primary DNS IP address for easy reference |
| `multusIPs` | string[] | IPs assigned to pods via Multus (from network-status annotation) |
| `upstream.url` | string | NextDNS upstream URL being used |
| `replicas.desired` | int32 | Desired replica count |
| `replicas.ready` | int32 | Ready replica count |
| `replicas.available` | int32 | Available replica count |
| `gatewayReady` | bool | Whether the Gateway is programmed and accepting traffic |
| `ready` | bool | Whether the CoreDNS deployment is fully ready |
| `conditions` | []Condition | Standard Kubernetes conditions |
| `lastUpdated` | Time | Last time the status was updated |
| `observedGeneration` | int64 | Generation last processed by the controller |

### Conditions

| Type | True | False |
|------|------|-------|
| **Ready** | All CoreDNS resources deployed and healthy | Workload, service, or configmap has issues |
| **ProfileResolved** | Referenced NextDNSProfile exists and is Ready | Profile not found or not in Ready state |
| **GatewayReady** | Gateway is programmed by external controller | Gateway not programmed, CRDs missing, or no class name configured |
| **TCPRouteReady** | TCPRoute reconciled successfully | TCPRoute creation/update failed |
| **UDPRouteReady** | UDPRoute reconciled successfully | UDPRoute creation/update failed |
