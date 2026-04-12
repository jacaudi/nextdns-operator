# NextDNS Operator Documentation

Comprehensive documentation for the NextDNS Kubernetes Operator. For a quick overview and getting started guide, see the [root README](../README.md).

---

## Documentation Index

| File | Description |
|------|-------------|
| [profile-configuration.md](profile-configuration.md) | ConfigMap export and observe mode for `NextDNSProfile` |
| [coredns.md](coredns.md) | CoreDNS deployment, upstream protocols, plugin configuration (cache, metrics, health, errors, rewrite, hosts, domain overrides) |
| [multus.md](multus.md) | Multus CNI integration: NAD setup, static IPs, status reporting |
| [gateway.md](gateway.md) | Gateway API exposure: setup, infrastructure field, proxy replicas |
| [reference.md](reference.md) | Complete CRD field reference for all 5 CRDs, status fields, and conditions |

---

## Breaking Change — v0.18.0

Plugin-level fields (`upstream`, `cache`, `metrics`, `logging`, `domainOverrides`) are now grouped under `spec.corefile`. Manifests using the old top-level form will be rejected by CRD validation.

See the [CoreDNS docs](coredns.md) for the full before/after migration example.

---

## Drift Detection

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
- Syncs include +/-10% jitter to prevent all resources from hitting the API simultaneously
- Each profile makes ~1 API call per sync period
- List resources (allowlist, denylist, tldlist) sync status but don't call the NextDNS API directly
- Setting to `0` disables periodic syncing (event-driven only)

---

## Troubleshooting

### Profile Not Syncing

**Symptoms:** Profile shows `Ready: False` with `Synced: False`.

**Check:**
```bash
kubectl describe nextdnsprofile my-profile
```

**Common causes:**
1. **Invalid API key**: Verify the Secret exists and contains a valid key.
   ```bash
   kubectl get secret nextdns-credentials -o jsonpath='{.data.api-key}' | base64 -d
   ```
2. **API rate limiting**: The operator may be hitting NextDNS API rate limits. Check operator logs for 429 errors.
   ```bash
   kubectl logs -n nextdns-operator-system deploy/nextdns-operator -f
   ```
3. **Invalid profile ID**: If using `profileID` to adopt an existing profile, verify the ID exists in your NextDNS account.

### CoreDNS Not Starting

**Symptoms:** `NextDNSCoreDNS` shows `Ready: false`.

**Check:**
```bash
kubectl describe nextdnscoredns home-dns
kubectl get pods -l app.kubernetes.io/managed-by=nextdns-operator
```

**Common causes:**
1. **Profile not ready**: The referenced profile must be in `Ready` state. Check the `ProfileResolved` condition.
   ```bash
   kubectl get nextdnsprofile my-profile -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'
   ```
2. **Image pull errors**: Verify the CoreDNS image is accessible from your cluster.
   ```bash
   kubectl describe pod -l app.kubernetes.io/managed-by=nextdns-operator | grep -A5 Events
   ```
3. **Resource constraints**: If pods are pending, check for insufficient CPU/memory on nodes.
   ```bash
   kubectl get events --field-selector reason=FailedScheduling
   ```

### List References Not Resolving

**Symptoms:** `ReferencesResolved` condition is `False`.

**Check:**
```bash
kubectl get nextdnsprofile my-profile -o jsonpath='{.status.conditions[?(@.type=="ReferencesResolved")]}'
```

**Common causes:**
1. **List not found**: Verify the referenced list resource exists.
   ```bash
   kubectl get nextdnsallowlist,nextdnsdenylist,nextdnstldlist
   ```
2. **Wrong namespace**: If the list is in a different namespace, specify it in the reference.
   ```yaml
   allowlistRefs:
     - name: shared-allowlist
       namespace: dns-config
   ```
3. **List not ready**: The referenced list itself must have at least one domain/TLD entry (enforced by `MinItems=1` validation).

### Reading Conditions

