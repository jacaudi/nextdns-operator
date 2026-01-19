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
| **NextDNS Client** | `github.com/jacaudi/nextdns-go` | Full API coverage, MIT licensed |
| **K8s Libraries** | controller-runtime | Standard for operators, handles reconciliation loops |

---

## CRD Architecture

### Multi-CRD Design

The operator uses a multi-CRD architecture to enable reusability and separation of concerns:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CRD Relationships                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────┐                                                       │
│   │ NextDNSProfile  │───references──►┌─────────────────┐                   │
│   │                 │                │ NextDNSAllowlist │ (0..N)            │
│   │  - name         │                └─────────────────┘                   │
│   │  - security     │                                                       │
│   │  - privacy      │───references──►┌─────────────────┐                   │
│   │  - parental     │                │ NextDNSDenylist  │ (0..N)            │
│   │  - settings     │                └─────────────────┘                   │
│   │  - rewrites     │                                                       │
│   │                 │───references──►┌─────────────────┐                   │
│   │  - allowlistRefs│                │ NextDNSTLDList   │ (0..N)            │
│   │  - denylistRefs │                └─────────────────┘                   │
│   │  - tldListRefs  │                                                       │
│   └─────────────────┘                                                       │
│                                                                             │
│   Note: List CRDs can be referenced by MULTIPLE profiles (shared policies) │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### CRD Overview

| CRD | Purpose | Scope | Reusable |
|-----|---------|-------|----------|
| **NextDNSProfile** | Core profile with settings, security, privacy | Namespaced | No (1:1 with NextDNS profile) |
| **NextDNSAllowlist** | Domains to always allow | Namespaced | Yes (many profiles) |
| **NextDNSDenylist** | Domains to always block | Namespaced | Yes (many profiles) |
| **NextDNSTLDList** | TLDs to block for security | Namespaced | Yes (many profiles) |

### Benefits of Multi-CRD

1. **Reusability**: Share a corporate denylist across all profiles
2. **Separation of Concerns**: Security team manages TLD blocks, app teams manage allowlists
3. **Scalability**: Large lists don't bloat the profile resource
4. **GitOps-friendly**: Smaller, focused resources for easier review
5. **RBAC Flexibility**: Different permissions per resource type

---

## Detailed CRD Specification

### 1. NextDNSAllowlist CRD

Defines a reusable list of domains that should always be allowed (bypass blocking).

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSAllowlist
metadata:
  name: corporate-allowlist
  namespace: default
spec:
  # Human-readable description
  description: "Corporate services that must always resolve"

  # List of domains to allow
  domains:
    - domain: "*.microsoft.com"
      active: true
    - domain: "*.office365.com"
      active: true
    - domain: "zoom.us"
      active: true
    - domain: "*.slack.com"
      active: true
    - domain: "internal.company.com"
      active: true
      # Optional: reason for allowlisting (for documentation)
      reason: "Internal corporate portal"

status:
  # Number of active domains in this list
  domainCount: 5

  # Profiles currently using this allowlist
  profileRefs:
    - name: corporate-dns
      namespace: default
    - name: developer-dns
      namespace: dev-team

  # Standard conditions
  conditions:
    - type: Ready
      status: "True"
      reason: Valid
      message: "Allowlist validated successfully"
```

---

### 2. NextDNSDenylist CRD

Defines a reusable list of domains that should always be blocked.

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSDenylist
metadata:
  name: security-denylist
  namespace: default
spec:
  # Human-readable description
  description: "Known malicious and unwanted domains"

  # List of domains to block
  domains:
    - domain: "malware.example.com"
      active: true
      reason: "Known malware distribution"
    - domain: "*.phishing-site.net"
      active: true
      reason: "Phishing campaign"
    - domain: "ads.trackernetwork.com"
      active: true
    - domain: "cryptominer.bad.com"
      active: true
      reason: "Cryptomining scripts"
    # Wildcards supported
    - domain: "*.malicious-cdn.com"
      active: true

status:
  # Number of active domains in this list
  domainCount: 5

  # Profiles currently using this denylist
  profileRefs:
    - name: corporate-dns
      namespace: default

  conditions:
    - type: Ready
      status: "True"
      reason: Valid
      message: "Denylist validated successfully"
```

