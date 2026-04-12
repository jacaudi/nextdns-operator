# Gateway API

As an alternative to LoadBalancer, DNS traffic can be exposed through Gateway API using TCPRoute and UDPRoute resources. The operator creates a dedicated Gateway per NextDNSCoreDNS CR and attaches routes for port 53 TCP and UDP traffic.

> For the full documentation index, see the [main docs page](README.md).

---

## Setup

Gateway API requires an external gateway controller (e.g., Envoy Gateway, Cilium, Istio) to be installed in the cluster. The operator references a GatewayClass managed by that controller — it does not implement its own gateway data plane.

**Gateway and LoadBalancer are mutually exclusive.** When `gateway` is configured, the operator forces the Service type to ClusterIP (used as the route backend).

```yaml
spec:
  gateway:
    gatewayClassName: envoy-gateway  # references an external GatewayClass
    addresses:
      - value: "192.168.1.53"        # IP requested from the gateway implementation
    annotations:
      external-dns.alpha.kubernetes.io/hostname: dns.example.com
```

Multiple addresses are supported for high availability or dual-stack deployments:

```yaml
spec:
  gateway:
    gatewayClassName: envoy
    addresses:
      - value: "10.10.21.81"
      - value: "10.10.21.82"
```

To use a hostname type address instead of an IP, set the `type` field explicitly:

```yaml
spec:
  gateway:
    gatewayClassName: envoy
    addresses:
      - type: Hostname
        value: "dns.example.com"
      - type: IPAddress   # default when type is omitted
        value: "192.168.1.53"
```

The `gatewayClassName` can be specified per-CR or set as an operator-level default via the `--gateway-class-name` flag or `GATEWAY_CLASS_NAME` environment variable. If set per-CR, it overrides the operator default.

**Supported gateway controllers:**

| Controller | GatewayClass Name | Notes |
|---|---|---|
| Envoy Gateway | User-created (e.g., `eg-nextdns`) | Programs Envoy proxy |
| Cilium | `cilium` | Uses eBPF data plane |
| Istio | `istio` | Programs Envoy gateway |
| Kong | `kong` | Programs Kong proxy |
| Contour | `contour` | Programs Envoy via Contour |

---

## Infrastructure

The `infrastructure` field propagates metadata to resources created by the gateway implementation (e.g., the LoadBalancer Service that Envoy Gateway creates). This is useful for passing annotations or labels that the gateway controller's generated resources need.

For example, when using Envoy Gateway with Cilium LB IPAM, the `lbipam.cilium.io/ips` annotation must reach the generated LoadBalancer Service for IP allocation:

```yaml
spec:
  gateway:
    gatewayClassName: envoy
    addresses:
      - value: "10.10.21.81"
      - value: "10.10.21.82"
    infrastructure:
      annotations:
        lbipam.cilium.io/ips: "10.10.21.81,10.10.21.82"
```

Labels and a `parametersRef` (for implementation-specific configuration) are also supported:

```yaml
spec:
  gateway:
    gatewayClassName: envoy
    addresses:
      - value: "192.168.1.53"
    infrastructure:
      annotations:
        lbipam.cilium.io/ips: "192.168.1.53"
      labels:
        app.kubernetes.io/managed-by: nextdns-operator
      parametersRef:
        group: gateway.envoyproxy.io
        kind: EnvoyProxy
        name: custom-proxy-config
```

---

## Proxy Replicas

`deployment.replicas` and `gateway.replicas` control two separate tiers:

- `deployment.replicas` — CoreDNS pod count (the DNS server itself)
- `gateway.replicas` — gateway proxy pod count (e.g., Envoy pods managed by Envoy Gateway)

Use `gateway.replicas` to horizontally scale the proxy tier without manually creating implementation-specific CRs. For Envoy Gateway, the operator auto-generates an `EnvoyProxy` CR and wires it via `infrastructure.parametersRef`:

```yaml
spec:
  deployment:
    replicas: 2              # CoreDNS pods
  gateway:
    gatewayClassName: envoy-gateway
    addresses:
      - value: "192.168.1.53"
    replicas: 2              # Envoy proxy pods
    infrastructure:
      annotations:
        lbipam.cilium.io/ips: "192.168.1.53"
```

When `gateway.replicas` is set with an Envoy Gateway GatewayClass, the operator:
1. Looks up the `GatewayClass` to detect the controller implementation
2. Creates or updates an `EnvoyProxy` CR named `{cr-name}-envoyproxy` in the same namespace
3. Automatically sets `infrastructure.parametersRef` on the Gateway to reference it

**`gateway.replicas` and `infrastructure.parametersRef` are mutually exclusive.** If both are set, the operator sets `GatewayReady=False` with reason `InvalidConfiguration` and does not create any resources. Use one or the other.

**Unsupported implementations:** If the GatewayClass `controllerName` is not recognized, the operator sets a warning condition (`GatewayReplicasUnsupported`) and skips the `replicas` field. Create the implementation-specific configuration manually and reference it via `infrastructure.parametersRef`.

Currently supported for `gateway.replicas`:
- Envoy Gateway (`gateway.envoyproxy.io/gatewayclass-controller`)

---

## Status and Cleanup

**Status:** When the Gateway is programmed by the controller, `status.gatewayReady` becomes `true` and `status.endpoints` is populated with the assigned addresses. If the Gateway is not yet programmed, the operator falls back to reporting the requested addresses from the spec.

**Cleanup:** If `spec.gateway` is removed from a CR, the operator deletes the orphaned Gateway, TCPRoute, and UDPRoute resources and clears the gateway-related conditions.
