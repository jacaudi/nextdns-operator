# NextDNS Kubernetes Operator

A Kubernetes operator for managing [NextDNS](https://nextdns.io) profiles declaratively using Custom Resources.

## Features

- Declarative DNS management as Kubernetes resources
- Multi-CRD architecture (shared allowlists, denylists, TLD lists)
- Full NextDNS API coverage
- Profile lifecycle management (create, adopt, delete)
- Automatic drift detection
- ConfigMap export for app integration
- Observe mode for safe profile adoption
- CoreDNS plugin extensibility (rewrite, hosts, forward tuning, health/ready/errors/metrics config via `spec.corefile`)
- Gateway API support (TCPRoute/UDPRoute) for DNS traffic exposure, including proxy replica control (`spec.gateway.replicas`)

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
# Install from OCI registry (installs latest release)
helm install nextdns-operator oci://ghcr.io/jacaudi/charts/nextdns-operator \
  --namespace nextdns-operator-system \
  --create-namespace
```

### Local Development

```bash
# Install CRDs
task install

# Deploy operator
task deploy
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
- [NextDNSProfile (observe mode)](config/samples/nextdns_v1alpha1_nextdnsprofile_observe.yaml) - Profile in observe-only mode for safe adoption
- [NextDNSAllowlist](config/samples/nextdns_v1alpha1_nextdnsallowlist.yaml) - Shared allowlist for business services
- [NextDNSDenylist](config/samples/nextdns_v1alpha1_nextdnsdenylist.yaml) - Shared denylist for malicious domains
- [NextDNSTLDList](config/samples/nextdns_v1alpha1_nextdnstldlist.yaml) - Shared list of high-risk TLDs
- [NextDNSCoreDNS](config/samples/nextdns_v1alpha1_nextdnscoredns.yaml) - CoreDNS deployment with NextDNS upstream
- [NextDNSCoreDNS (advanced)](config/samples/nextdns_v1alpha1_nextdnscoredns_advanced.yaml) - Advanced CoreDNS sample showcasing all plugin configuration
- [NextDNSCoreDNS with Gateway](config/samples/nextdns_v1alpha1_nextdnscoredns_gateway.yaml) - CoreDNS with Gateway API exposure

## Documentation

For detailed configuration guides, CRD reference, troubleshooting, and architecture documentation, see the **[full documentation](docs/README.md)**.

| Page | Covers |
|------|--------|
| [docs/README.md](docs/README.md) | Documentation index, breaking change callout (v0.18.0), drift detection, troubleshooting, architecture and reconciliation flow |
| [docs/profile-configuration.md](docs/profile-configuration.md) | ConfigMap export, observe mode, transitioning from observe to managed |
| [docs/coredns.md](docs/coredns.md) | CoreDNS deployment modes, upstream protocols, `spec.corefile` grouping, cache, metrics, health, ready, errors, query logging, forward tuning, domain overrides, static hosts, query rewriting |
| [docs/multus.md](docs/multus.md) | Multus CNI integration, NAD setup, static IPs, status reporting |
| [docs/gateway.md](docs/gateway.md) | Gateway API setup, infrastructure field, proxy replica control (`spec.gateway.replicas`) |
| [docs/reference.md](docs/reference.md) | Complete CRD field reference for all 5 CRDs, status fields, and conditions |

## Development

```bash
# Run tests
task test

# Build
task build
```

## Acknowledgements

This project stands on the shoulders of giants:

- **[bjw-s](https://github.com/bjw-s)** - For the excellent [helm-charts](https://github.com/bjw-s-labs/helm-charts) library and app-template that powers the Helm chart for this operator. The common library pattern has been invaluable.

- **[amalucelli](https://github.com/amalucelli)** - For creating the original [nextdns-go](https://github.com/amalucelli/nextdns-go) client library that this operator's fork is based on. The solid foundation made building this operator possible.

## License

Apache 2.0
