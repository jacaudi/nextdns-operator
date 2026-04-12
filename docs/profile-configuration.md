# NextDNSProfile Configuration

The `NextDNSProfile` custom resource manages a NextDNS profile declaratively. Each CR maps to one profile in the NextDNS dashboard and supports the full range of security, privacy, parental control, and settings options.

> For the full documentation index, see the [main docs page](README.md).

---

## ConfigMap Export

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

---

## Observe Mode

Observe mode lets you safely adopt an existing NextDNS profile into GitOps management without modifying it. The operator reads the full remote profile configuration and stores it in `status.observedConfig`, but never writes any changes back to NextDNS.

The `observedConfig` captures the complete profile state: security, privacy (blocklists, natives), parental control (including blockBypass and recreation schedules), denylist, allowlist, rewrites, settings (logs with location and drop fields, block page, performance, web3), blocked TLDs, and DNS setup endpoints (IPv4, IPv6, LinkedIP, DNSCrypt).

This is useful when you have profiles already configured through the NextDNS dashboard and want to bring them under declarative management without risk of accidentally overwriting settings.

**Create a NextDNSProfile in observe mode:**

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSProfile
metadata:
  name: my-existing-profile
spec:
  mode: observe
  profileID: "abc123"
  credentialsRef:
    name: nextdns-credentials
```

The operator will read the remote profile and populate `status.observedConfig` with the full configuration. No changes are made to the remote profile.

**Inspect the observed configuration:**

```bash
kubectl get nextdnsprofile my-existing-profile -o jsonpath='{.status.observedConfig}' | jq .
```

This returns the complete remote profile configuration, including security settings, privacy blocklists, deny/allowlists, rewrites, parental controls, and settings.

**Use the suggested spec for easy transition:**

```bash
kubectl get nextdnsprofile my-existing-profile -o jsonpath='{.status.suggestedSpec}' | jq .
```

The `suggestedSpec` field contains a spec-compatible translation of the observed configuration. You can copy fields directly from `suggestedSpec` into your CR spec when transitioning to managed mode.

The `observedConfig` also includes a `setup` section with DNS endpoint information (IPv4/IPv6 addresses, LinkedIP configuration, DNSCrypt stamp). This data is read-only and not included in `suggestedSpec`.

> **Limitations:** Some fields cannot be derived from the NextDNS API and are omitted from `suggestedSpec`:
> - `settings.logs.logClientsIPs` and `settings.logs.logDomains` -- not exposed by the API
> - `blockedTLDs` are included for reference but must be placed in a `NextDNSTLDList` CR and referenced via `spec.tldListRefs`

### Transitioning to Managed Mode

Once you have inspected the observed configuration and are ready to manage the profile declaratively, follow these steps:

1. **Inspect `status.suggestedSpec`** to see the spec-compatible translation of the remote configuration.
2. **Copy the desired configuration sections** from `status.suggestedSpec` into the `spec` of your NextDNSProfile CR. The `suggestedSpec` provides values in the correct spec format. Use `status.observedConfig` as a reference for the raw API values.
3. **Add `spec.name`** with the profile name.
4. **Change `spec.mode` to `managed`** (or remove it entirely, since `managed` is the default).

**Example -- transitioning from observe to managed:**

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSProfile
metadata:
  name: my-existing-profile
spec:
  # mode: managed  # This is the default, so it can be omitted
  name: "My Profile"
  profileID: "abc123"
  credentialsRef:
    name: nextdns-credentials
  security:
    aiThreatDetection: true
    googleSafeBrowsing: true
  privacy:
    blocklists:
      - id: nextdns-recommended
    disguisedTrackers: true
  settings:
    logs:
      enabled: true
      retention: 30d
```

> **Transition guard:** The operator blocks switching to managed mode if `observedConfig` exists in status but the spec contains no configuration sections (security, privacy, denylist, allowlist, rewrites, parentalControl, or settings). This prevents accidentally overwriting a configured profile with empty settings. Populate at least one configuration section in the spec before switching to managed mode.
