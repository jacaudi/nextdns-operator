# NextDNS Kubernetes Operator - Implementation Plan

## Overview

This document outlines the architecture and implementation plan for a Kubernetes operator that manages NextDNS profiles through Custom Resource Definitions (CRDs).

## Goals

1. Declaratively manage NextDNS profiles via Kubernetes resources
2. Synchronize Kubernetes desired state with NextDNS API
3. Support full lifecycle management (create, update, delete)
4. Provide status feedback on NextDNS resource state

---

## Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| **Language** | Go | Kubernetes-native, strong typing, excellent k8s libraries |
| **Framework** | Kubebuilder | Official k8s-sigs project, generates scaffolding, well-documented |
| **NextDNS Client** | `github.com/amalucelli/nextdns-go` | Full API coverage, MIT licensed, actively maintained |
| **K8s Libraries** | controller-runtime | Standard for operators, handles reconciliation loops |

---

## CRD Architecture

### Option A: Single CRD (Recommended for v1)

A single `NextDNSProfile` CRD that encapsulates all profile configuration:

```
NextDNSProfile (nextdns.io/v1alpha1)
├── spec
│   ├── name: string
│   ├── security: SecuritySpec
│   ├── privacy: PrivacySpec
│   ├── parentalControl: ParentalControlSpec
│   ├── denylist: []DomainEntry
│   ├── allowlist: []DomainEntry
│   ├── rewrites: []RewriteEntry
│   ├── settings: SettingsSpec
│   └── credentialsRef: SecretReference
└── status
    ├── profileID: string
    ├── conditions: []Condition
    ├── lastSyncTime: Time
    └── observedGeneration: int64
```

**Pros:**
- Simple mental model (1 CR = 1 NextDNS profile)
- Atomic operations
- Easier to manage permissions

**Cons:**
- Large spec for complex profiles
- Can't share components across profiles

### Option B: Multi-CRD Architecture (Future)

For advanced use cases, split into multiple CRDs:

```
NextDNSProfile          - Core profile identity
NextDNSSecurityPolicy   - Security settings (reusable)
NextDNSPrivacyPolicy    - Privacy settings (reusable)
NextDNSDenylist         - Denylist (reusable across profiles)
NextDNSAllowlist        - Allowlist (reusable across profiles)
```

**Recommendation:** Start with Option A (single CRD) for v1, consider multi-CRD for v2 based on user feedback.

---

## Detailed CRD Specification

### NextDNSProfile CRD

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSProfile
metadata:
  name: my-profile
  namespace: default
spec:
  # Human-readable name in NextDNS dashboard
  name: "Production DNS Profile"

  # Reference to Secret containing API key
  credentialsRef:
    name: nextdns-credentials
    key: api-key

  # Optional: Manage existing profile instead of creating new
  # If set, operator adopts this profile instead of creating
  profileID: "abc123"

  # Security settings
  security:
    aiThreatDetection: true
    threatIntelligenceFeeds:
      - cins
      - emerging-threats
    googleSafeBrowsing: true
    cryptojacking: true
    dnsRebinding: true
    idnHomographs: true
    typosquatting: true
    dga: true
    nrd: true
    ddns: false
    parking: true
    csam: true

  # Privacy settings
  privacy:
    blocklists:
      - id: nextdns-recommended
        active: true
      - id: oisd
        active: true
    natives:
      - id: apple
        active: true
      - id: windows
        active: true
    disguisedTrackers: true
    allowAffiliate: false

  # Parental control (optional)
  parentalControl:
    categories:
      - id: gambling
        active: true
      - id: adult
        active: true
    services:
      - id: tiktok
        active: false
      - id: youtube
        active: false
    safeSearch: true
    youtubeRestrictedMode: false

  # Custom denylist
  denylist:
    - domain: "ads.example.com"
      active: true
    - domain: "tracker.example.com"
      active: true

  # Custom allowlist
  allowlist:
    - domain: "trusted.example.com"
      active: true

  # DNS rewrites
  rewrites:
    - from: "internal.example.com"
      to: "192.168.1.100"
      active: true

  # General settings
  settings:
    logs:
      enabled: true
      privacy:
        logClientsIPs: false
        logDomains: true
      retention: 7d  # 1h, 6h, 1d, 7d, 30d, 90d, 1y, 2y
    blockPage:
      enabled: true
    performance:
      ecs: true
      cacheBoost: true
      cnameFlattening: true
    web3: false

