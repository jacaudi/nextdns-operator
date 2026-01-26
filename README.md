# NextDNS Kubernetes Operator

A Kubernetes operator for managing [NextDNS](https://nextdns.io) profiles declaratively using Custom Resources.

## Features

- **Declarative DNS Management**: Define NextDNS profiles as Kubernetes resources
- **Multi-CRD Architecture**: Separate resources for allowlists, denylists, and TLD lists that can be shared across profiles
- **Full NextDNS API Coverage**: Security, privacy, parental control, and settings configuration
- **Profile Lifecycle Management**: Create new profiles or adopt existing ones; operator-created profiles are deleted on resource removal
- **Drift Detection**: Automatic periodic reconciliation (default: 1 hour) catches manual changes made outside the operator

## Custom Resources

| CRD | Description |
|-----|-------------|
| `NextDNSProfile` | Main profile configuration with security, privacy, and parental control settings |
| `NextDNSAllowlist` | Reusable list of allowed domains |
| `NextDNSDenylist` | Reusable list of blocked domains |
| `NextDNSTLDList` | Reusable list of blocked TLDs |

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

## Configuration

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

## License

Apache 2.0
