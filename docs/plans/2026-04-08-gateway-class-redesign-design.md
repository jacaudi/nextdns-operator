# GatewayClass Redesign: Consumer, Not Controller

## Overview

The original Gateway API integration (2026-04-07) incorrectly had the operator creating its own GatewayClass with `controllerName: nextdns.io/coredns-gateway`. The operator is not a gateway controller -- it doesn't implement the data plane. It should reference an existing GatewayClass managed by an external controller (Envoy Gateway, Cilium Gateway API, Istio, etc.).

This redesign removes GatewayClass creation and makes the operator a pure consumer of external gateway infrastructure.

## Goals

- Remove GatewayClass creation from the operator
- Allow users to specify which external GatewayClass to reference, per-CR or via operator default
- Add cleanup logic for gateway resources when `spec.gateway` is removed from a CR
- Maintain pluggability across gateway implementations (Envoy Gateway, Cilium, Istio, etc.)

## Non-Goals

- Creating or managing GatewayClass resources
- Supporting shared Gateways (operator still creates a dedicated Gateway per CR)
- Adding a `gatewayRef` mode to reference existing Gateway resources

## Design Decisions

1. **Consumer, not controller** -- the operator creates Gateway/TCPRoute/UDPRoute resources that reference an externally-managed GatewayClass. The external controller (e.g., Envoy Gateway) programs the data plane.
2. **Per-CR with operator default** -- `spec.gateway.gatewayClassName` overrides the operator-level default. This supports mixed environments (different controllers for different CRs) while keeping simple cases simple.
3. **Dedicated Gateway per CR** -- DNS on port 53 typically needs its own IP. Sharing a Gateway with HTTP traffic would be unusual. The operator creates one Gateway per NextDNSCoreDNS CR.
4. **Explicit cleanup** -- when `spec.gateway` is removed from a CR, the operator deletes the orphaned Gateway, TCPRoute, and UDPRoute resources and clears related conditions.
5. **No default GatewayClass name** -- the operator default is empty string. Users must configure it (via Helm/flag/env) or specify it per-CR. This prevents silently referencing a non-existent class.

## API Changes

### GatewayConfig (modified)

```go
type GatewayConfig struct {
    // GatewayClassName specifies which GatewayClass to use for the Gateway.
    // This must reference a GatewayClass managed by an external controller
    // (e.g., Envoy Gateway, Cilium, Istio).
    // If omitted, uses the operator's default gateway class name.
    // +optional
    GatewayClassName *string `json:"gatewayClassName,omitempty"`

    // Addresses specifies the IP addresses for the Gateway.
    // These are requested from the Gateway implementation.
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinItems=1
    Addresses []GatewayAddress `json:"addresses"`

    // Annotations specifies additional annotations for the Gateway resource.
    // +optional
    Annotations map[string]string `json:"annotations,omitempty"`
}
```

### CR Example

Using per-CR class name:

```yaml
spec:
  gateway:
    gatewayClassName: eg-nextdns
    addresses:
      - value: "10.10.21.53"
    annotations:
      external-dns.alpha.kubernetes.io/hostname: dns.example.com
```

Using operator default (no `gatewayClassName` in CR):

```yaml
spec:
  gateway:
    addresses:
      - value: "10.10.21.53"
```

## Startup Changes (cmd/main.go)

### Remove

- GatewayClass creation via `manager.RunnableFunc` (the `CreateOrUpdate` block for GatewayClass)
- GatewayClass creation log messages

### Keep

- Gateway API CRD detection via discovery client (still needed to know if Gateway/Route CRDs are available)
- `--gateway-class-name` flag and `GATEWAY_CLASS_NAME` env var (repurposed as default class to reference)
- Scheme registration for gateway-api types (still creating Gateway/Route resources)
- `GatewayAPIAvailable` and `GatewayClassName` fields passed to the reconciler

### Change

- Default value for `--gateway-class-name` changes from `"nextdns-coredns"` to `""` (empty)
- Startup log remains `"Gateway API CRDs detected, enabling gateway support"` but no longer followed by GatewayClass creation

