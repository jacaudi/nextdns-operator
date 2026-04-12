# Multus CNI Integration

For advanced networking scenarios, you can attach CoreDNS pods to additional networks using [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni). The operator provides first-class support for Multus via the `spec.multus` field, making it easy to expose DNS services directly on a VLAN or dedicated network interface.

> For the full documentation index, see the [main docs page](README.md).

---

## Setup

**Example: CoreDNS on a VLAN with static IPs**

First, create a NetworkAttachmentDefinition for your VLAN. When using `spec.multus.ips` to request specific IPs per pod, the operator passes them to Multus via the annotation's `ips` field. Use an IPAM plugin that supports runtime IP requests, such as `whereabouts`:

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
        "type": "whereabouts",
        "range": "192.168.100.0/24",
        "gateway": "192.168.100.1"
      }
    }
```

> **Note:** The IPAM plugin in the NAD must support per-pod IP requests via the Multus `ips` annotation field. Plugins like `whereabouts` and `static` (without hardcoded addresses) support this. If you omit `spec.multus.ips`, the IPAM plugin assigns IPs from its configured range automatically.

Then reference it in your NextDNSCoreDNS resource using `spec.multus`:

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: vlan-dns
spec:
  profileRef:
    name: my-profile

  multus:
    networkAttachmentDefinition: dns-vlan
    ips:
      - 192.168.100.53
      - 192.168.100.54

  deployment:
    mode: DaemonSet

  service:
    type: ClusterIP  # Internal only; clients use Multus IPs directly

  corefile:
    upstream:
      primary: DoT
```

The operator automatically:
- Generates the `k8s.v1.cni.cncf.io/networks` annotation on pod templates in the correct Multus JSON format
- Reads `k8s.v1.cni.cncf.io/network-status` from running pods to discover assigned IPs
- Reports the Multus IPs in `status.multusIPs` and adds them to `status.endpoints`
- Warns if the number of static IPs is less than the number of replicas
- Validates that each IP is a valid IPv4 address

The CoreDNS pods will have interfaces on both the cluster network and the VLAN, accessible at `192.168.100.53` and `192.168.100.54`.

---

## MultusConfig Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `networkAttachmentDefinition` | string | Yes | Name of the existing NetworkAttachmentDefinition CR |
| `namespace` | string | No | Namespace of the NAD (defaults to the CR's namespace) |
| `ips` | string[] | No | Static IPs to request from IPAM (one per pod) |

> **Note:** If you set `k8s.v1.cni.cncf.io/networks` in `spec.deployment.podAnnotations` while also using `spec.multus`, the operator-managed value takes precedence and a warning is logged. Use one approach or the other.

---

## Interaction with Deployment Modes

Multus is most commonly used with `DaemonSet` mode so that each node gets its own CoreDNS pod with a node-local Multus IP. When using `DaemonSet` mode, the `replicas` field is ignored — one pod per matching node is scheduled automatically.

See [CoreDNS deployment modes](coredns.md#deployment-modes) for details on `Deployment` vs `DaemonSet`.
