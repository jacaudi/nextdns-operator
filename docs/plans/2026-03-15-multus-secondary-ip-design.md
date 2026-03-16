# Multus Secondary IP Support for CoreDNS Deployments

**Issue:** [#54 — Explore adding secondary IP support to CoreDNS deployments](https://github.com/jacaudi/nextdns-operator/issues/54)
**Date:** 2026-03-15

> **For Claude:** REQUIRED SUB-SKILLS (must use in order):
> 1. `superpowers:using-git-worktrees` — Isolate work in a dedicated worktree
> 2. Choose execution mode (load `superpowers:test-driven-development` alongside whichever is chosen — all agents/sessions must use TDD):
>    - **Subagent-Driven (this session):** `superpowers:subagent-driven-development` + `superpowers:test-driven-development` — Dispatch fresh subagent per task, review between tasks
>    - **Parallel Session (separate):** `superpowers:executing-plans` + `superpowers:test-driven-development` — Batch execution with checkpoints
> 3. `superpowers:verification-before-completion` — Verify all tests pass before claiming done
> 4. `superpowers:requesting-code-review` — Code review after EACH task (built into subagent-driven; must be explicitly invoked after every task when using executing-plans)
> 5. After ALL tasks: dispatch independent and comprehensive code review on full diff (automatic in subagent-driven; must be explicitly dispatched when using executing-plans)
> 6. `superpowers:finishing-a-development-branch` — Complete the branch

## Summary

Add first-class Multus CNI support to the NextDNSCoreDNS CRD, allowing CoreDNS pods to receive secondary network interfaces with static IPs on a specified VLAN. This enables CoreDNS to be reachable on multiple IPs across different networks without requiring multiple Kubernetes Services.

## Motivation

In home/office network setups, DNS servers need to be reachable by devices on the local network. Rather than relying on a LoadBalancer Service (which introduces an extra hop and requires an LB controller), Multus can attach CoreDNS pods directly to a VLAN with their own IPs. Clients configure both IPs as DNS servers for redundancy.

## Design

### Use Case

- 2 CoreDNS replicas with `ClusterIP` Service for internal cluster traffic
- Multus attaches each pod to VLAN 30 via an existing `NetworkAttachmentDefinition`
- IPAM assigns one IP per pod from a static list (`10.10.30.100`, `10.10.30.101`)
- CoreDNS listens on `0.0.0.0:53`, so it serves DNS on both the cluster and VLAN interfaces
- Clients on the LAN configure both IPs as DNS servers

### CRD API Changes

New types added to `api/v1alpha1/nextdnscoredns_types.go`:

```go
// MultusConfig configures secondary network attachment via Multus CNI
type MultusConfig struct {
    // NetworkAttachmentDefinition is the name of the existing
    // NetworkAttachmentDefinition CR to attach to CoreDNS pods
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    NetworkAttachmentDefinition string `json:"networkAttachmentDefinition"`

    // Namespace is the namespace of the NetworkAttachmentDefinition
    // Defaults to the namespace of the NextDNSCoreDNS resource
    // +optional
    Namespace string `json:"namespace,omitempty"`

    // IPs is an optional list of static IPs to request from IPAM
    // When specified, the IPAM plugin assigns one per pod from this list
    // The number of IPs should be >= the number of replicas
    // +optional
    IPs []string `json:"ips,omitempty"`
}
```

New field on `NextDNSCoreDNSSpec`:

```go
// Multus configures a secondary network interface via Multus CNI
// +optional
Multus *MultusConfig `json:"multus,omitempty"`
```

New status field on `NextDNSCoreDNSStatus`:

```go
// MultusIPs lists the IPs assigned to pods via Multus
// +optional
MultusIPs []string `json:"multusIPs,omitempty"`
```

### Example CR

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: home-dns
  namespace: default
spec:
  profileRef:
    name: my-profile
  upstream:
    primary: DoT
  deployment:
    replicas: 2
  service:
    type: ClusterIP
  multus:
    networkAttachmentDefinition: vlan30-macvlan
    ips:
      - 10.10.30.100
      - 10.10.30.101
  cache:
    enabled: true
  metrics:
    enabled: true
```

### Controller Behavior

#### Multus Annotation Generation

When `spec.multus` is set, the controller generates the `k8s.v1.cni.cncf.io/networks` pod annotation in JSON format during workload reconciliation (Deployment/DaemonSet pod template).

With static IPs:
```json
[{"name": "vlan30-macvlan", "namespace": "default", "ips": ["10.10.30.100", "10.10.30.101"]}]
```

Without static IPs (IPAM assigns from its configured range):
```json
[{"name": "vlan30-macvlan", "namespace": "default"}]
```

The operator merges this with any existing `spec.deployment.podAnnotations`. If the user manually sets `k8s.v1.cni.cncf.io/networks` in `podAnnotations`, the operator-managed Multus config takes precedence and a warning is logged.

#### Validation

At reconciliation time:
- If `spec.multus.ips` is specified and `spec.deployment.replicas` is set, warn if `len(ips) < replicas` (more pods than IPs could cause IPAM failures)
- IPs are validated as IPv4 addresses via kubebuilder regex validation
- `networkAttachmentDefinition` name is non-empty (enforced by `MinLength=1`)
- No validation that the NetworkAttachmentDefinition exists — CNI reports errors at pod scheduling time via pod events

#### Status Reporting

The controller reads `k8s.v1.cni.cncf.io/network-status` annotations from running pods (populated by Multus after pod start) to extract assigned IPs. These are:
- Stored in `status.multusIPs`
- Appended to `status.endpoints` as additional `DNSEndpoint` entries (UDP + TCP on port 53)

### What This Design Does NOT Do

- Create or manage `NetworkAttachmentDefinition` CRs (separation of concerns)
- Validate that the referenced `NetworkAttachmentDefinition` exists (CNI's responsibility)
- Assign specific IPs to specific pods (IPAM plugin's responsibility)
- Modify CoreDNS configuration (already listens on `0.0.0.0:53`)