status:
  # NextDNS-assigned profile ID
  profileID: "abc123"

  # Fingerprint endpoint for DNS configuration
  fingerprint: "abc123.dns.nextdns.io"

  # Standard Kubernetes conditions
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2025-01-19T10:00:00Z"
      reason: Synced
      message: "Profile successfully synced with NextDNS"
    - type: Synced
      status: "True"
      lastTransitionTime: "2025-01-19T10:00:00Z"
      reason: Success
      message: "All settings applied"

  # Last successful sync time
  lastSyncTime: "2025-01-19T10:00:00Z"

  # Track spec changes
  observedGeneration: 1
```

---

## Go Types Definition

```go
// api/v1alpha1/nextdnsprofile_types.go

package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NextDNSProfileSpec defines the desired state of NextDNSProfile
type NextDNSProfileSpec struct {
    // Name is the human-readable name shown in NextDNS dashboard
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    // +kubebuilder:validation:MaxLength=100
    Name string `json:"name"`

    // CredentialsRef references a Secret containing the NextDNS API key
    // +kubebuilder:validation:Required
    CredentialsRef SecretKeySelector `json:"credentialsRef"`

    // ProfileID optionally specifies an existing NextDNS profile to manage
    // If not set, a new profile will be created
    // +optional
    ProfileID string `json:"profileID,omitempty"`

    // Security configures threat protection settings
    // +optional
    Security *SecuritySpec `json:"security,omitempty"`

    // Privacy configures tracker and ad blocking
    // +optional
    Privacy *PrivacySpec `json:"privacy,omitempty"`

    // ParentalControl configures content filtering
    // +optional
    ParentalControl *ParentalControlSpec `json:"parentalControl,omitempty"`

    // Denylist specifies domains to block
    // +optional
    Denylist []DomainEntry `json:"denylist,omitempty"`

    // Allowlist specifies domains to allow (bypass blocking)
    // +optional
    Allowlist []DomainEntry `json:"allowlist,omitempty"`

    // Rewrites specifies DNS rewrites
    // +optional
    Rewrites []RewriteEntry `json:"rewrites,omitempty"`

    // Settings configures logging, performance, and other options
    // +optional
    Settings *SettingsSpec `json:"settings,omitempty"`
}

// SecretKeySelector references a key in a Secret
type SecretKeySelector struct {
    // Name is the name of the Secret
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Key is the key within the Secret
    // +kubebuilder:default=api-key
    Key string `json:"key,omitempty"`
}

// SecuritySpec defines security/threat protection settings
type SecuritySpec struct {
    // AIThreatDetection enables AI-based threat detection
    // +kubebuilder:default=true
    AIThreatDetection *bool `json:"aiThreatDetection,omitempty"`

    // ThreatIntelligenceFeeds specifies which threat feeds to use
    // +optional
    ThreatIntelligenceFeeds []string `json:"threatIntelligenceFeeds,omitempty"`

    // GoogleSafeBrowsing enables Google Safe Browsing protection
    // +kubebuilder:default=true
    GoogleSafeBrowsing *bool `json:"googleSafeBrowsing,omitempty"`

    // Cryptojacking blocks cryptomining scripts
    // +kubebuilder:default=true
    Cryptojacking *bool `json:"cryptojacking,omitempty"`

    // DNSRebinding protects against DNS rebinding attacks
    // +kubebuilder:default=true
    DNSRebinding *bool `json:"dnsRebinding,omitempty"`

    // IDNHomographs blocks IDN homograph attacks
    // +kubebuilder:default=true
    IDNHomographs *bool `json:"idnHomographs,omitempty"`

    // Typosquatting blocks typosquatting domains
    // +kubebuilder:default=true
    Typosquatting *bool `json:"typosquatting,omitempty"`

    // DGA blocks algorithmically-generated domains
    // +kubebuilder:default=true
    DGA *bool `json:"dga,omitempty"`

    // NRD blocks newly registered domains
    // +kubebuilder:default=false
    NRD *bool `json:"nrd,omitempty"`

    // DDNS blocks dynamic DNS hostnames
    // +kubebuilder:default=false
    DDNS *bool `json:"ddns,omitempty"`

    // Parking blocks parked domains
    // +kubebuilder:default=true
    Parking *bool `json:"parking,omitempty"`

    // CSAM blocks child sexual abuse material
    // +kubebuilder:default=true
    CSAM *bool `json:"csam,omitempty"`
}