---

### 3. NextDNSTLDList CRD

Defines a list of Top-Level Domains (TLDs) to block for security purposes.
NextDNS supports blocking entire TLDs that are commonly associated with abuse.

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSTLDList
metadata:
  name: high-risk-tlds
  namespace: default
spec:
  # Human-readable description
  description: "High-risk TLDs commonly used for malicious purposes"

  # List of TLDs to block
  tlds:
    # Free/cheap TLDs often abused
    - tld: "tk"
      active: true
      reason: "High abuse rate - free TLD"
    - tld: "ml"
      active: true
      reason: "High abuse rate - free TLD"
    - tld: "ga"
      active: true
      reason: "High abuse rate - free TLD"
    - tld: "cf"
      active: true
      reason: "High abuse rate - free TLD"
    - tld: "gq"
      active: true
      reason: "High abuse rate - free TLD"

    # Other high-risk TLDs
    - tld: "xyz"
      active: true
      reason: "Commonly used in phishing"
    - tld: "top"
      active: true
      reason: "High spam/malware rate"
    - tld: "pw"
      active: true
      reason: "Palau - high abuse"
    - tld: "cc"
      active: true
      reason: "Cocos Islands - high abuse"

    # Country-code TLDs with sanctions or restrictions
    - tld: "ru"
      active: false  # Disabled by default - user choice
      reason: "Russia - optional block"
    - tld: "cn"
      active: false  # Disabled by default - user choice
      reason: "China - optional block"

status:
  # Number of active TLDs in this list
  tldCount: 9

  # Profiles currently using this TLD list
  profileRefs:
    - name: corporate-dns
      namespace: default

  conditions:
    - type: Ready
      status: "True"
      reason: Valid
      message: "TLD list validated successfully"
```

---

### 4. NextDNSProfile CRD

The main profile resource that references the list CRDs and contains other settings.

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

  # ============================================================
  # REFERENCES TO LIST CRDs (new multi-CRD approach)
  # ============================================================

  # References to NextDNSAllowlist resources
  # All domains from referenced allowlists are merged
  allowlistRefs:
    - name: corporate-allowlist
    - name: developer-tools-allowlist
      # Optional: namespace if different from profile
      namespace: shared-policies

  # References to NextDNSDenylist resources
  # All domains from referenced denylists are merged
  denylistRefs:
    - name: security-denylist
    - name: ads-denylist
      namespace: shared-policies

  # References to NextDNSTLDList resources
  # All TLDs from referenced lists are merged
  tldListRefs:
    - name: high-risk-tlds
    - name: regional-blocks
      namespace: shared-policies

  # ============================================================
  # INLINE LISTS (for simple cases, no separate CRD needed)
  # ============================================================

  # Inline denylist (merged with denylistRefs)
  denylist:
    - domain: "one-off-block.example.com"
      active: true

  # Inline allowlist (merged with allowlistRefs)
  allowlist:
    - domain: "one-off-allow.example.com"
      active: true

  # ============================================================
  # OTHER SETTINGS
  # ============================================================

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

  # Aggregated counts from all sources
  aggregatedCounts:
    allowlistDomains: 15    # Total from refs + inline
    denylistDomains: 42     # Total from refs + inline
    blockedTLDs: 9          # Total from TLD refs

  # Status of referenced resources
  referencedResources:
    allowlists:
      - name: corporate-allowlist
        namespace: default
        ready: true
        domainCount: 5
      - name: developer-tools-allowlist
        namespace: shared-policies
        ready: true
        domainCount: 10
    denylists:
      - name: security-denylist
        namespace: default
        ready: true
        domainCount: 30
    tldLists:
      - name: high-risk-tlds
        namespace: default
        ready: true
        tldCount: 9

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
    - type: ReferencesResolved
      status: "True"
      lastTransitionTime: "2025-01-19T10:00:00Z"
      reason: AllResolved
      message: "All referenced allowlists, denylists, and TLD lists found"

  # Last successful sync time
  lastSyncTime: "2025-01-19T10:00:00Z"

  # Track spec changes
  observedGeneration: 1
```

---

## Go Types Definition

### List CRD Types

