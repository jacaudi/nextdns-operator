# NextDNS Kubernetes Operator

A Kubernetes operator for managing [NextDNS](https://nextdns.io) profiles declaratively using Custom Resources.

## Features

- **Declarative DNS Management**: Define NextDNS profiles as Kubernetes resources
- **Multi-CRD Architecture**: Separate resources for allowlists, denylists, and TLD lists that can be shared across profiles
- **Full NextDNS API Coverage**: Security, privacy, parental control, and settings configuration
- **Profile Lifecycle Management**: Create new profiles or adopt existing ones; operator-created profiles are deleted on resource removal

## Custom Resources

| CRD | Description |
|-----|-------------|
| `NextDNSProfile` | Main profile configuration with security, privacy, and parental control settings |
| `NextDNSAllowlist` | Reusable list of allowed domains |
| `NextDNSDenylist` | Reusable list of blocked domains |
| `NextDNSTLDList` | Reusable list of blocked TLDs |

## Quick Start

1. **Create a Secret with your NextDNS API key:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nextdns-credentials
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
spec:
  name: "My DNS Profile"
  credentialsRef:
    name: nextdns-credentials
  security:
    aiThreatDetection: true
    googleSafeBrowsing: true
```

## Examples

See the [config/samples](config/samples/) directory for complete examples:

- [NextDNSProfile](config/samples/nextdns_v1alpha1_nextdnsprofile.yaml) - Full profile with security, privacy, and settings
- [NextDNSAllowlist](config/samples/nextdns_v1alpha1_nextdnsallowlist.yaml) - Shared allowlist for business services
- [NextDNSDenylist](config/samples/nextdns_v1alpha1_nextdnsdenylist.yaml) - Shared denylist for malicious domains
- [NextDNSTLDList](config/samples/nextdns_v1alpha1_nextdnstldlist.yaml) - Shared list of high-risk TLDs

## Installation

### Helm (Recommended)

The chart is based on the [bjw-s app-template](https://bjw-s-labs.github.io/helm-charts/docs/app-template/).

```bash
# Add the bjw-s Helm repository (required dependency)
helm repo add bjw-s https://bjw-s-labs.github.io/helm-charts

# Install with Helm
helm install nextdns-operator ./chart
```

### Manual

```bash
# Install CRDs
make install

# Run the operator
make run
```

## Development

```bash
# Run tests
make test

# Build
make build
```

## License

Apache 2.0