// PrivacySpec defines privacy and ad-blocking settings
type PrivacySpec struct {
    // Blocklists specifies which ad/tracker blocklists to enable
    // +optional
    Blocklists []BlocklistEntry `json:"blocklists,omitempty"`

    // Natives specifies native tracking protection (per-vendor)
    // +optional
    Natives []NativeEntry `json:"natives,omitempty"`

    // DisguisedTrackers blocks trackers using CNAME cloaking
    // +kubebuilder:default=true
    DisguisedTrackers *bool `json:"disguisedTrackers,omitempty"`

    // AllowAffiliate allows affiliate & tracking links
    // +kubebuilder:default=false
    AllowAffiliate *bool `json:"allowAffiliate,omitempty"`
}

// BlocklistEntry references a privacy blocklist
type BlocklistEntry struct {
    // ID is the blocklist identifier (e.g., "nextdns-recommended", "oisd")
    ID string `json:"id"`
    // Active indicates if this blocklist is enabled
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`
}

// NativeEntry configures native tracker blocking for a vendor
type NativeEntry struct {
    // ID is the vendor identifier (e.g., "apple", "windows", "samsung")
    ID string `json:"id"`
    // Active indicates if blocking is enabled for this vendor
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`
}

// ParentalControlSpec defines parental control settings
type ParentalControlSpec struct {
    // Categories specifies content categories to block
    // +optional
    Categories []CategoryEntry `json:"categories,omitempty"`

    // Services specifies specific services to block
    // +optional
    Services []ServiceEntry `json:"services,omitempty"`

    // SafeSearch enforces safe search on search engines
    // +kubebuilder:default=false
    SafeSearch *bool `json:"safeSearch,omitempty"`

    // YouTubeRestrictedMode enforces YouTube restricted mode
    // +kubebuilder:default=false
    YouTubeRestrictedMode *bool `json:"youtubeRestrictedMode,omitempty"`
}

// CategoryEntry references a content category
type CategoryEntry struct {
    // ID is the category identifier (e.g., "gambling", "adult", "violence")
    ID string `json:"id"`
    // Active indicates if this category is blocked
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`
}

// ServiceEntry references a specific service
type ServiceEntry struct {
    // ID is the service identifier (e.g., "tiktok", "youtube", "facebook")
    ID string `json:"id"`
    // Active indicates if this service is blocked
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`
}

// DomainEntry represents a domain in allow/deny lists
type DomainEntry struct {
    // Domain is the domain name
    // +kubebuilder:validation:Required
    Domain string `json:"domain"`
    // Active indicates if this entry is enabled
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`
}

// RewriteEntry defines a DNS rewrite rule
type RewriteEntry struct {
    // From is the source domain
    // +kubebuilder:validation:Required
    From string `json:"from"`
    // To is the target (IP or domain)
    // +kubebuilder:validation:Required
    To string `json:"to"`
    // Active indicates if this rewrite is enabled
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`
}

// SettingsSpec defines general profile settings
type SettingsSpec struct {
    // Logs configures query logging
    // +optional
    Logs *LogsSpec `json:"logs,omitempty"`

    // BlockPage configures the block page
    // +optional
    BlockPage *BlockPageSpec `json:"blockPage,omitempty"`

    // Performance configures performance optimizations
    // +optional
    Performance *PerformanceSpec `json:"performance,omitempty"`

    // Web3 enables Web3 domain resolution
    // +kubebuilder:default=false
    Web3 *bool `json:"web3,omitempty"`
}

// LogsSpec configures logging settings
type LogsSpec struct {
    // Enabled turns logging on/off
    // +kubebuilder:default=true
    Enabled *bool `json:"enabled,omitempty"`

    // LogClientsIPs logs client IP addresses
    // +kubebuilder:default=false
    LogClientsIPs *bool `json:"logClientsIPs,omitempty"`

    // LogDomains logs queried domains
    // +kubebuilder:default=true
    LogDomains *bool `json:"logDomains,omitempty"`

    // Retention specifies log retention period
    // +kubebuilder:validation:Enum=1h;6h;1d;7d;30d;90d;1y;2y
    // +kubebuilder:default="7d"
    Retention string `json:"retention,omitempty"`
}

// BlockPageSpec configures the block page
type BlockPageSpec struct {
    // Enabled shows a block page instead of failing silently
    // +kubebuilder:default=true
    Enabled *bool `json:"enabled,omitempty"`
}