```go
// api/v1alpha1/nextdnsallowlist_types.go

package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NextDNSAllowlistSpec defines the desired state of NextDNSAllowlist
type NextDNSAllowlistSpec struct {
    // Description provides context for this allowlist
    // +optional
    Description string `json:"description,omitempty"`

    // Domains is the list of domains to allow
    // +kubebuilder:validation:MinItems=1
    Domains []AllowlistDomainEntry `json:"domains"`
}

// AllowlistDomainEntry represents a domain in the allowlist
type AllowlistDomainEntry struct {
    // Domain is the domain name (supports wildcards like *.example.com)
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    Domain string `json:"domain"`

    // Active indicates if this entry is enabled
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`

    // Reason documents why this domain is allowlisted
    // +optional
    Reason string `json:"reason,omitempty"`
}

// NextDNSAllowlistStatus defines the observed state of NextDNSAllowlist
type NextDNSAllowlistStatus struct {
    // DomainCount is the number of active domains
    DomainCount int `json:"domainCount,omitempty"`

    // ProfileRefs lists profiles using this allowlist
    // +optional
    ProfileRefs []ResourceReference `json:"profileRefs,omitempty"`

    // Conditions represent the latest available observations
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Domains",type=integer,JSONPath=`.status.domainCount`
// +kubebuilder:printcolumn:name="Profiles",type=integer,JSONPath=`.status.profileRefs`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NextDNSAllowlist is the Schema for the nextdnsallowlists API
type NextDNSAllowlist struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   NextDNSAllowlistSpec   `json:"spec,omitempty"`
    Status NextDNSAllowlistStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSAllowlistList contains a list of NextDNSAllowlist
type NextDNSAllowlistList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []NextDNSAllowlist `json:"items"`
}
```

```go
// api/v1alpha1/nextdnsdenylist_types.go

package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NextDNSDenylistSpec defines the desired state of NextDNSDenylist
type NextDNSDenylistSpec struct {
    // Description provides context for this denylist
    // +optional
    Description string `json:"description,omitempty"`

    // Domains is the list of domains to block
    // +kubebuilder:validation:MinItems=1
    Domains []DenylistDomainEntry `json:"domains"`
}

// DenylistDomainEntry represents a domain in the denylist
type DenylistDomainEntry struct {
    // Domain is the domain name (supports wildcards like *.example.com)
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    Domain string `json:"domain"`

    // Active indicates if this entry is enabled
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`

    // Reason documents why this domain is denylisted
    // +optional
    Reason string `json:"reason,omitempty"`
}

// NextDNSDenylistStatus defines the observed state of NextDNSDenylist
type NextDNSDenylistStatus struct {
    // DomainCount is the number of active domains
    DomainCount int `json:"domainCount,omitempty"`

    // ProfileRefs lists profiles using this denylist
    // +optional
    ProfileRefs []ResourceReference `json:"profileRefs,omitempty"`

    // Conditions represent the latest available observations
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Domains",type=integer,JSONPath=`.status.domainCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NextDNSDenylist is the Schema for the nextdnsdenylists API
type NextDNSDenylist struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   NextDNSDenylistSpec   `json:"spec,omitempty"`
    Status NextDNSDenylistStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSDenylistList contains a list of NextDNSDenylist
type NextDNSDenylistList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []NextDNSDenylist `json:"items"`
}
```

```go
// api/v1alpha1/nextdnstldlist_types.go

package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NextDNSTLDListSpec defines the desired state of NextDNSTLDList
type NextDNSTLDListSpec struct {
    // Description provides context for this TLD list
    // +optional
    Description string `json:"description,omitempty"`

    // TLDs is the list of top-level domains to block
    // +kubebuilder:validation:MinItems=1
    TLDs []TLDEntry `json:"tlds"`
}

// TLDEntry represents a TLD in the block list
type TLDEntry struct {
    // TLD is the top-level domain (without the dot, e.g., "tk", "xyz")
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    // +kubebuilder:validation:MaxLength=63
    // +kubebuilder:validation:Pattern=`^[a-zA-Z][a-zA-Z0-9]*$`
    TLD string `json:"tld"`

    // Active indicates if this TLD is blocked
    // +kubebuilder:default=true
    Active *bool `json:"active,omitempty"`

    // Reason documents why this TLD is blocked
    // +optional
    Reason string `json:"reason,omitempty"`
}

