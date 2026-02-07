# NextDNS Kubernetes Operator

A Kubernetes operator for managing [NextDNS](https://nextdns.io) profiles declaratively using Custom Resources.

## Features

- **Declarative DNS Management**: Define NextDNS profiles as Kubernetes resources
- **Multi-CRD Architecture**: Separate resources for allowlists, denylists, and TLD lists that can be shared across profiles
- **Full NextDNS API Coverage**: Security, privacy, parental control, and settings configuration
- **Profile Lifecycle Management**: Create new profiles or adopt existing ones; operator-created profiles are deleted on resource removal
- **Drift Detection**: Automatic periodic reconciliation (default: 1 hour) catches manual changes made outside the operator
- **ConfigMap Export**: Optionally create a ConfigMap with DNS connection details for easy integration with other applications
- **ConfigMap Import**: Import base profile configuration from a ConfigMap JSON, with spec fields taking precedence

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

## Documentation

For detailed configuration guides, CRD reference, troubleshooting, and architecture documentation, see the **[full documentation](docs/README.md)**.

Covers: ConfigMap export/import, CoreDNS deployment (upstream protocols, Multus CNI, domain overrides), drift detection, complete CRD field reference, status conditions, and troubleshooting.

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