// PerformanceSpec configures performance settings
type PerformanceSpec struct {
    // ECS enables EDNS Client Subnet
    // +kubebuilder:default=true
    ECS *bool `json:"ecs,omitempty"`

    // CacheBoost enables extended caching
    // +kubebuilder:default=true
    CacheBoost *bool `json:"cacheBoost,omitempty"`

    // CNAMEFlattening enables CNAME flattening
    // +kubebuilder:default=true
    CNAMEFlattening *bool `json:"cnameFlattening,omitempty"`
}

// NextDNSProfileStatus defines the observed state of NextDNSProfile
type NextDNSProfileStatus struct {
    // ProfileID is the NextDNS-assigned profile identifier
    ProfileID string `json:"profileID,omitempty"`

    // Fingerprint is the DNS endpoint (e.g., "abc123.dns.nextdns.io")
    Fingerprint string `json:"fingerprint,omitempty"`

    // Conditions represent the latest available observations
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // LastSyncTime is the last time the profile was synced with NextDNS
    LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

    // ObservedGeneration is the generation last processed by the controller
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Profile ID",type=string,JSONPath=`.status.profileID`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NextDNSProfile is the Schema for the nextdnsprofiles API
type NextDNSProfile struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   NextDNSProfileSpec   `json:"spec,omitempty"`
    Status NextDNSProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSProfileList contains a list of NextDNSProfile
type NextDNSProfileList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []NextDNSProfile `json:"items"`
}

func init() {
    SchemeBuilder.Register(&NextDNSProfile{}, &NextDNSProfileList{})
}
```

---

## Controller Reconciliation Logic

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     Reconcile Request                           │
│                  (Namespace/Name of CR)                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. Fetch NextDNSProfile CR                                     │
│     - If not found (deleted) → return success                   │
│     - If found → continue                                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. Check for Deletion (DeletionTimestamp set?)                 │
│     YES → Handle Finalizer:                                     │
│           - Delete profile from NextDNS API                     │
│           - Remove finalizer                                    │
│           - Return success                                      │
│     NO → Continue                                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. Ensure Finalizer Present                                    │
│     - Add "nextdns.io/finalizer" if missing                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. Fetch API Credentials                                       │
│     - Read Secret from credentialsRef                           │
│     - If missing → set condition, requeue                       │
│     - Create NextDNS client                                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. Determine Create vs Update                                  │
│     - If status.profileID empty AND spec.profileID empty:       │
│       → CREATE new profile in NextDNS                           │
│     - If status.profileID set OR spec.profileID set:            │
│       → UPDATE existing profile                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  6. Sync Configuration to NextDNS                               │
│     For each sub-resource:                                      │
│     - Security settings                                         │
│     - Privacy settings                                          │
│     - Parental control                                          │
│     - Denylist                                                  │
│     - Allowlist                                                 │
│     - Rewrites                                                  │
│     - General settings                                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  7. Update Status                                               │
│     - Set profileID                                             │
│     - Set fingerprint                                           │
│     - Update conditions (Ready, Synced)                         │
│     - Set lastSyncTime                                          │
│     - Set observedGeneration                                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  8. Return Result                                               │
│     - Success → return without requeue                          │
│     - Transient error → return with requeue (exponential back)  │
│     - Permanent error → set condition, don't requeue            │
└─────────────────────────────────────────────────────────────────┘
```

### Subreconciler Pattern

Break the main reconciliation into focused subreconcilers:

```go
// internal/controller/subreconcilers.go

type SubReconciler interface {
    Reconcile(ctx context.Context, profile *v1alpha1.NextDNSProfile, client *nextdns.Client) error
}

// Subreconcilers (executed in order)
var subreconcilers = []SubReconciler{
    &FinalizerReconciler{},
    &ProfileReconciler{},      // Creates/adopts profile
    &SecurityReconciler{},     // Syncs security settings
    &PrivacyReconciler{},      // Syncs privacy settings
    &ParentalControlReconciler{},
    &DenylistReconciler{},
    &AllowlistReconciler{},
    &RewritesReconciler{},
    &SettingsReconciler{},
    &StatusReconciler{},       // Updates CR status
}
```

---

## Credential Management

### Secret Structure

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: nextdns-credentials
  namespace: default
type: Opaque
stringData:
  api-key: "your-nextdns-api-key-here"
```

### Security Considerations

1. **RBAC**: Controller needs `get` permission on Secrets in watched namespaces
2. **Namespacing**: Credentials can be namespace-scoped or cluster-scoped
3. **Rotation**: Support credential rotation without downtime
4. **Audit**: Log credential access (not values)

### Cluster-Scoped Credentials Option

For multi-tenant clusters, support a cluster-level credential:

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCredentials  # Optional future CRD
metadata:
  name: default
spec:
  secretRef:
    name: nextdns-api-key
    namespace: nextdns-system
```