## Reconciler Changes

### GatewayClass Name Resolution

`reconcileGateway` resolves the class name with this precedence:

1. `spec.gateway.gatewayClassName` (CR-level override)
2. `r.GatewayClassName` (operator default)
3. If neither is set, return error

### Validation (in reconcile loop)

When `spec.gateway` is set, validate that a GatewayClass name is resolvable before attempting reconciliation. If not, set condition:

```
Type:    GatewayReady
Status:  False
Reason:  NoGatewayClassName
Message: No gatewayClassName specified in spec.gateway and no operator default configured
```

### Gateway Resource Cleanup

When `spec.gateway` is nil (removed from CR), the reconcile loop:

1. Attempts to delete `<cr-name>-dns` Gateway (ignore NotFound)
2. Attempts to delete `<cr-name>-dns-tcp` TCPRoute (ignore NotFound)
3. Attempts to delete `<cr-name>-dns-udp` UDPRoute (ignore NotFound)
4. Removes conditions `GatewayReady`, `TCPRouteReady`, `UDPRouteReady` from `status.conditions` via `meta.RemoveStatusCondition`
5. Sets `status.gatewayReady = false`

Resource names are deterministic, so direct lookup + delete is safe and idempotent.

### No Changes Needed

- `reconcileTCPRoute` / `reconcileUDPRoute` -- these reference the Gateway by name, not the GatewayClass
- `updateGatewayStatus` -- reads Gateway status, unrelated to GatewayClass

## RBAC Changes

### Remove

```go
// - gatewayclasses verbs: get;list;watch;create;update;patch
```

### Keep

```go
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tcproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=udproutes,verbs=get;list;watch;create;update;patch;delete
```

## Helm Chart Changes

- `gatewayAPI.gatewayClassName` value stays, semantics change to "default GatewayClass to reference"
- Default value changes from `"nextdns-coredns"` to `""`
- Update comments in `values.yaml` to explain this references an external GatewayClass

## Sample CR Update

Update `config/samples/nextdns_v1alpha1_nextdnscoredns_gateway.yaml`:

```yaml
spec:
  gateway:
    gatewayClassName: envoy-gateway  # references an external GatewayClass
    addresses:
      - value: "192.168.1.53"
    annotations:
      external-dns.alpha.kubernetes.io/hostname: dns.example.com
```

## Test Changes

### New Test Cases

- CR-level `gatewayClassName` is used when set
- Operator default `GatewayClassName` is used as fallback
- Error when neither CR nor operator default specifies a class name
- Gateway cleanup when `spec.gateway` is removed (Gateway, TCPRoute, UDPRoute deleted)
- Conditions cleared after gateway cleanup

### Updated Test Cases

- Existing `TestReconcileGateway` updated to verify the GatewayClass name comes from the resolved source (not a hardcoded value)

## Cluster Cleanup (Operational)

After upgrading to this version, the existing `nextdns-coredns` GatewayClass must be manually deleted:

```bash
kubectl delete gatewayclass nextdns-coredns
```

It has no owner references, so it won't be garbage collected automatically.

## Pluggability Across Gateway Controllers

The design is controller-agnostic by construction:

| Controller | GatewayClass Name | How It Works |
|---|---|---|
| Envoy Gateway | `eg-nextdns` (user-created) | Envoy Gateway sees Gateway referencing its class, programs Envoy proxy |
| Cilium Gateway API | `cilium` | Cilium's built-in GatewayClass, programs Cilium's eBPF data plane |
| Istio | `istio` | Istio sees Gateway referencing its class, programs Envoy sidecar/gateway |
| Kong | `kong` | Kong's GatewayClass, programs Kong proxy |
| Contour | `contour` | Contour's GatewayClass, programs Envoy via Contour |

The operator doesn't need to know which controller is in use. It creates standard Gateway API resources with the specified class name, and the controller that owns that class handles the rest. This is the entire point of the Gateway API abstraction.