```bash
# Check profile status
kubectl get nextdnsprofile my-profile -o yaml

# Quick check with jsonpath
kubectl get nextdnsprofile my-profile -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}'

# Check CoreDNS status
kubectl get nextdnscoredns home-dns -o yaml
```

**Healthy profile:**
```yaml
conditions:
  - type: Ready
    status: "True"
  - type: Synced
    status: "True"
  - type: ReferencesResolved
    status: "True"
```

**Profile waiting for list references:**
```yaml
conditions:
  - type: Ready
    status: "False"
    reason: ReferencesNotResolved
  - type: ReferencesResolved
    status: "False"
    reason: AllowlistNotFound
    message: "Allowlist 'business-apps' not found in namespace 'default'"
```

**CoreDNS waiting for profile:**
```yaml
conditions:
  - type: Ready
    status: "False"
    reason: ProfileNotReady
  - type: ProfileResolved
    status: "False"
    reason: ProfileNotReady
    message: "Waiting for profile to become ready"
```

---

## Architecture

### Multi-CRD Architecture

```
                    ┌─────────────────────┐
                    │   NextDNSProfile    │
                    │                     │
                    │  - security         │
                    │  - privacy          │
                    │  - parentalControl  │
                    │  - settings         │
                    │  - inline lists     │
                    └────────┬────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
    ┌─────────▼──────┐ ┌────▼───────┐ ┌────▼───────┐
    │NextDNSAllowlist│ │NextDNS     │ │NextDNS     │
    │                │ │Denylist    │ │TLDList     │
    │ - domains[]    │ │            │ │            │
    │ - description  │ │ - domains[]│ │ - tlds[]   │
    └────────────────┘ └────────────┘ └────────────┘
              │              │              │
              └──────────────┼──────────────┘
                             │ shared across
                             │ multiple profiles
                             ▼
                    ┌─────────────────────┐
                    │   NextDNSCoreDNS   │
                    │                     │
                    │  - profileRef ──────┼──► NextDNSProfile
                    │  - upstream         │
                    │  - deployment       │
                    │  - service          │
                    └─────────────────────┘
```

List resources (`NextDNSAllowlist`, `NextDNSDenylist`, `NextDNSTLDList`) are **reusable** — a single list can be referenced by multiple profiles. The profile controller merges domains from all referenced lists with inline entries.

### Reconciliation Flow

**NextDNSProfile reconciliation:**

1. Read the profile spec and resolve credentials from the referenced Secret
2. Resolve all list references (`allowlistRefs`, `denylistRefs`, `tldListRefs`) and merge domains
3. If `profileID` is set, adopt the existing profile; otherwise, create a new one
4. Apply the merged configuration to the NextDNS API
5. If `configMapRef.enabled`, create/update the ConfigMap with connection details
6. Update status with profile ID, fingerprint, aggregated counts, and conditions

**NextDNSCoreDNS reconciliation:**

1. Resolve the referenced `NextDNSProfile` and verify it is Ready
2. Generate a CoreDNS Corefile based on upstream config, cache settings, domain overrides, and profile fingerprint
3. Create/update the ConfigMap containing the Corefile
4. Create/update the Deployment or DaemonSet
5. Create/update the Service
6. Update status with endpoints, replica counts, and conditions

### How List References Work

When a profile references a list resource:

1. The profile controller watches for changes to referenced list resources
2. When a list changes, all profiles referencing it are re-reconciled
3. Domains from all referenced lists are merged with inline `allowlist`/`denylist` entries
4. Deduplication ensures no domain appears twice in the final list sent to the API
5. The `referencedResources` status field tracks each list's name, namespace, readiness, and item count

### How ConfigMap Export Works

**Export** (`configMapRef`): After syncing a profile, the operator creates a ConfigMap containing the profile's DNS connection details (profile ID, DoT/DoH/DoQ endpoints, IPv4/IPv6 addresses). Other workloads can consume this ConfigMap via `envFrom` or volume mounts.
