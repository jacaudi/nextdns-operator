# NextDNS Operator Documentation

Comprehensive documentation for the NextDNS Kubernetes Operator. For a quick overview and getting started guide, see the [root README](../README.md).

## Table of Contents

- [Configuration](#configuration)
  - [ConfigMap Export](#configmap-export)
  - [ConfigMap Import](#configmap-import)
    - [Supported Import Fields](#supported-import-fields)
- [CoreDNS Deployment](#coredns-deployment)
  - [Basic Setup](#basic-setup)
  - [Upstream Protocols](#upstream-protocols)
  - [Deployment Modes](#deployment-modes)
  - [Service Configuration](#service-configuration)
  - [Caching](#caching)
  - [Metrics & Monitoring](#metrics--monitoring)
  - [Query Logging](#query-logging)
  - [Resource Requirements](#resource-requirements)
  - [Multus CNI Integration](#multus-cni-integration)
  - [Domain Overrides](#domain-overrides)
- [Drift Detection](#drift-detection)
- [CRD Reference](#crd-reference)
  - [NextDNSProfile](#nextdnsprofile)
  - [NextDNSAllowlist](#nextdnsallowlist)
  - [NextDNSDenylist](#nextdnsdenylist)
  - [NextDNSTLDList](#nextdnstldlist)
  - [NextDNSCoreDNS](#nextdnscoredns)
- [Status & Conditions](#status--conditions)
- [Troubleshooting](#troubleshooting)
- [Architecture](#architecture)

---

## Configuration

### ConfigMap Export

Optionally create a ConfigMap containing your profile's DNS connection details. This is useful for configuring DNS clients (CoreDNS, Blocky, etc.) or injecting connection details into pods.

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSProfile
metadata:
  name: my-profile
spec:
  name: "My Profile"
  credentialsRef:
    name: nextdns-credentials
  configMapRef:
    enabled: true
    name: my-dns-config  # optional, defaults to "<profile-name>-nextdns"
```

The created ConfigMap contains:

```yaml
data:
  NEXTDNS_PROFILE_ID: "abc123"
  NEXTDNS_DOT: "abc123.dns.nextdns.io"
  NEXTDNS_DOH: "https://dns.nextdns.io/abc123"
  NEXTDNS_DOQ: "quic://abc123.dns.nextdns.io"
  NEXTDNS_IPV4_1: "45.90.28.0"
  NEXTDNS_IPV4_2: "45.90.30.0"
  NEXTDNS_IPV6_1: "2a07:a8c0::"
  NEXTDNS_IPV6_2: "2a07:a8c1::"
```

Use it in your pods with `envFrom`:

```yaml
envFrom:
  - configMapRef:
      name: my-dns-config
```

### ConfigMap Import

Import base profile configuration from a ConfigMap containing JSON. Explicit spec fields always take precedence over imported values, making this useful for shared base configurations, migration from existing profiles, or templating.

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSProfile
metadata:
  name: my-profile
spec:
  name: "My Profile"
  credentialsRef:
    name: nextdns-credentials
  configImportRef:
    name: base-profile-config
    key: config.json  # optional, defaults to "config.json"
  # Explicit spec fields override imported values
  security:
    nrd: false  # overrides imported nrd value
```

The referenced ConfigMap contains profile configuration as JSON:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: base-profile-config
data:
  config.json: |
    {
      "security": {
        "aiThreatDetection": true,
        "googleSafeBrowsing": true,
        "nrd": true
      },
      "privacy": {
        "blocklists": [
          {"id": "nextdns-recommended", "active": true}
        ],
        "disguisedTrackers": true
      },
      "denylist": [
        {"domain": "ads.example.com", "active": true}
      ],
      "settings": {
        "logs": {"enabled": true, "retention": "30d"}
      }
    }
```

**Precedence rules:**
- Explicit spec fields always win over imported values
- Imported values fill in fields not set in the spec
- Lists (denylist, allowlist, rewrites, blocklists, etc.) are merged with deduplication
- Changes to the import ConfigMap trigger re-reconciliation

#### Supported Import Fields

Below is a complete JSON example showing every supported field. All fields are optional -- include only what you need.

```json
{
  "security": {
    "aiThreatDetection": true,
    "googleSafeBrowsing": true,
    "cryptojacking": true,
    "dnsRebinding": true,
    "idnHomographs": true,
    "typosquatting": true,
    "dga": true,
    "nrd": false,
    "ddns": false,
    "parking": true,
    "csam": true,
    "threatIntelligenceFeeds": ["feed-id-1", "feed-id-2"]
  },
  "privacy": {
    "blocklists": [
      {"id": "nextdns-recommended", "active": true},
      {"id": "oisd", "active": true}
    ],
    "natives": [
      {"id": "apple", "active": true},
      {"id": "windows", "active": true}
    ],
    "disguisedTrackers": true,
    "allowAffiliate": false
  },
  "parentalControl": {
    "categories": [
      {"id": "gambling", "active": true},
      {"id": "adult", "active": true}
    ],
    "services": [
      {"id": "tiktok", "active": true},
      {"id": "facebook", "active": false}
    ],
    "safeSearch": false,
    "youtubeRestrictedMode": false
  },
  "denylist": [
    {"domain": "ads.example.com", "active": true},
    {"domain": "tracker.example.com", "active": true}
  ],
  "allowlist": [
    {"domain": "safe.example.com", "active": true}
  ],
  "rewrites": [
    {"from": "app.internal", "to": "10.0.0.5", "active": true}
  ],
  "settings": {
    "logs": {
      "enabled": true,
      "logClientsIPs": false,
      "logDomains": true,
      "retention": "30d"
    },
    "blockPage": {
      "enabled": true
    },
    "performance": {
      "ecs": true,
      "cacheBoost": true,
      "cnameFlattening": true
    },
    "web3": false
  }
}
```

| Section | Field | Type | Description |
|---------|-------|------|-------------|
| `security` | `aiThreatDetection` | bool | AI-based threat detection |
| | `googleSafeBrowsing` | bool | Google Safe Browsing protection |
| | `cryptojacking` | bool | Block cryptomining scripts |
| | `dnsRebinding` | bool | DNS rebinding attack protection |
| | `idnHomographs` | bool | Block IDN homograph attacks |
| | `typosquatting` | bool | Block typosquatting domains |
| | `dga` | bool | Block algorithmically-generated domains |
| | `nrd` | bool | Block newly registered domains |
| | `ddns` | bool | Block dynamic DNS hostnames |
| | `parking` | bool | Block parked domains |
| | `csam` | bool | Block child sexual abuse material |
| | `threatIntelligenceFeeds` | string[] | Threat feed identifiers to enable |
| `privacy` | `blocklists` | object[] | Ad/tracker blocklists (`id`, `active`) |
| | `natives` | object[] | Native tracking protection per vendor (`id`, `active`) |
| | `disguisedTrackers` | bool | Block CNAME-cloaked trackers |
| | `allowAffiliate` | bool | Allow affiliate & tracking links |
| `parentalControl` | `categories` | object[] | Content categories to block (`id`, `active`) |
| | `services` | object[] | Specific services to block (`id`, `active`) |
| | `safeSearch` | bool | Enforce safe search on search engines |
| | `youtubeRestrictedMode` | bool | Enforce YouTube restricted mode |
| `denylist` | | object[] | Domains to block (`domain`, `active`) |
| `allowlist` | | object[] | Domains to allow (`domain`, `active`) |
| `rewrites` | | object[] | DNS rewrites (`from`, `to`, `active`) |
| `settings.logs` | `enabled` | bool | Enable query logging |
| | `logClientsIPs` | bool | Log client IP addresses |
| | `logDomains` | bool | Log queried domains |
| | `retention` | string | Log retention period (`1h`, `6h`, `1d`, `7d`, `30d`, `90d`, `1y`, `2y`) |
| `settings.blockPage` | `enabled` | bool | Show block page instead of failing silently |
| `settings.performance` | `ecs` | bool | EDNS Client Subnet |
| | `cacheBoost` | bool | Extended caching |
| | `cnameFlattening` | bool | CNAME flattening |
| `settings` | `web3` | bool | Web3 domain resolution |

**Notes:**
- Unknown fields in the JSON are silently ignored (the operator logs a warning).
- Domain values in `denylist`, `allowlist`, and `rewrites` are validated against the same rules as inline spec domains.
- List entries have configurable upper bounds: 1000 for deny/allowlists, 500 for rewrites, and 100 for blocklists.

---

## CoreDNS Deployment

Deploy a dedicated CoreDNS instance that forwards DNS queries to NextDNS using the `NextDNSCoreDNS` custom resource. This is useful for providing DNS services to devices on your network (home routers, IoT devices, etc.) that can't use DoH/DoT directly.

### Basic Setup

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: home-dns
spec:
  profileRef:
    name: my-profile  # References an existing NextDNSProfile

  upstream:
    primary: DoT      # DNS over TLS (recommended)

  deployment:
    mode: Deployment
    replicas: 2

  service:
    type: LoadBalancer
    loadBalancerIP: "192.168.1.53"  # Optional static IP

  cache:
    enabled: true
    successTTL: 3600  # Cache TTL in seconds
```

**Check deployment status:**

```bash
kubectl get nextdnscoredns home-dns
# NAME       PROFILE ID   DNS IP          READY   AGE
# home-dns   abc123       192.168.1.53    true    5m
```

### Upstream Protocols

The `upstream.primary` field controls how CoreDNS connects to NextDNS. Three protocols are supported:

| Protocol | Port | Encrypted | Recommended |
|----------|------|-----------|-------------|
| `DoT` | 853 | Yes (TLS) | Yes (default) |
| `DoH` | 443 | Yes (HTTPS) | Yes |
| `DNS` | 53 | No | No |

**DNS over TLS (DoT)** is the default and recommended protocol. It encrypts DNS queries using TLS on port 853, providing privacy without the overhead of HTTPS.

**DNS over HTTPS (DoH)** encrypts DNS queries inside HTTPS requests. This can be useful in environments where port 853 is blocked, since DoH uses the standard HTTPS port 443.

**Plain DNS** sends queries unencrypted on port 53. This offers the lowest latency but provides no privacy.

> **Security Note:** Using plain DNS (`DNS` protocol) exposes your NextDNS profile ID in unencrypted traffic. Your DNS queries and the profile ID are visible to anyone observing network traffic. Use DoT or DoH for privacy in untrusted networks.

```yaml
upstream:
  primary: DoT  # DoT, DoH, or DNS
```

### Deployment Modes

CoreDNS can be deployed as either a **Deployment** or a **DaemonSet**:

**Deployment** (default): Runs a configurable number of replicas. Best for most use cases where you want centralized DNS serving with horizontal scaling.

```yaml
deployment:
  mode: Deployment
  replicas: 2  # default: 2, minimum: 1
```

**DaemonSet**: Runs one CoreDNS pod on every matching node (or every node if no nodeSelector is set). Best for scenarios where you want DNS available on every node, such as when using Multus CNI to expose DNS on node-local network interfaces.

```yaml
deployment:
  mode: DaemonSet
  # replicas is ignored in DaemonSet mode
```

### Service Configuration

The operator creates a Kubernetes Service for the CoreDNS deployment. Three service types are supported:

**ClusterIP** (default): Exposes the service on a cluster-internal IP. Only reachable from within the cluster.

```yaml
service:
  type: ClusterIP
```

**LoadBalancer**: Exposes the service via a cloud provider load balancer or MetalLB. Use `loadBalancerIP` to request a specific IP address.

```yaml
service:
  type: LoadBalancer
  loadBalancerIP: "192.168.1.53"  # optional static IP
  annotations:
    metallb.universe.tf/address-pool: dns-pool  # example: MetalLB annotation
```

**NodePort**: Exposes the service on each node's IP at a static port. Useful when you want to reach DNS via any node IP.

```yaml
service:
  type: NodePort
```

**Name override**: By default, the service is named after the NextDNSCoreDNS resource. Use `nameOverride` to set a custom name:

```yaml
service:
  nameOverride: my-dns-service
```

### Caching

CoreDNS caching is enabled by default with a 3600-second (1 hour) TTL for successful responses. The cache respects upstream TTL values -- if the upstream response has a lower TTL, that value is used instead.

```yaml
cache:
  enabled: true       # default: true
  successTTL: 3600    # default: 3600 (seconds)
```

To disable caching:

```yaml
cache:
  enabled: false
```

Setting `successTTL: 0` keeps the cache enabled but uses only upstream TTL values without overriding.

### Metrics & Monitoring

CoreDNS exposes a Prometheus metrics endpoint on port 9153 by default.

```yaml
metrics:
  enabled: true  # default: true
```

**ServiceMonitor** (for Prometheus Operator):

> **Note:** ServiceMonitor reconciliation is not yet implemented in the controller. The configuration is accepted but the ServiceMonitor resource is not currently created. This is planned for a future release.

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    namespace: monitoring  # optional, defaults to resource namespace
    interval: "30s"        # default: 30s
    labels:
      release: prometheus  # match your Prometheus Operator selector
```

### Query Logging

Enable CoreDNS query logging for debugging DNS resolution issues. Disabled by default to reduce log volume.

```yaml
logging:
  enabled: true  # default: false
```

When enabled, CoreDNS logs all incoming DNS queries to stdout. This is useful for debugging but can generate significant log volume in production.

### Resource Requirements

Configure compute resources, node placement, and tolerations for CoreDNS pods:

```yaml
deployment:
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi
  nodeSelector:
    kubernetes.io/os: linux
    node-role.kubernetes.io/dns: ""
  tolerations:
    - key: node-role.kubernetes.io/dns
      operator: Exists
      effect: NoSchedule
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values: ["coredns"]
            topologyKey: kubernetes.io/hostname
```

**Security defaults**: CoreDNS containers run with a read-only root filesystem and all Linux capabilities dropped. No additional security configuration is needed.

### Multus CNI Integration

For advanced networking scenarios, you can attach CoreDNS pods to additional networks using [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni). This is useful for exposing DNS services directly on a VLAN or dedicated network interface.

**Example: CoreDNS on a VLAN with primary and secondary IPs**

First, create a NetworkAttachmentDefinition for your VLAN:

```yaml
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: dns-vlan
  namespace: default
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "type": "macvlan",
      "master": "eth0.100",
      "mode": "bridge",
      "ipam": {
        "type": "static",
        "addresses": [
          { "address": "192.168.100.53/24" },
          { "address": "192.168.100.54/24" }
        ],
        "routes": [
          { "dst": "0.0.0.0/0", "gw": "192.168.100.1" }
        ]
      }
    }
```

Then reference it in your NextDNSCoreDNS resource:

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: vlan-dns
spec:
  profileRef:
    name: my-profile

  upstream:
    primary: DoT

  deployment:
    mode: DaemonSet
    podAnnotations:
      k8s.v1.cni.cncf.io/networks: dns-vlan

  service:
    type: ClusterIP  # Internal only; clients use Multus IPs directly
```

The CoreDNS pods will now have interfaces on both the cluster network and the VLAN, accessible at `192.168.100.53` and `192.168.100.54`.

### Domain Overrides

Configure domain-specific DNS upstream servers for split-horizon DNS:

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
        - 10.0.0.1
        - 10.0.0.2
      cacheTTL: 60
    - domain: internal.local
      upstreams:
        - 192.168.1.1
```

This generates a Corefile with domain-specific server blocks:

```
corp.example.com {
    forward . 10.0.0.1 10.0.0.2
    cache 60
    errors
}

internal.local {
    forward . 192.168.1.1
    cache 30
    errors
}

. {
    forward . tls://45.90.28.0 tls://45.90.30.0 {
        tls_servername profileid.dns.nextdns.io
    }
    cache 3600
    ...
}
```

**Use cases:**
- Forward internal domains to internal DNS servers
- Split-horizon DNS for private zones
- Override resolution for specific domains without affecting NextDNS upstream

---

## Drift Detection

The operator periodically reconciles all resources to detect and correct drift from manual changes made outside Kubernetes.

**Configure via environment variable:**
```bash
SYNC_PERIOD=30m ./nextdns-operator
```

**Configure via command-line flag:**
```bash
./nextdns-operator --sync-period=30m
```

**Disable periodic syncing:**
```bash
SYNC_PERIOD=0 ./nextdns-operator
```

**Default:** `1h` (60 minutes)

**Behavior:**
- Syncs include +/-10% jitter to prevent all resources from hitting the API simultaneously
- Each profile makes ~1 API call per sync period
- List resources (allowlist, denylist, tldlist) sync status but don't call the NextDNS API directly
- Setting to `0` disables periodic syncing (event-driven only)

---

## CRD Reference

### NextDNSProfile

The primary resource for managing a NextDNS profile. Each `NextDNSProfile` maps to one profile in the NextDNS dashboard.

#### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | | Human-readable name shown in NextDNS dashboard (1-100 chars) |
| `credentialsRef.name` | string | Yes | | Name of the Secret containing the API key |
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
| `configImportRef` | ConfigImportRef | No | | Import base config from a ConfigMap |

**SecuritySpec:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `aiThreatDetection` | *bool | `true` | AI-based threat detection |
| `threatIntelligenceFeeds` | string[] | | Threat feed identifiers to enable |
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

**SettingsSpec:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `logs.enabled` | *bool | `true` | Enable query logging |
| `logs.logClientsIPs` | *bool | `false` | Log client IP addresses |
| `logs.logDomains` | *bool | `true` | Log queried domains |
| `logs.retention` | string | `7d` | Log retention (`1h`, `6h`, `1d`, `7d`, `30d`, `90d`, `1y`, `2y`) |
| `blockPage.enabled` | *bool | `true` | Show block page instead of failing silently |
| `performance.ecs` | *bool | `true` | EDNS Client Subnet for geo-aware responses |
| `performance.cacheBoost` | *bool | `true` | Extended caching at NextDNS edge |
| `performance.cnameFlattening` | *bool | `true` | CNAME flattening |
| `web3` | *bool | `false` | Web3 domain resolution |

**Shared types:**

| Type | Fields | Description |
|------|--------|-------------|
| `ListReference` | `name` (required), `namespace` (optional) | Reference to a list CRD; namespace defaults to profile's namespace |
| `DomainEntry` | `domain` (required), `active` (default: true), `reason` (optional) | Domain entry for allow/deny lists; supports wildcards (`*.example.com`) |
| `RewriteEntry` | `from` (required), `to` (required), `active` (default: true) | DNS rewrite rule |
| `ConfigMapRef` | `enabled` (default: false), `name` (optional) | ConfigMap export config; name defaults to `<profile-name>-nextdns` |
| `ConfigImportRef` | `name` (required), `key` (default: `config.json`) | ConfigMap import reference |

#### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `profileID` | string | NextDNS-assigned profile identifier |
| `fingerprint` | string | DNS endpoint (e.g., `abc123.dns.nextdns.io`) |
| `aggregatedCounts.allowlistDomains` | int | Total allowlisted domains from all sources |
| `aggregatedCounts.denylistDomains` | int | Total denylisted domains from all sources |
| `aggregatedCounts.blockedTLDs` | int | Total blocked TLDs from all sources |
| `referencedResources.allowlists` | []ReferencedResourceStatus | Status of each referenced allowlist |
| `referencedResources.denylists` | []ReferencedResourceStatus | Status of each referenced denylist |
| `referencedResources.tldLists` | []ReferencedResourceStatus | Status of each referenced TLD list |
| `conditions` | []Condition | Standard Kubernetes conditions (see [Status & Conditions](#status--conditions)) |
| `lastSyncTime` | Time | Last time the profile was synced with NextDNS API |
| `observedGeneration` | int64 | Generation last processed by the controller |
| `configImportResourceVersion` | string | ResourceVersion of the imported ConfigMap (for change detection) |

#### Conditions

| Type | Description |
|------|-------------|
| `Ready` | Overall readiness -- `True` when the profile is fully synced and operational |
| `Synced` | `True` when the profile spec has been successfully applied to the NextDNS API |
| `ReferencesResolved` | `True` when all referenced list resources (allowlist, denylist, TLD) exist and are ready |
| `ConfigImported` | `True` when the config import ConfigMap has been successfully read and merged |

### NextDNSAllowlist

A reusable list of domains to allow. Can be referenced by multiple `NextDNSProfile` resources via `allowlistRefs`.

#### Spec Fields

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

#### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `domainCount` | int | Number of active domains in this list |
| `profileRefs` | ResourceReference[] | Profiles currently using this allowlist |
| `conditions` | []Condition | Standard Kubernetes conditions |

### NextDNSDenylist

A reusable list of domains to block. Can be referenced by multiple `NextDNSProfile` resources via `denylistRefs`.

#### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `description` | string | No | | Human-readable description of this denylist |
| `domains` | DomainEntry[] | Yes (min 1) | | Domains to block |

#### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `domainCount` | int | Number of active domains in this list |
| `profileRefs` | ResourceReference[] | Profiles currently using this denylist |
| `conditions` | []Condition | Standard Kubernetes conditions |

### NextDNSTLDList

A reusable list of top-level domains to block. Can be referenced by multiple `NextDNSProfile` resources via `tldListRefs`.

#### Spec Fields

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

#### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `tldCount` | int | Number of active TLDs in this list |
| `profileRefs` | ResourceReference[] | Profiles currently using this TLD list |
| `conditions` | []Condition | Standard Kubernetes conditions |

### NextDNSCoreDNS

Deploys a CoreDNS instance configured to forward DNS queries to a NextDNS profile.

#### Spec Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `profileRef.name` | string | Yes | | Name of the NextDNSProfile to use |
| `profileRef.namespace` | string | No | | Namespace (defaults to same namespace) |
| `upstream.primary` | DNSProtocol | No | `DoT` | Upstream protocol: `DoT`, `DoH`, or `DNS` |
| `deployment.mode` | DeploymentMode | No | `Deployment` | `Deployment` or `DaemonSet` |
| `deployment.replicas` | *int32 | No | `2` | Replicas (Deployment mode only, min: 1) |
| `deployment.image` | string | No | `mirror.gcr.io/coredns/coredns:1.13.1` | CoreDNS container image |
| `deployment.nodeSelector` | map[string]string | No | | Node label selector |
| `deployment.affinity` | Affinity | No | | Pod scheduling constraints |
| `deployment.tolerations` | Toleration[] | No | | Pod tolerations |
| `deployment.resources` | ResourceRequirements | No | | CPU/memory requests and limits |
| `deployment.podAnnotations` | map[string]string | No | | Additional pod annotations (useful for Multus) |
| `service.type` | CoreDNSServiceType | No | `ClusterIP` | `ClusterIP`, `LoadBalancer`, or `NodePort` |
| `service.loadBalancerIP` | string | No | | Static IP for LoadBalancer (valid IPv4) |
| `service.annotations` | map[string]string | No | | Additional service annotations |
| `service.nameOverride` | string | No | | Custom service name |
| `metrics.enabled` | *bool | No | `true` | Enable Prometheus metrics endpoint |
| `metrics.serviceMonitor.enabled` | bool | No | `false` | Create Prometheus ServiceMonitor |
| `metrics.serviceMonitor.namespace` | string | No | | ServiceMonitor namespace |
| `metrics.serviceMonitor.interval` | string | No | `30s` | Scrape interval |
| `metrics.serviceMonitor.labels` | map[string]string | No | | Additional ServiceMonitor labels |
| `cache.enabled` | *bool | No | `true` | Enable DNS response caching |
| `cache.successTTL` | *int32 | No | `3600` | Cache TTL for successful responses (seconds) |
| `logging.enabled` | *bool | No | `false` | Enable DNS query logging |
| `domainOverrides` | DomainOverride[] | No | | Domain-specific upstream overrides |

Each `DomainOverride` has:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `domain` | string | Yes | | DNS domain to override (e.g., `corp.example.com`) |
| `upstreams` | string[] | Yes (min 1) | | Upstream DNS server IPs (IPv4 or IPv4:port) |
| `cacheTTL` | *int32 | No | | Cache TTL for this domain (seconds) |

#### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `profileID` | string | NextDNS profile ID from the referenced profile |
| `fingerprint` | string | DNS fingerprint from the referenced profile |
| `endpoints` | DNSEndpoint[] | DNS endpoints exposed by the service (`ip`, `port`, `protocol`) |
| `dnsIP` | string | Primary DNS IP address for easy reference |
| `upstream.url` | string | NextDNS upstream URL being used |
| `replicas.desired` | int32 | Desired replica count |
| `replicas.ready` | int32 | Ready replica count |
| `replicas.available` | int32 | Available replica count |
| `ready` | bool | Whether the CoreDNS deployment is fully ready |
| `conditions` | []Condition | Standard Kubernetes conditions |
| `lastUpdated` | Time | Last time the status was updated |
| `observedGeneration` | int64 | Generation last processed by the controller |

#### Conditions

| Type | Description |
|------|-------------|
| `Ready` | Overall readiness -- `True` when all CoreDNS resources (workload, service, configmap) are deployed and healthy |
| `ProfileResolved` | `True` when the referenced NextDNSProfile exists and is in Ready state |

---

## Status & Conditions

All CRDs use standard Kubernetes conditions to communicate their state. Conditions follow the [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties).

### Reading Conditions

```bash
# Check profile status
kubectl get nextdnsprofile my-profile -o yaml

# Quick check with jsonpath
kubectl get nextdnsprofile my-profile -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'

# Check CoreDNS status
kubectl get nextdnscoredns home-dns -o yaml
```

### NextDNSProfile Conditions

| Condition | True | False |
|-----------|------|-------|
| **Ready** | Profile is fully synced and operational | One or more subsystems have issues |
| **Synced** | Spec successfully applied to NextDNS API | API sync failed (check `message` for details) |
| **ReferencesResolved** | All referenced lists exist and are ready | One or more list references are missing or not ready |
| **ConfigImported** | ConfigMap read and merged successfully | Import failed (missing ConfigMap, invalid JSON, etc.) |

### NextDNSCoreDNS Conditions

| Condition | True | False |
|-----------|------|-------|
| **Ready** | All CoreDNS resources deployed and healthy | Workload, service, or configmap has issues |
| **ProfileResolved** | Referenced NextDNSProfile exists and is Ready | Profile not found or not in Ready state |

### Common Status Patterns

**Healthy profile:**
```yaml
conditions:
  - type: Ready
    status: "True"
  - type: Synced
    status: "True"
  - type: ReferencesResolved
    status: "True"
```

**Profile waiting for list references:**
```yaml
conditions:
  - type: Ready
    status: "False"
    reason: ReferencesNotResolved
  - type: ReferencesResolved
    status: "False"
    reason: AllowlistNotFound
    message: "Allowlist 'business-apps' not found in namespace 'default'"
```

**CoreDNS waiting for profile:**
```yaml
conditions:
  - type: Ready
    status: "False"
    reason: ProfileNotReady
  - type: ProfileResolved
    status: "False"
    reason: ProfileNotReady
    message: "Waiting for profile to become ready"
```

---

## Troubleshooting

### Profile Not Syncing

**Symptoms:** Profile shows `Ready: False` with `Synced: False`.

**Check:**
```bash
kubectl describe nextdnsprofile my-profile
```

**Common causes:**
1. **Invalid API key**: Verify the Secret exists and contains a valid key.
   ```bash
   kubectl get secret nextdns-credentials -o jsonpath='{.data.api-key}' | base64 -d
   ```
2. **API rate limiting**: The operator may be hitting NextDNS API rate limits. Check operator logs for 429 errors.
   ```bash
   kubectl logs -n nextdns-operator-system deploy/nextdns-operator -f
   ```
3. **Invalid profile ID**: If using `profileID` to adopt an existing profile, verify the ID exists in your NextDNS account.

### ConfigMap Import Failures

**Symptoms:** `ConfigImported` condition is `False`.

**Check:**
```bash
kubectl get nextdnsprofile my-profile -o jsonpath='{.status.conditions[?(@.type=="ConfigImported")]}'
```

**Common causes:**
1. **ConfigMap not found**: Verify the ConfigMap exists in the same namespace.
   ```bash
   kubectl get configmap base-profile-config
   ```
2. **Invalid JSON**: Check that the JSON in the ConfigMap is valid.
   ```bash
   kubectl get configmap base-profile-config -o jsonpath='{.data.config\.json}' | jq .
   ```
3. **Wrong key**: Ensure the `key` field in `configImportRef` matches the actual key in the ConfigMap data (default: `config.json`).

### CoreDNS Not Starting

**Symptoms:** `NextDNSCoreDNS` shows `Ready: false`.

**Check:**
```bash
kubectl describe nextdnscoredns home-dns
kubectl get pods -l app.kubernetes.io/managed-by=nextdns-operator
```

**Common causes:**
1. **Profile not ready**: The referenced profile must be in `Ready` state. Check the `ProfileResolved` condition.
   ```bash
   kubectl get nextdnsprofile my-profile -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
   ```
2. **Image pull errors**: Verify the CoreDNS image is accessible from your cluster.
   ```bash
   kubectl describe pod -l app.kubernetes.io/managed-by=nextdns-operator | grep -A5 Events
   ```
3. **Resource constraints**: If pods are pending, check for insufficient CPU/memory on nodes.
   ```bash
   kubectl get events --field-selector reason=FailedScheduling
   ```

### List References Not Resolving

**Symptoms:** `ReferencesResolved` condition is `False`.

**Check:**
```bash
kubectl get nextdnsprofile my-profile -o jsonpath='{.status.conditions[?(@.type=="ReferencesResolved")]}'
```

**Common causes:**
1. **List not found**: Verify the referenced list resource exists.
   ```bash
   kubectl get nextdnsallowlist,nextdnsdenylist,nextdnstldlist
   ```
2. **Wrong namespace**: If the list is in a different namespace, specify it in the reference.
   ```yaml
   allowlistRefs:
     - name: shared-allowlist
       namespace: dns-config
   ```
3. **List not ready**: The referenced list itself must have at least one domain/TLD entry (enforced by `MinItems=1` validation).

---

## Architecture

### Multi-CRD Architecture

```
                    ┌─────────────────────┐
                    │   NextDNSProfile    │
                    │                     │
                    │  - security         │
                    │  - privacy          │
                    │  - parentalControl  │
                    │  - settings         │
                    │  - inline lists     │
                    └────────┬────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
    ┌─────────▼──────┐ ┌────▼───────┐ ┌────▼───────┐
    │NextDNSAllowlist│ │NextDNS     │ │NextDNS     │
    │                │ │Denylist    │ │TLDList     │
    │ - domains[]    │ │            │ │            │
    │ - description  │ │ - domains[]│ │ - tlds[]   │
    └────────────────┘ └────────────┘ └────────────┘
              │              │              │
              └──────────────┼──────────────┘
                             │ shared across
                             │ multiple profiles
                             ▼
                    ┌─────────────────────┐
                    │   NextDNSCoreDNS   │
                    │                     │
                    │  - profileRef ──────┼──► NextDNSProfile
                    │  - upstream         │
                    │  - deployment       │
                    │  - service          │
                    └─────────────────────┘
```

List resources (`NextDNSAllowlist`, `NextDNSDenylist`, `NextDNSTLDList`) are **reusable** -- a single list can be referenced by multiple profiles. The profile controller merges domains from all referenced lists with inline entries.

### Reconciliation Flow

**NextDNSProfile reconciliation:**

1. Read the profile spec and resolve credentials from the referenced Secret
2. If `configImportRef` is set, read the ConfigMap and merge imported values with spec (spec takes precedence)
3. Resolve all list references (`allowlistRefs`, `denylistRefs`, `tldListRefs`) and merge domains
4. If `profileID` is set, adopt the existing profile; otherwise, create a new one
5. Apply the merged configuration to the NextDNS API
6. If `configMapRef.enabled`, create/update the ConfigMap with connection details
7. Update status with profile ID, fingerprint, aggregated counts, and conditions

**NextDNSCoreDNS reconciliation:**

1. Resolve the referenced `NextDNSProfile` and verify it is Ready
2. Generate a CoreDNS Corefile based on upstream config, cache settings, domain overrides, and profile fingerprint
3. Create/update the ConfigMap containing the Corefile
4. Create/update the Deployment or DaemonSet
5. Create/update the Service
6. Update status with endpoints, replica counts, and conditions

### How List References Work

When a profile references a list resource:

1. The profile controller watches for changes to referenced list resources
2. When a list changes, all profiles referencing it are re-reconciled
3. Domains from all referenced lists are merged with inline `allowlist`/`denylist` entries
4. Deduplication ensures no domain appears twice in the final list sent to the API
5. The `referencedResources` status field tracks each list's name, namespace, readiness, and item count

### How ConfigMap Import/Export Works

**Export** (`configMapRef`): After syncing a profile, the operator creates a ConfigMap containing the profile's DNS connection details (profile ID, DoT/DoH/DoQ endpoints, IPv4/IPv6 addresses). Other workloads can consume this ConfigMap via `envFrom` or volume mounts.

**Import** (`configImportRef`): Before syncing, the operator reads a ConfigMap containing profile configuration as JSON. Imported values act as defaults -- explicit spec fields always take precedence. The operator watches the ConfigMap's `ResourceVersion` and re-imports when it changes.