// NextDNSTLDListStatus defines the observed state of NextDNSTLDList
type NextDNSTLDListStatus struct {
    // TLDCount is the number of active TLDs
    TLDCount int `json:"tldCount,omitempty"`

    // ProfileRefs lists profiles using this TLD list
    // +optional
    ProfileRefs []ResourceReference `json:"profileRefs,omitempty"`

    // Conditions represent the latest available observations
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="TLDs",type=integer,JSONPath=`.status.tldCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NextDNSTLDList is the Schema for the nextdnstldlists API
type NextDNSTLDList struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   NextDNSTLDListSpec   `json:"spec,omitempty"`
    Status NextDNSTLDListStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NextDNSTLDListList contains a list of NextDNSTLDList
type NextDNSTLDListList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []NextDNSTLDList `json:"items"`
}
```

### Shared Types

```go
// api/v1alpha1/shared_types.go

package v1alpha1

// ResourceReference identifies a Kubernetes resource
type ResourceReference struct {
    // Name of the resource
    Name string `json:"name"`

    // Namespace of the resource (optional, defaults to same namespace)
    // +optional
    Namespace string `json:"namespace,omitempty"`
}

// ListReference references a list CRD (allowlist, denylist, or TLD list)
type ListReference struct {
    // Name of the list resource
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Namespace of the list resource (defaults to profile's namespace)
    // +optional
    Namespace string `json:"namespace,omitempty"`
}

// ReferencedResourceStatus tracks the status of a referenced resource
type ReferencedResourceStatus struct {
    // Name of the resource
    Name string `json:"name"`

    // Namespace of the resource
    Namespace string `json:"namespace"`

    // Ready indicates if the resource is ready
    Ready bool `json:"ready"`

    // Count of items (domains or TLDs)
    Count int `json:"count,omitempty"`
}
```

### Profile Types (Updated)

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

    // ===========================================
    // List References (Multi-CRD Architecture)
    // ===========================================

    // AllowlistRefs references NextDNSAllowlist resources
    // Domains from all referenced allowlists are merged
    // +optional
    AllowlistRefs []ListReference `json:"allowlistRefs,omitempty"`

    // DenylistRefs references NextDNSDenylist resources
    // Domains from all referenced denylists are merged
    // +optional
    DenylistRefs []ListReference `json:"denylistRefs,omitempty"`

    // TLDListRefs references NextDNSTLDList resources
    // TLDs from all referenced lists are merged
    // +optional
    TLDListRefs []ListReference `json:"tldListRefs,omitempty"`

    // ===========================================
    // Inline Lists (for simple cases)
    // ===========================================

    // Denylist specifies inline domains to block (merged with DenylistRefs)
    // +optional
    Denylist []DomainEntry `json:"denylist,omitempty"`

    // Allowlist specifies inline domains to allow (merged with AllowlistRefs)
    // +optional
    Allowlist []DomainEntry `json:"allowlist,omitempty"`

    // ===========================================
    // Other Settings
    // ===========================================

    // Security configures threat protection settings
    // +optional
    Security *SecuritySpec `json:"security,omitempty"`

    // Privacy configures tracker and ad blocking
    // +optional
    Privacy *PrivacySpec `json:"privacy,omitempty"`

    // ParentalControl configures content filtering
    // +optional
    ParentalControl *ParentalControlSpec `json:"parentalControl,omitempty"`

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
    &ProfileReconciler{},           // Creates/adopts profile
    &ListReferencesReconciler{},    // Resolves allowlist/denylist/TLD refs
    &SecurityReconciler{},          // Syncs security settings
    &PrivacyReconciler{},           // Syncs privacy settings
    &ParentalControlReconciler{},
    &DenylistReconciler{},          // Syncs merged denylist to NextDNS
    &AllowlistReconciler{},         // Syncs merged allowlist to NextDNS
    &TLDListReconciler{},           // Syncs merged TLD list to NextDNS
    &RewritesReconciler{},
    &SettingsReconciler{},
    &StatusReconciler{},            // Updates CR status
}
```

### Controller Watch Configuration

The profile controller must watch changes to referenced list resources:

```go
// internal/controller/nextdnsprofile_controller.go

