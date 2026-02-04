# NextDNS Kubernetes Operator

A Kubernetes operator for managing [NextDNS](https://nextdns.io) profiles declaratively using Custom Resources.

## Features

- **Declarative DNS Management**: Define NextDNS profiles as Kubernetes resources
- **Multi-CRD Architecture**: Separate resources for allowlists, denylists, and TLD lists that can be shared across profiles
- **Full NextDNS API Coverage**: Security, privacy, parental control, and settings configuration
- **Profile Lifecycle Management**: Create new profiles or adopt existing ones; operator-created profiles are deleted on resource removal
- **Drift Detection**: Automatic periodic reconciliation (default: 1 hour) catches manual changes made outside the operator
- **ConfigMap Export**: Optionally create a ConfigMap with DNS connection details for easy integration with other applications

## Custom Resources

| CRD | Description |
|-----|-------------|
| `NextDNSProfile` | Main profile configuration with security, privacy, and parental control settings |
| `NextDNSAllowlist` | Reusable list of allowed domains |
| `NextDNSDenylist` | Reusable list of blocked domains |
| `NextDNSTLDList` | Reusable list of blocked TLDs |
| `NextDNSCoreDNS` | Deploy CoreDNS instances forwarding to NextDNS upstream |

## Installation

### Helm (Recommended)

```bash
# Install from OCI registry
helm install nextdns-operator oci://ghcr.io/jacaudi/charts/nextdns-operator \
  --version 0.1.0 \
  --namespace nextdns-operator-system \
  --create-namespace
```

### Kubectl

```bash
# Install CRDs
kubectl apply -f https://github.com/jacaudi/nextdns-operator/releases/latest/download/install.yaml

# Deploy operator
kubectl apply -f https://github.com/jacaudi/nextdns-operator/releases/latest/download/operator.yaml
```

### Local Development

```bash
# Install CRDs
make install

# Run locally
make run
```

## Quick Start

Once the operator is installed:

1. **Create a Secret with your NextDNS API key:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nextdns-credentials
  namespace: default
type: Opaque
stringData:
  api-key: "your-nextdns-api-key"
```

2. **Create a NextDNSProfile:**

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSProfile
metadata:
  name: my-profile
  namespace: default
spec:
  name: "My DNS Profile"
  credentialsRef:
    name: nextdns-credentials
  security:
    aiThreatDetection: true
    googleSafeBrowsing: true
```

3. **Apply the resources:**

```bash
kubectl apply -f secret.yaml
kubectl apply -f profile.yaml
```

4. **Check the status:**

```bash
kubectl get nextdnsprofile my-profile -o yaml
```

## Examples

See the [config/samples](config/samples/) directory for complete examples:

- [NextDNSProfile](config/samples/nextdns_v1alpha1_nextdnsprofile.yaml) - Full profile with security, privacy, and settings
- [NextDNSAllowlist](config/samples/nextdns_v1alpha1_nextdnsallowlist.yaml) - Shared allowlist for business services
- [NextDNSDenylist](config/samples/nextdns_v1alpha1_nextdnsdenylist.yaml) - Shared denylist for malicious domains
- [NextDNSTLDList](config/samples/nextdns_v1alpha1_nextdnstldlist.yaml) - Shared list of high-risk TLDs
- [NextDNSCoreDNS](config/samples/nextdns_v1alpha1_nextdnscoredns.yaml) - CoreDNS deployment with NextDNS upstream

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

### CoreDNS Deployment

Deploy a dedicated CoreDNS instance that forwards DNS queries to NextDNS. This is useful for providing DNS services to devices on your network (home routers, IoT devices, etc.) that can't use DoH/DoT directly.

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
    fallback: DoH     # Fallback to DNS over HTTPS

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

**Features:**

- **Upstream protocols**: DoT (DNS over TLS), DoH (DNS over HTTPS), or plain DNS
- **Deployment modes**: Kubernetes Deployment (with replicas) or DaemonSet
- **Service types**: ClusterIP, LoadBalancer, or NodePort
- **Placement controls**: nodeSelector, affinity, tolerations
- **Caching**: Configurable DNS response caching
- **Metrics**: Prometheus metrics endpoint with optional ServiceMonitor
- **Security**: Containers run read-only with dropped capabilities

**Check deployment status:**

```bash
kubectl get nextdnscoredns home-dns
# NAME       PROFILE ID   DNS IP          READY   AGE
# home-dns   abc123       192.168.1.53    true    5m
```

> **Security Note:** Using plain DNS (`DNS` protocol) exposes your NextDNS profile ID in unencrypted traffic. Use DoT or DoH for privacy in untrusted networks.

### Drift Detection

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
- Syncs include ±10% jitter to prevent all resources from hitting the API simultaneously
- Each profile makes ~1 API call per sync period
- List resources (allowlist, denylist, tldlist) sync status but don't call the NextDNS API directly
- Setting to `0` disables periodic syncing (event-driven only)

## Development

```bash
# Run tests
make test

# Build
make build
```

## Acknowledgements

This project stands on the shoulders of giants:

- **[bjw-s](https://github.com/bjw-s)** - For the excellent [helm-charts](https://github.com/bjw-s-labs/helm-charts) library and app-template that powers the Helm chart for this operator. The common library pattern has been invaluable.

- **[amalucelli](https://github.com/amalucelli)** - For creating the original [nextdns-go](https://github.com/amalucelli/nextdns-go) client library that this operator's fork is based on. The solid foundation made building this operator possible.

## License

Apache 2.0
