# CoreDNS Deployment

Deploy a dedicated CoreDNS instance that forwards DNS queries to NextDNS using the `NextDNSCoreDNS` custom resource. This is useful for providing DNS services to devices on your network (home routers, IoT devices, etc.) that can't use DoH/DoT directly.

> For the full documentation index, see the [main docs page](README.md).

> **Breaking change in v0.18.0:** Plugin-level fields (`upstream`, `cache`, `metrics`, `logging`, `domainOverrides`) are now grouped under `spec.corefile`. Manifests using the old top-level form will be rejected by CRD validation. See [#122](https://github.com/jacaudi/nextdns-operator/issues/122) for the migration.
>
> <details>
> <summary>Before / after example</summary>
>
> **Before (v0.17.x):**
>
> ```yaml
> spec:
>   profileRef:
>     name: my-profile
>   upstream:
>     primary: DoT
>   cache:
>     enabled: true
> ```
>
> **After (v0.18.0+):**
>
> ```yaml
> spec:
>   profileRef:
>     name: my-profile
>   corefile:
>     upstream:
>       primary: DoT
>     cache:
>       enabled: true
> ```
>
> Kubernetes-level fields (`profileRef`, `deployment`, `service`, `multus`, `gateway`) stay at the top level.
> </details>

---

## Basic Setup

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: home-dns
spec:
  profileRef:
    name: my-profile  # References an existing NextDNSProfile

  deployment:
    mode: Deployment
    replicas: 2

  service:
    type: LoadBalancer
    loadBalancerIP: "192.168.1.53"  # Optional static IP

  corefile:
    upstream:
      primary: DoT      # DNS over TLS (recommended)
    cache:
      enabled: true
      successTTL: 3600  # Cache TTL in seconds
```

`spec.corefile` is fully optional — omit the block entirely and the operator applies sensible defaults (DoT upstream, cache enabled with 3600s TTL, metrics enabled, logging disabled). A minimal manifest needs only `profileRef` plus whatever Kubernetes-level exposure (`service` or `gateway`) you want.

**Check deployment status:**

```bash
kubectl get nextdnscoredns home-dns
# NAME       PROFILE ID   DNS IP          READY   AGE
# home-dns   abc123       192.168.1.53    true    5m
```

---

## Upstream Protocols

The `corefile.upstream.primary` field controls how CoreDNS connects to NextDNS. Three protocols are supported:

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
corefile:
  upstream:
    primary: DoT  # DoT, DoH, or DNS
```

---

## Deployment Modes

CoreDNS can be deployed as either a **Deployment** or a **DaemonSet**:

**Deployment** (default): Runs a configurable number of replicas. Best for most use cases where you want centralized DNS serving with horizontal scaling.

```yaml
deployment:
  mode: Deployment
  replicas: 2  # default: 2, minimum: 1
```

**DaemonSet**: Runs one CoreDNS pod on every matching node (or every node if no nodeSelector is set). Best for scenarios where you want DNS available on every node, such as when using [Multus CNI](multus.md) to expose DNS on node-local network interfaces.

```yaml
deployment:
  mode: DaemonSet
  # replicas is ignored in DaemonSet mode
```

---

## Service Configuration

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

**Name override**: By default, the service is named after the NextDNSCoreDNS resource. Use `nameOverride` to set a custom name:

```yaml
service:
  nameOverride: my-dns-service
```

For Gateway API-based exposure (alternative to LoadBalancer), see [gateway.md](gateway.md).

---

## Caching

CoreDNS caching is enabled by default with a 3600-second (1 hour) TTL for successful responses. The cache respects upstream TTL values — if the upstream response has a lower TTL, that value is used instead.

```yaml
corefile:
  cache:
    enabled: true       # default: true
    successTTL: 3600    # default: 3600 (seconds)
```

To disable caching:

```yaml
corefile:
  cache:
    enabled: false
```

Setting `successTTL: 0` keeps the cache enabled but uses only upstream TTL values without overriding.

---

## Metrics & Monitoring

CoreDNS exposes a Prometheus metrics endpoint on port 9153 by default. The listen port is configurable if you need to avoid a conflict with a sidecar or another container.

```yaml
corefile:
  metrics:
    enabled: true  # default: true
    port: 9153     # default: 9153
```

The configured port applies to the CoreDNS `prometheus` plugin listener. If you change it, remember to update any ServiceMonitor / scrape config that targets CoreDNS so it talks to the new port.

> **Note:** ServiceMonitor for Prometheus Operator is configured via Helm values, not the CRD. See the Helm chart `values.yaml` for ServiceMonitor configuration.

---

## Health Plugin (Liveness)

The CoreDNS [`health`](https://coredns.io/plugins/health/) plugin serves the HTTP endpoint used for the pod's Kubernetes liveness probe. The operator keeps the deployment's liveness probe in sync with this configuration automatically — if you change `corefile.health.port`, the probe port is updated to match.

```yaml
corefile:
  health:
    enabled: true    # default: true
    port: 8080       # default: 8080
    lameduck: 10s    # optional, omit to skip the directive
```

- `enabled: false` removes both the `health` plugin directive from the Corefile AND the deployment's `livenessProbe`. Use this only in niche scenarios where you have an alternative liveness strategy.
- `lameduck` delays health endpoint failure during shutdown so load balancers (including upstream Gateway implementations) can drain traffic cleanly. Must be a Go duration string such as `10s`, `500ms`, or `2m`.
- `port` must differ from `corefile.ready.port` and `corefile.metrics.port`. The operator rejects colliding configurations at reconcile time.

---

## Ready Plugin (Readiness)

The CoreDNS [`ready`](https://coredns.io/plugins/ready/) plugin serves the HTTP endpoint used for the pod's Kubernetes readiness probe. As with `health`, the deployment's readiness probe tracks this configuration automatically.

```yaml
corefile:
  ready:
    enabled: true   # default: true
    port: 8181      # default: 8181
```

- `enabled: false` removes both the `ready` plugin directive and the deployment's `readinessProbe`. This is almost never what you want in production.
- `port` must differ from `corefile.health.port` and `corefile.metrics.port`.

---

## Errors Plugin

The CoreDNS [`errors`](https://coredns.io/plugins/errors/) plugin logs DNS resolution errors to stderr. It is enabled by default. The optional `consolidate` directive reduces log spam by collapsing repeated error messages matching a pattern within a time window.

```yaml
corefile:
  errors:
    enabled: true  # default: true
    consolidate:
      - interval: 5m
        pattern: "^[a-z]+error"
```

Each consolidate rule needs both `interval` (a Go duration string) and `pattern` (a regular expression matched against log lines). When `consolidate` is empty, the plugin emits a bare `errors` directive — the pre-feature default.

---

## Query Logging

Enable CoreDNS query logging for debugging DNS resolution issues. Disabled by default to reduce log volume.

```yaml
corefile:
  logging:
    enabled: true  # default: false
```

When enabled, CoreDNS logs all incoming DNS queries to stdout. This is useful for debugging but can generate significant log volume in production.

---

## Resource Requirements

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

---

## Device Identification

Identify your CoreDNS instance in NextDNS Analytics and Logs using the optional `corefile.upstream.deviceName` field. When set, the device name is embedded in the upstream DNS endpoint so NextDNS can attribute queries to a specific deployment.

> **Note:** Device identification only works with DoT (via SNI) and DoH (via URL path). When using plain DNS protocol, `deviceName` is ignored and a `DeviceNameIgnored` warning condition is set on the CR.

```yaml
corefile:
  upstream:
    primary: DoT
    deviceName: home-router
```

**How it works per protocol:**

| Protocol | Behavior | Example endpoint |
|----------|----------|-----------------|
| DoT | Device name prepended to SNI hostname; spaces converted to `--` | `home-router-abc123.dns.nextdns.io` |
| DoH | Device name URL-encoded and appended to path | `https://dns.nextdns.io/abc123/home-router` |
| DNS | Ignored (plain DNS has no mechanism for device identification) | `45.90.28.0` |

**Naming rules:**
- Only alphanumeric characters, hyphens, and spaces are allowed
- Maximum 63 characters
- Spaces are converted to `--` for DoT (SNI hostname) and URL-encoded (`%20`) for DoH
- The same device name is used for all pods in the deployment

---

## Forward Plugin Tuning

The CoreDNS [`forward` plugin](https://coredns.io/plugins/forward/) forwards DNS queries to NextDNS. By default it uses CoreDNS built-in defaults for failover policy, concurrency, health-check interval, and connection expiration. Use `spec.corefile.upstream.forward` to override these when you need tighter control over upstream behavior.

The entire `forward` block is optional — omitting it leaves CoreDNS defaults in effect and the forward directive shape is unchanged.

```yaml
corefile:
  upstream:
    primary: DoT
    forward:
      policy: round_robin     # random | round_robin | sequential
      maxConcurrent: 1000     # cap concurrent upstream queries (min 1)
      healthCheck: 5s         # upstream health check interval (Go duration)
      expire: 30s             # close idle upstream connections after (Go duration)
      maxFails: 2             # failed health checks before marking upstream down
```

**Tunable options:**

| Field | Default (CoreDNS) | Description |
|-------|------------------|-------------|
| `policy` | `random` | Failover policy when multiple upstreams are configured. `random` spreads load randomly; `round_robin` distributes in order; `sequential` always tries upstreams in declared order. |
| `maxConcurrent` | unlimited | Maximum number of concurrent queries forwarded upstream. Use to prevent thundering-herd on busy resolvers. Minimum 1. |
| `healthCheck` | `500ms` | Interval between upstream health checks. Shorter intervals detect failures faster at the cost of more health-check traffic. Must be a Go duration string (e.g. `5s`, `500ms`). |
| `expire` | `10s` | How long idle upstream connections are kept open before being closed. Must be a Go duration string. |
| `maxFails` | `2` | Number of consecutive failed health checks before an upstream is marked down. Set to `0` to disable marking upstreams down. |

> **Note:** When `forward.policy`, `healthCheck`, or `expire` contain invalid values (unknown policy name or unparseable duration), the controller rejects the configuration and surfaces an error condition on the CR — no Corefile is generated until the issue is resolved.

---

## Domain Overrides

Configure domain-specific DNS upstream servers for split-horizon DNS:

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: home-dns
spec:
  profileRef:
    name: my-profile
  corefile:
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

## Static Hosts

Use `spec.corefile.hosts` to express inline static hostname-to-IP overrides without running a separate upstream DNS server. This maps to the CoreDNS [hosts plugin](https://coredns.io/plugins/hosts/).

**When to use hosts vs domainOverrides:**
- Use `hosts` for individual FQDNs that should always resolve to a specific IP (e.g., an internal service with a stable IP).
- Use `domainOverrides` when an entire DNS zone should be forwarded to a different resolver (e.g., `corp.example.com` → internal DNS server).

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: home-dns
spec:
  profileRef:
    name: my-profile
  corefile:
    hosts:
      entries:
        - ip: 192.168.1.100
          hostnames:
            - grafana.internal
            - grafana.example.com
        - ip: 192.168.1.101
          hostnames:
            - prometheus.internal
      fallthrough: true   # default — unmatched names resolve via NextDNS
      ttl: 3600           # optional, CoreDNS default if omitted
```

**Plugin ordering:** The `hosts` block fires before `forward` in the generated Corefile. When `fallthrough: true` (the default), any hostname not found in the static entries is passed to the next plugin (forward → NextDNS). Set `fallthrough: false` to return NXDOMAIN for unmatched names.

---

## Query Rewriting

Use `spec.corefile.rewrite` to rewrite DNS query names before they are forwarded to NextDNS. This uses the CoreDNS [`rewrite` plugin](https://coredns.io/plugins/rewrite/) and is useful for CNAME flattening, domain remapping, and subdomain canonicalization.

**When to use rewrite vs domainOverrides:**
- Use `domainOverrides` to forward a domain to a *different upstream server* (e.g., an internal DNS server).
- Use `rewrite` to change the *query name itself* before forwarding to the same upstream (NextDNS). For example, rewriting `service.example.com` to `ingress.cluster.local` so NextDNS resolves the canonical name.

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: home-dns
spec:
  profileRef:
    name: my-profile
  corefile:
    rewrite:
      # Exact match: rewrite a specific name to a different name
      - type: name
        match: service.example.com
        replacement: ingress.cluster.local
      # Suffix match: rewrite all names ending in .old.example.com
      - type: name
        matcher: suffix
        match: .old.example.com
        replacement: .new.example.com
```

This generates the following directives in the catch-all server block, **before** the `forward` directive (CoreDNS plugin evaluation order requires rewrites to fire first):

```
. {
    rewrite name service.example.com ingress.cluster.local
    rewrite name suffix .old.example.com .new.example.com
    forward . tls://45.90.28.0 tls://45.90.30.0 {
        tls_servername profileid.dns.nextdns.io
    }
    ...
}
```

**Supported `type` values:** `name`, `class`, `type`, `ttl`, `edns0`. **Supported `matcher` values (for `type: name`):** `exact` (default), `prefix`, `suffix`, `substring`, `regex`.

See the [CoreDNS rewrite plugin documentation](https://coredns.io/plugins/rewrite/) for the full rule syntax.