func (r *NextDNSProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1alpha1.NextDNSProfile{}).
        // Watch Allowlists and trigger reconcile for referencing profiles
        Watches(
            &v1alpha1.NextDNSAllowlist{},
            handler.EnqueueRequestsFromMapFunc(r.findProfilesForAllowlist),
        ).
        // Watch Denylists and trigger reconcile for referencing profiles
        Watches(
            &v1alpha1.NextDNSDenylist{},
            handler.EnqueueRequestsFromMapFunc(r.findProfilesForDenylist),
        ).
        // Watch TLDLists and trigger reconcile for referencing profiles
        Watches(
            &v1alpha1.NextDNSTLDList{},
            handler.EnqueueRequestsFromMapFunc(r.findProfilesForTLDList),
        ).
        // Watch referenced Secrets for credential changes
        Watches(
            &corev1.Secret{},
            handler.EnqueueRequestsFromMapFunc(r.findProfilesForSecret),
        ).
        Complete(r)
}

// findProfilesForAllowlist returns all profiles that reference the given allowlist
func (r *NextDNSProfileReconciler) findProfilesForAllowlist(
    ctx context.Context,
    obj client.Object,
) []reconcile.Request {
    allowlist := obj.(*v1alpha1.NextDNSAllowlist)

    // List all profiles
    var profiles v1alpha1.NextDNSProfileList
    if err := r.List(ctx, &profiles); err != nil {
        return nil
    }

    var requests []reconcile.Request
    for _, profile := range profiles.Items {
        for _, ref := range profile.Spec.AllowlistRefs {
            refNs := ref.Namespace
            if refNs == "" {
                refNs = profile.Namespace
            }
            if ref.Name == allowlist.Name && refNs == allowlist.Namespace {
                requests = append(requests, reconcile.Request{
                    NamespacedName: types.NamespacedName{
                        Name:      profile.Name,
                        Namespace: profile.Namespace,
                    },
                })
                break
            }
        }
    }
    return requests
}
```

### List Resolution and Merging

```go
// internal/controller/subreconcilers/list_references.go

type ListReferencesReconciler struct {
    Client client.Client
}