---

## Error Handling Strategy

| Error Type | Action | Requeue |
|------------|--------|---------|
| CR not found | Return success (already deleted) | No |
| Secret not found | Set condition, warn | Yes (30s) |
| Invalid API key | Set condition, error | Yes (5m) |
| NextDNS API rate limit | Set condition, backoff | Yes (exponential) |
| NextDNS API unavailable | Set condition, retry | Yes (30s) |
| Profile not found (deleted externally) | Recreate or error | Configurable |
| Validation error | Set condition, surface | No |

---

## Project Structure

```
nextdns-operator/
├── api/
│   └── v1alpha1/
│       ├── groupversion_info.go
│       ├── nextdnsprofile_types.go
│       ├── nextdnsprofile_webhook.go  # Validation webhook
│       └── zz_generated.deepcopy.go
├── cmd/
│   └── main.go
├── config/
│   ├── crd/
│   │   └── bases/
│   │       └── nextdns.io_nextdnsprofiles.yaml
│   ├── default/
│   ├── manager/
│   ├── rbac/
│   └── samples/
│       └── nextdns_v1alpha1_nextdnsprofile.yaml
├── internal/
│   ├── controller/
│   │   ├── nextdnsprofile_controller.go
│   │   ├── nextdnsprofile_controller_test.go
│   │   └── subreconcilers/
│   │       ├── finalizer.go
│   │       ├── profile.go
│   │       ├── security.go
│   │       ├── privacy.go
│   │       └── ...
│   └── nextdns/
│       ├── client.go          # Wrapper around nextdns-go
│       └── client_test.go
├── test/
│   └── e2e/
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## Implementation Phases

### Phase 1: Foundation (MVP)
- [ ] Initialize Kubebuilder project
- [ ] Define NextDNSProfile CRD types
- [ ] Implement basic controller with create/update/delete
- [ ] Add finalizer for cleanup
- [ ] Basic credential management via Secrets
- [ ] Unit tests for controller logic

### Phase 2: Full Feature Parity
- [ ] Implement all NextDNS settings (security, privacy, parental, etc.)
- [ ] Add validation webhook
- [ ] Comprehensive status conditions
- [ ] Integration tests with NextDNS API (mocked)

### Phase 3: Production Readiness
- [ ] Metrics and observability (Prometheus)
- [ ] Helm chart for installation
- [ ] Documentation and examples
- [ ] E2E tests
- [ ] CI/CD pipeline

### Phase 4: Advanced Features (Future)
- [ ] Multi-CRD architecture (shared policies)
- [ ] Cross-cluster profile sync
- [ ] Drift detection and reconciliation
- [ ] Webhook for external DNS configuration

---

## Sample Usage

### Create a Profile

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSProfile
metadata:
  name: corporate-dns
  namespace: default
spec:
  name: "Corporate DNS Policy"
  credentialsRef:
    name: nextdns-credentials
  security:
    aiThreatDetection: true
    googleSafeBrowsing: true
    cryptojacking: true
    dnsRebinding: true
  privacy:
    blocklists:
      - id: nextdns-recommended
      - id: oisd
    disguisedTrackers: true
  denylist:
    - domain: "malware.example.com"
    - domain: "phishing.example.com"
  settings:
    logs:
      enabled: true
      retention: "30d"
```

### Check Status

```bash
$ kubectl get nextdnsprofiles
NAME            PROFILE ID   READY   AGE
corporate-dns   abc123       True    5m

$ kubectl describe nextdnsprofile corporate-dns
...
Status:
  Profile ID:   abc123
  Fingerprint:  abc123.dns.nextdns.io
  Conditions:
    Type:    Ready
    Status:  True
    Reason:  Synced
    Message: Profile successfully synced with NextDNS
```

---

## Dependencies

```go
// go.mod (key dependencies)
module github.com/yourusername/nextdns-operator

go 1.22

require (
    github.com/amalucelli/nextdns-go v0.5.0
    k8s.io/apimachinery v0.30.0
    k8s.io/client-go v0.30.0
    sigs.k8s.io/controller-runtime v0.18.0
)
```

---

## References

- [NextDNS API Documentation](https://nextdns.github.io/api/)
- [nextdns-go Client Library](https://github.com/amalucelli/nextdns-go)
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
