# Implementation Plan: Import Profile Configuration from ConfigMap JSON

**Issue:** [#43](https://github.com/jacaudi/nextdns-operator/issues/43) - Feature: Import profile configuration from ConfigMap JSON
**Author:** @jacaudi
**Status:** Planning

## Overview

This feature will allow users to import NextDNS profile configurations from a ConfigMap containing JSON, enabling migration of existing profiles to operator management.

---

## Phase 1: API Type Changes

**File: `api/v1alpha1/nextdnsprofile_types.go`**

### 1. Add new `ConfigImportRef` type:

```go
// ConfigImportRef references a ConfigMap containing profile configuration JSON
type ConfigImportRef struct {
    // Name is the name of the ConfigMap containing the configuration
    // +kubebuilder:validation:Required
    Name string `json:"name"`

    // Key is the key in the ConfigMap containing the JSON
    // +kubebuilder:default="config.json"
    // +optional
    Key string `json:"key,omitempty"`

    // Namespace is the namespace of the ConfigMap
    // Defaults to the namespace of the NextDNSProfile
    // +optional
    Namespace string `json:"namespace,omitempty"`
}
```

### 2. Add field to `NextDNSProfileSpec`:

```go
// ConfigImportRef references a ConfigMap containing profile configuration to import
// Fields specified directly in the spec take precedence over imported values
// +optional
ConfigImportRef *ConfigImportRef `json:"configImportRef,omitempty"`
```

### 3. Add status field for import tracking:

```go
// ConfigImportStatus tracks the last imported configuration
type ConfigImportStatus struct {
    // ConfigMapResourceVersion is the resource version of the imported ConfigMap
    ConfigMapResourceVersion string `json:"configMapResourceVersion,omitempty"`
    // LastImportTime is when the configuration was last imported
    LastImportTime *metav1.Time `json:"lastImportTime,omitempty"`
}
```

---

## Phase 2: Define Import JSON Schema

**File: `internal/configimport/types.go` (new file)**

```go
package configimport

// ProfileConfigJSON defines the JSON structure for importing profile configuration
type ProfileConfigJSON struct {
    Security        *SecurityConfigJSON        `json:"security,omitempty"`
    Privacy         *PrivacyConfigJSON         `json:"privacy,omitempty"`
    ParentalControl *ParentalControlConfigJSON `json:"parentalControl,omitempty"`
    Settings        *SettingsConfigJSON        `json:"settings,omitempty"`
    Denylist        []DomainEntryJSON          `json:"denylist,omitempty"`
    Allowlist       []DomainEntryJSON          `json:"allowlist,omitempty"`
    Rewrites        []RewriteEntryJSON         `json:"rewrites,omitempty"`
}

// SecurityConfigJSON mirrors SecuritySpec for JSON import
type SecurityConfigJSON struct {
    AIThreatDetection  *bool `json:"aiThreatDetection,omitempty"`
    GoogleSafeBrowsing *bool `json:"googleSafeBrowsing,omitempty"`
    Cryptojacking      *bool `json:"cryptojacking,omitempty"`
    DNSRebinding       *bool `json:"dnsRebinding,omitempty"`
    IDNHomographs      *bool `json:"idnHomographs,omitempty"`
    Typosquatting      *bool `json:"typosquatting,omitempty"`
    DGA                *bool `json:"dga,omitempty"`
    NRD                *bool `json:"nrd,omitempty"`
    DDNS               *bool `json:"ddns,omitempty"`
    Parking            *bool `json:"parking,omitempty"`
    CSAM               *bool `json:"csam,omitempty"`
}

// PrivacyConfigJSON mirrors PrivacySpec for JSON import
type PrivacyConfigJSON struct {
    Blocklists        []BlocklistEntryJSON `json:"blocklists,omitempty"`
    Natives           []NativeEntryJSON    `json:"natives,omitempty"`
    DisguisedTrackers *bool                `json:"disguisedTrackers,omitempty"`
    AllowAffiliate    *bool                `json:"allowAffiliate,omitempty"`
}

// BlocklistEntryJSON represents a blocklist entry in JSON
type BlocklistEntryJSON struct {
    ID     string `json:"id"`
    Active *bool  `json:"active,omitempty"`
}

// NativeEntryJSON represents a native tracking entry in JSON
type NativeEntryJSON struct {
    ID     string `json:"id"`
    Active *bool  `json:"active,omitempty"`
}

// ParentalControlConfigJSON mirrors ParentalControlSpec for JSON import
type ParentalControlConfigJSON struct {
    Categories            []CategoryEntryJSON `json:"categories,omitempty"`
    Services              []ServiceEntryJSON  `json:"services,omitempty"`
    SafeSearch            *bool               `json:"safeSearch,omitempty"`
    YouTubeRestrictedMode *bool               `json:"youtubeRestrictedMode,omitempty"`
}

// CategoryEntryJSON represents a category entry in JSON
type CategoryEntryJSON struct {
    ID     string `json:"id"`
    Active *bool  `json:"active,omitempty"`
}

// ServiceEntryJSON represents a service entry in JSON
type ServiceEntryJSON struct {
    ID     string `json:"id"`
    Active *bool  `json:"active,omitempty"`
}

// SettingsConfigJSON mirrors SettingsSpec for JSON import
type SettingsConfigJSON struct {
    Logs        *LogsConfigJSON        `json:"logs,omitempty"`
    BlockPage   *BlockPageConfigJSON   `json:"blockPage,omitempty"`
    Performance *PerformanceConfigJSON `json:"performance,omitempty"`
    Web3        *bool                  `json:"web3,omitempty"`
}

// LogsConfigJSON represents logging settings in JSON
type LogsConfigJSON struct {
    Enabled       *bool  `json:"enabled,omitempty"`
    LogClientsIPs *bool  `json:"logClientsIPs,omitempty"`
    LogDomains    *bool  `json:"logDomains,omitempty"`
    Retention     string `json:"retention,omitempty"`
}

// BlockPageConfigJSON represents block page settings in JSON
type BlockPageConfigJSON struct {
    Enabled *bool `json:"enabled,omitempty"`
}

// PerformanceConfigJSON represents performance settings in JSON
type PerformanceConfigJSON struct {
    ECS             *bool `json:"ecs,omitempty"`
    CacheBoost      *bool `json:"cacheBoost,omitempty"`
    CNAMEFlattening *bool `json:"cnameFlattening,omitempty"`
}

// DomainEntryJSON represents a domain entry in JSON
type DomainEntryJSON struct {
    Domain string `json:"domain"`
    Active *bool  `json:"active,omitempty"`
    Reason string `json:"reason,omitempty"`
}

// RewriteEntryJSON represents a rewrite entry in JSON
type RewriteEntryJSON struct {
    From string `json:"from"`
    To   string `json:"to"`
}
```

---

## Phase 3: Config Import Logic

**File: `internal/configimport/importer.go` (new file)**

```go
package configimport

import (
    "context"
    "encoding/json"
    "fmt"

    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"

    nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

// Importer handles importing profile configuration from ConfigMaps
type Importer struct {
    client client.Client
}

// NewImporter creates a new config importer
func NewImporter(c client.Client) *Importer {
    return &Importer{client: c}
}

// Import reads and parses configuration from a ConfigMap
func (i *Importer) Import(ctx context.Context, ref *nextdnsv1alpha1.ConfigImportRef, profileNS string) (*ProfileConfigJSON, string, error) {
    // 1. Determine namespace (default to profile's namespace)
    ns := ref.Namespace
    if ns == "" {
        ns = profileNS
    }

    // 2. Get ConfigMap
    cm := &corev1.ConfigMap{}
    if err := i.client.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: ns}, cm); err != nil {
        return nil, "", fmt.Errorf("failed to get ConfigMap: %w", err)
    }

    // 3. Get the config key (default: "config.json")
    key := ref.Key
    if key == "" {
        key = "config.json"
    }

    // 4. Parse JSON
    data, ok := cm.Data[key]
    if !ok {
        return nil, "", fmt.Errorf("key %q not found in ConfigMap", key)
    }

    var config ProfileConfigJSON
    if err := json.Unmarshal([]byte(data), &config); err != nil {
        return nil, "", fmt.Errorf("failed to parse JSON: %w", err)
    }

    return &config, cm.ResourceVersion, nil
}

// MergeWithSpec merges imported config with spec (spec takes precedence)
func MergeWithSpec(imported *ProfileConfigJSON, spec *nextdnsv1alpha1.NextDNSProfileSpec) {
    // Security: only apply imported if spec field is nil
    if spec.Security == nil && imported.Security != nil {
        spec.Security = convertSecurityConfig(imported.Security)
    } else if spec.Security != nil && imported.Security != nil {
        mergeSecurityConfig(spec.Security, imported.Security)
    }

    // Privacy: similar merge logic
    if spec.Privacy == nil && imported.Privacy != nil {
        spec.Privacy = convertPrivacyConfig(imported.Privacy)
    } else if spec.Privacy != nil && imported.Privacy != nil {
        mergePrivacyConfig(spec.Privacy, imported.Privacy)
    }

    // ParentalControl: similar merge logic
    if spec.ParentalControl == nil && imported.ParentalControl != nil {
        spec.ParentalControl = convertParentalControlConfig(imported.ParentalControl)
    } else if spec.ParentalControl != nil && imported.ParentalControl != nil {
        mergeParentalControlConfig(spec.ParentalControl, imported.ParentalControl)
    }

    // Settings: similar merge logic
    if spec.Settings == nil && imported.Settings != nil {
        spec.Settings = convertSettingsConfig(imported.Settings)
    } else if spec.Settings != nil && imported.Settings != nil {
        mergeSettingsConfig(spec.Settings, imported.Settings)
    }

    // Lists: merge (don't replace)
    if len(imported.Denylist) > 0 {
        spec.Denylist = append(spec.Denylist, convertDomainEntries(imported.Denylist)...)
    }
    if len(imported.Allowlist) > 0 {
        spec.Allowlist = append(spec.Allowlist, convertDomainEntries(imported.Allowlist)...)
    }
    if len(imported.Rewrites) > 0 {
        spec.Rewrites = append(spec.Rewrites, convertRewriteEntries(imported.Rewrites)...)
    }
}

// Helper functions to convert JSON types to API types
func convertSecurityConfig(j *SecurityConfigJSON) *nextdnsv1alpha1.SecuritySpec {
    return &nextdnsv1alpha1.SecuritySpec{
        AIThreatDetection:  j.AIThreatDetection,
        GoogleSafeBrowsing: j.GoogleSafeBrowsing,
        Cryptojacking:      j.Cryptojacking,
        DNSRebinding:       j.DNSRebinding,
        IDNHomographs:      j.IDNHomographs,
        Typosquatting:      j.Typosquatting,
        DGA:                j.DGA,
        NRD:                j.NRD,
        DDNS:               j.DDNS,
        Parking:            j.Parking,
        CSAM:               j.CSAM,
    }
}

func mergeSecurityConfig(spec *nextdnsv1alpha1.SecuritySpec, imported *SecurityConfigJSON) {
    // Only merge fields that are nil in spec
    if spec.AIThreatDetection == nil {
        spec.AIThreatDetection = imported.AIThreatDetection
    }
    if spec.GoogleSafeBrowsing == nil {
        spec.GoogleSafeBrowsing = imported.GoogleSafeBrowsing
    }
    if spec.Cryptojacking == nil {
        spec.Cryptojacking = imported.Cryptojacking
    }
    if spec.DNSRebinding == nil {
        spec.DNSRebinding = imported.DNSRebinding
    }
    if spec.IDNHomographs == nil {
        spec.IDNHomographs = imported.IDNHomographs
    }
    if spec.Typosquatting == nil {
        spec.Typosquatting = imported.Typosquatting
    }
    if spec.DGA == nil {
        spec.DGA = imported.DGA
    }
    if spec.NRD == nil {
        spec.NRD = imported.NRD
    }
    if spec.DDNS == nil {
        spec.DDNS = imported.DDNS
    }
    if spec.Parking == nil {
        spec.Parking = imported.Parking
    }
    if spec.CSAM == nil {
        spec.CSAM = imported.CSAM
    }
}

// ... additional helper functions for other config sections
```

---

## Phase 4: Controller Integration

**File: `internal/controller/nextdnsprofile_controller.go`**

Add to reconciliation flow (after fetching the profile, before syncing with NextDNS):

```go
func (r *NextDNSProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ... existing code to fetch profile ...

    // NEW: Import configuration from ConfigMap if specified
    if profile.Spec.ConfigImportRef != nil {
        importer := configimport.NewImporter(r.Client)
        imported, resourceVersion, err := importer.Import(ctx, profile.Spec.ConfigImportRef, profile.Namespace)
        if err != nil {
            // Set condition and return error
            r.setCondition(&profile, "ConfigImportFailed", metav1.ConditionFalse, "ImportError", err.Error())
            return ctrl.Result{}, err
        }

        // Merge imported config with spec (spec takes precedence)
        configimport.MergeWithSpec(imported, &profile.Spec)

        // Update import status
        if profile.Status.ConfigImport == nil {
            profile.Status.ConfigImport = &nextdnsv1alpha1.ConfigImportStatus{}
        }
        profile.Status.ConfigImport.ConfigMapResourceVersion = resourceVersion
        profile.Status.ConfigImport.LastImportTime = &metav1.Time{Time: time.Now()}
    }

    // ... rest of existing reconciliation logic ...
}
```

Add watch for ConfigMap changes:

```go
func (r *NextDNSProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&nextdnsv1alpha1.NextDNSProfile{}).
        // NEW: Watch ConfigMaps referenced by profiles
        Watches(
            &corev1.ConfigMap{},
            handler.EnqueueRequestsFromMapFunc(r.findProfilesForConfigMap),
        ).
        Complete(r)
}

func (r *NextDNSProfileReconciler) findProfilesForConfigMap(ctx context.Context, obj client.Object) []reconcile.Request {
    cm := obj.(*corev1.ConfigMap)

    // List all profiles
    var profiles nextdnsv1alpha1.NextDNSProfileList
    if err := r.Client.List(ctx, &profiles); err != nil {
        return nil
    }

    var requests []reconcile.Request
    for _, profile := range profiles.Items {
        if profile.Spec.ConfigImportRef != nil {
            ns := profile.Spec.ConfigImportRef.Namespace
            if ns == "" {
                ns = profile.Namespace
            }
            if profile.Spec.ConfigImportRef.Name == cm.Name && ns == cm.Namespace {
                requests = append(requests, reconcile.Request{
                    NamespacedName: types.NamespacedName{
                        Name:      profile.Name,
                        Namespace: profile.Namespace,
                    },
                })
            }
        }
    }
    return requests
}
```

---

## Phase 5: Validation

**File: `internal/configimport/validation.go` (new file)**

```go
package configimport

import (
    "encoding/json"
    "fmt"
)

// ValidateJSON validates the structure and values of imported JSON
func ValidateJSON(data []byte) error {
    var config ProfileConfigJSON
    if err := json.Unmarshal(data, &config); err != nil {
        return fmt.Errorf("invalid JSON syntax: %w", err)
    }

    // Validate privacy settings
    if config.Privacy != nil && config.Privacy.Blocklists != nil {
        for _, bl := range config.Privacy.Blocklists {
            if bl.ID == "" {
                return fmt.Errorf("blocklist entry missing required 'id' field")
            }
        }
    }

    // Validate domain entries
    for i, d := range config.Denylist {
        if d.Domain == "" {
            return fmt.Errorf("denylist entry %d missing required 'domain' field", i)
        }
    }
    for i, d := range config.Allowlist {
        if d.Domain == "" {
            return fmt.Errorf("allowlist entry %d missing required 'domain' field", i)
        }
    }

    // Validate rewrite entries
    for i, r := range config.Rewrites {
        if r.From == "" {
            return fmt.Errorf("rewrite entry %d missing required 'from' field", i)
        }
        if r.To == "" {
            return fmt.Errorf("rewrite entry %d missing required 'to' field", i)
        }
    }

    return nil
}
```

---

## Phase 6: Add New Condition Type

Add to `api/v1alpha1/` (new file or existing conditions file):

```go
const (
    // ConditionConfigImported indicates if configuration was successfully imported
    ConditionConfigImported = "ConfigImported"
)
```

---

## Tasks Checklist

- [ ] **API Types**: Add `ConfigImportRef`, `ConfigImportStatus` to CRD types
- [ ] **Run `make generate manifests`**: Regenerate CRDs
- [ ] **Import Package**: Create `internal/configimport/` with types, importer, validation
- [ ] **Controller Integration**: Add import logic to reconciliation loop
- [ ] **Watch Setup**: Add ConfigMap watch to controller
- [ ] **Unit Tests**: Test import, merge, and validation logic
- [ ] **Integration Tests**: Test end-to-end ConfigMap import flow
- [ ] **Documentation**: Update README with ConfigMap import examples
- [ ] **Sample Resources**: Add sample ConfigMap with example JSON structure

---

## Example Usage

### ConfigMap with profile configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-profile-config
data:
  config.json: |
    {
      "security": {
        "aiThreatDetection": true,
        "googleSafeBrowsing": true,
        "cryptojacking": true
      },
      "privacy": {
        "blocklists": [
          {"id": "nextdns-recommended"},
          {"id": "oisd"}
        ],
        "disguisedTrackers": true
      },
      "denylist": [
        {"domain": "ads.example.com", "reason": "Imported from backup"}
      ]
    }
```

### NextDNSProfile referencing the ConfigMap:

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSProfile
metadata:
  name: my-profile
spec:
  name: "My Profile"
  credentialsRef:
    name: nextdns-credentials
  configImportRef:
    name: my-profile-config
    key: config.json
  # Spec fields override imported values
  security:
    nrd: true  # This overrides imported security.nrd
```

---

## Open Questions to Address

1. **Should imported ConfigMap changes trigger re-sync?**
   - Recommendation: Yes, via watch on ConfigMap resources

2. **Support for partial imports?**
   - Recommendation: Yes, only set fields are imported

3. **Validation strictness?**
   - Recommendation: Warn on unknown fields, error on invalid values

---

## Estimated Effort

| Phase | Complexity | Files Modified |
|-------|------------|----------------|
| API Types | Low | 1-2 files |
| JSON Schema | Medium | 1 new file |
| Import Logic | Medium | 1 new file |
| Controller Integration | Medium | 1 file |
| Validation | Low | 1 new file |
| Tests | Medium | 2-3 new files |
| Documentation | Low | 1-2 files |