func (r *ListReferencesReconciler) Reconcile(
    ctx context.Context,
    profile *v1alpha1.NextDNSProfile,
    _ *nextdns.Client,
) (*ResolvedLists, error) {
    resolved := &ResolvedLists{
        Allowlist: make([]DomainEntry, 0),
        Denylist:  make([]DomainEntry, 0),
        TLDs:      make([]TLDEntry, 0),
    }

    // Resolve allowlist references
    for _, ref := range profile.Spec.AllowlistRefs {
        ns := ref.Namespace
        if ns == "" {
            ns = profile.Namespace
        }

        var allowlist v1alpha1.NextDNSAllowlist
        if err := r.Client.Get(ctx, types.NamespacedName{
            Name:      ref.Name,
            Namespace: ns,
        }, &allowlist); err != nil {
            if apierrors.IsNotFound(err) {
                // Set condition: reference not found
                return nil, fmt.Errorf("allowlist %s/%s not found", ns, ref.Name)
            }
            return nil, err
        }

        // Append domains from this allowlist
        for _, entry := range allowlist.Spec.Domains {
            if entry.Active == nil || *entry.Active {
                resolved.Allowlist = append(resolved.Allowlist, DomainEntry{
                    Domain: entry.Domain,
                    Source: fmt.Sprintf("allowlist/%s/%s", ns, ref.Name),
                })
            }
        }
    }

    // Append inline allowlist entries
    for _, entry := range profile.Spec.Allowlist {
        if entry.Active == nil || *entry.Active {
            resolved.Allowlist = append(resolved.Allowlist, DomainEntry{
                Domain: entry.Domain,
                Source: "inline",
            })
        }
    }

    // Similar logic for denylist and TLD refs...

    return resolved, nil
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
│       ├── shared_types.go              # Shared types (ResourceReference, etc.)
│       ├── nextdnsprofile_types.go      # Profile CRD
│       ├── nextdnsallowlist_types.go    # Allowlist CRD
│       ├── nextdnsdenylist_types.go     # Denylist CRD
│       ├── nextdnstldlist_types.go      # TLD List CRD
│       ├── nextdnsprofile_webhook.go    # Validation webhook
│       └── zz_generated.deepcopy.go
├── cmd/
│   └── main.go
├── config/
│   ├── crd/
│   │   └── bases/
│   │       ├── nextdns.io_nextdnsprofiles.yaml
│   │       ├── nextdns.io_nextdnsallowlists.yaml
│   │       ├── nextdns.io_nextdnsdenylists.yaml
│   │       └── nextdns.io_nextdnstldlists.yaml
│   ├── default/
│   ├── manager/
│   ├── rbac/
│   └── samples/
│       ├── nextdns_v1alpha1_nextdnsprofile.yaml
│       ├── nextdns_v1alpha1_nextdnsallowlist.yaml
│       ├── nextdns_v1alpha1_nextdnsdenylist.yaml
│       └── nextdns_v1alpha1_nextdnstldlist.yaml
├── internal/
│   ├── controller/
│   │   ├── nextdnsprofile_controller.go
│   │   ├── nextdnsprofile_controller_test.go
│   │   ├── nextdnsallowlist_controller.go    # Watches for changes, triggers profile reconcile
│   │   ├── nextdnsdenylist_controller.go     # Watches for changes, triggers profile reconcile
│   │   ├── nextdnstldlist_controller.go      # Watches for changes, triggers profile reconcile
│   │   └── subreconcilers/
│   │       ├── finalizer.go
│   │       ├── profile.go
│   │       ├── security.go
│   │       ├── privacy.go
│   │       ├── lists.go                      # Resolves and merges list references
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
- [ ] Define all CRD types:
  - [ ] NextDNSProfile
  - [ ] NextDNSAllowlist
  - [ ] NextDNSDenylist
  - [ ] NextDNSTLDList
- [ ] Implement NextDNSProfile controller with create/update/delete
- [ ] Add finalizers for cleanup
- [ ] Basic credential management via Secrets
- [ ] Unit tests for controller logic

### Phase 2: List CRDs and References
- [ ] Implement list CRD controllers (Allowlist, Denylist, TLDList)
- [ ] Implement reference resolution in profile controller
- [ ] Add watches: profile reconciles when referenced lists change
- [ ] Merge logic for inline + referenced lists
- [ ] Update list status with referencing profiles
- [ ] Integration tests with list references

### Phase 3: Full Feature Parity
- [ ] Implement all NextDNS settings (security, privacy, parental, etc.)
- [ ] Add validation webhooks for all CRDs
- [ ] Comprehensive status conditions
- [ ] Cross-namespace reference support
- [ ] Integration tests with NextDNS API (mocked)

### Phase 4: Production Readiness
- [ ] Metrics and observability (Prometheus)
- [ ] Helm chart for installation
- [ ] Documentation and examples
- [ ] E2E tests
- [ ] CI/CD pipeline

### Phase 5: Advanced Features (Future)
- [ ] Cross-cluster profile sync
- [ ] Drift detection and reconciliation
- [ ] Webhook for external DNS configuration
- [ ] ClusterNextDNSAllowlist (cluster-scoped lists)

---

## Sample Usage

### 1. Create Reusable Lists

First, create the list resources that can be shared across profiles:

```yaml
# security-denylist.yaml - Shared security blocklist
apiVersion: nextdns.io/v1alpha1
kind: NextDNSDenylist
metadata:
  name: security-denylist
  namespace: shared-policies
spec:
  description: "Corporate security blocklist"
  domains:
    - domain: "malware.example.com"
      reason: "Known malware"
    - domain: "*.phishing-domain.net"
      reason: "Phishing campaign"
    - domain: "cryptominer.bad.com"
      reason: "Cryptojacking"
---
# corporate-allowlist.yaml - Essential business services
apiVersion: nextdns.io/v1alpha1
kind: NextDNSAllowlist
metadata:
  name: corporate-allowlist
  namespace: shared-policies
spec:
  description: "Essential business services"
  domains:
    - domain: "*.microsoft.com"
    - domain: "*.office365.com"
    - domain: "*.salesforce.com"
    - domain: "zoom.us"
---
# high-risk-tlds.yaml - Block risky TLDs
apiVersion: nextdns.io/v1alpha1
kind: NextDNSTLDList
metadata:
  name: high-risk-tlds
  namespace: shared-policies
spec:
  description: "TLDs with high abuse rates"
  tlds:
    - tld: "tk"
      reason: "Free TLD - high abuse"
    - tld: "ml"
      reason: "Free TLD - high abuse"
    - tld: "ga"
      reason: "Free TLD - high abuse"
    - tld: "xyz"
      reason: "Common in phishing"
```

### 2. Create a Profile Referencing the Lists

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

  # Reference shared lists
  allowlistRefs:
    - name: corporate-allowlist
      namespace: shared-policies

  denylistRefs:
    - name: security-denylist
      namespace: shared-policies

  tldListRefs:
    - name: high-risk-tlds
      namespace: shared-policies

  # Inline entries (profile-specific)
  allowlist:
    - domain: "internal.mycompany.com"
      reason: "Internal portal"

  # Standard settings
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

  settings:
    logs:
      enabled: true
      retention: "30d"
```

### 3. Check Status

```bash
# View all resources
$ kubectl get nextdnsprofiles,nextdnsallowlists,nextdnsdenylists,nextdnstldlists -A
NAMESPACE         NAME                                    PROFILE ID   READY   AGE
default           nextdnsprofile.nextdns.io/corporate-dns abc123       True    5m

NAMESPACE         NAME                                         DOMAINS   AGE
shared-policies   nextdnsallowlist.nextdns.io/corporate-allowlist   4    10m

NAMESPACE         NAME                                        DOMAINS   AGE
shared-policies   nextdnsdenylist.nextdns.io/security-denylist      3    10m

NAMESPACE         NAME                                       TLDS   AGE
shared-policies   nextdnstldlist.nextdns.io/high-risk-tlds      4    10m

# Detailed profile status
$ kubectl describe nextdnsprofile corporate-dns
...
Status:
  Profile ID:   abc123
  Fingerprint:  abc123.dns.nextdns.io
  Aggregated Counts:
    Allowlist Domains:  5   # 4 from ref + 1 inline
    Denylist Domains:   3
    Blocked TLDs:       4
  Referenced Resources:
    Allowlists:
      - Name:       corporate-allowlist
        Namespace:  shared-policies
        Ready:      true
        Count:      4
    Denylists:
      - Name:       security-denylist
        Namespace:  shared-policies
        Ready:      true
        Count:      3
    TLD Lists:
      - Name:       high-risk-tlds
        Namespace:  shared-policies
        Ready:      true
        Count:      4
  Conditions:
    Type:    Ready
    Status:  True
    Reason:  Synced
    Message: Profile successfully synced with NextDNS
    Type:    ReferencesResolved
    Status:  True
    Reason:  AllResolved
    Message: All referenced lists found and valid

# Check which profiles use a list
$ kubectl describe nextdnsdenylist security-denylist -n shared-policies
...
Status:
  Domain Count: 3
  Profile Refs:
    - Name:      corporate-dns
      Namespace: default
    - Name:      developer-dns
      Namespace: dev-team
```

### 4. Update a List (Triggers Profile Reconciliation)

```bash
# Add a domain to the denylist
$ kubectl patch nextdnsdenylist security-denylist -n shared-policies \
  --type='json' \
  -p='[{"op": "add", "path": "/spec/domains/-", "value": {"domain": "new-threat.com", "active": true}}]'

# The profile controller automatically reconciles
$ kubectl get nextdnsprofile corporate-dns -w
NAME            PROFILE ID   READY   AGE
corporate-dns   abc123       True    10m
corporate-dns   abc123       False   10m  # Reconciling
corporate-dns   abc123       True    10m  # Updated
```

---

## Dependencies

```go
// go.mod (key dependencies)
module github.com/yourusername/nextdns-operator

go 1.22

require (
    github.com/jacaudi/nextdns-go v0.5.0
    k8s.io/apimachinery v0.30.0
    k8s.io/client-go v0.30.0
    sigs.k8s.io/controller-runtime v0.18.0
)
```

---

## References

- [NextDNS API Documentation](https://nextdns.github.io/api/)
- [nextdns-go Client Library](https://github.com/jacaudi/nextdns-go)
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
