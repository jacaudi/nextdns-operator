# Observe Mode Completeness -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete observe mode by adding missing fields: setup section (#76), parental control gaps (#77), and logs.location (#78).

**Architecture:** Add new observed types, extend existing types with missing fields, add GetSetup to client interface, populate in readFullProfile, pass through in buildSuggestedSpec where applicable. Update documentation for all changes.

**Tech Stack:** Go, Kubebuilder, controller-runtime, nextdns-go v0.12.0

**Working directory:** New worktree from `main`

> **For Claude:** REQUIRED SUB-SKILLS (must use in order):
> 1. `superpowers:using-git-worktrees` -- Isolate work in a dedicated worktree
> 2. Choose execution mode (load `superpowers:test-driven-development` alongside):
>    - **Subagent-Driven (this session):** `superpowers:subagent-driven-development` + `superpowers:test-driven-development`
>    - **Parallel Session (separate):** `superpowers:executing-plans` + `superpowers:test-driven-development`
> 3. `superpowers:verification-before-completion` -- Verify all tests pass before claiming done
> 4. `superpowers:requesting-code-review` -- Code review after EACH task
> 5. After ALL tasks: dispatch independent and comprehensive code review on full diff
> 6. `superpowers:finishing-a-development-branch` -- Complete the branch

---

### Task 1: Add Setup section to observe mode (#76)

**Files:**
- Modify: `api/v1alpha1/nextdnsprofile_observed_types.go` (add ObservedSetup, ObservedLinkedIP types + field on ObservedConfig)
- Modify: `internal/nextdns/interface.go` (add GetSetup to ClientInterface)
- Modify: `internal/nextdns/client.go` (add GetSetup implementation)
- Modify: `internal/nextdns/mock_client.go` (add GetSetup mock + Setup data storage)
- Modify: `internal/controller/nextdnsprofile_controller.go` (populate in readFullProfile)
- Modify: `internal/controller/nextdnsprofile_controller_test.go` (test observe mode includes setup)

**New types to add to `nextdnsprofile_observed_types.go`:**

```go
// ObservedSetup represents observed DNS setup/endpoint configuration.
// This is read-only data from the API — not user-configurable via spec.
type ObservedSetup struct {
	// IPv4 contains DNS-over-HTTPS IPv4 addresses
	// +optional
	IPv4 []string `json:"ipv4,omitempty"`
	// IPv6 contains DNS-over-HTTPS IPv6 addresses
	// +optional
	IPv6 []string `json:"ipv6,omitempty"`
	// LinkedIP contains linked IP configuration
	// +optional
	LinkedIP *ObservedLinkedIP `json:"linkedIP,omitempty"`
	// DNSCrypt contains the DNSCrypt protocol stamp
	// +optional
	DNSCrypt string `json:"dnscrypt,omitempty"`
}

// ObservedLinkedIP represents observed linked IP configuration.
// Note: updateToken is excluded for security — it is sensitive.
type ObservedLinkedIP struct {
	// Servers contains the linked IP DNS server addresses
	// +optional
	Servers []string `json:"servers,omitempty"`
	// IP is the currently linked IP address
	// +optional
	IP string `json:"ip,omitempty"`
	// DDNS is the dynamic DNS hostname
	// +optional
	DDNS string `json:"ddns,omitempty"`
}
```

Add `Setup *ObservedSetup` field to `ObservedConfig` struct.

**ClientInterface addition:**
```go
// Setup operations
GetSetup(ctx context.Context, profileID string) (*nextdns.Setup, error)
```

**Client implementation** (`client.go`):
```go
func (c *Client) GetSetup(ctx context.Context, profileID string) (*nextdns.Setup, error) {
	start := time.Now()
	request := &nextdns.GetSetupRequest{ProfileID: profileID}
	setup, err := c.client.Setup.Get(ctx, request)
	metrics.RecordAPIRequest("GetSetup", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get setup: %w", err)
	}
	return setup, nil
}
```

**readFullProfile addition** (after rewrites, before return):
```go
// Get setup (read-only endpoint data)
setup, err := client.GetSetup(ctx, profileID)
if err != nil {
	return nil, "", fmt.Errorf("failed to get setup: %w", err)
}
observed.Setup = &nextdnsv1alpha1.ObservedSetup{
	IPv4:     setup.Ipv4,
	IPv6:     setup.Ipv6,
	DNSCrypt: setup.Dnscrypt,
}
if setup.LinkedIP != nil {
	observed.Setup.LinkedIP = &nextdnsv1alpha1.ObservedLinkedIP{
		Servers: setup.LinkedIP.Servers,
		IP:      setup.LinkedIP.IP,
		DDNS:    setup.LinkedIP.Ddns,
		// updateToken intentionally excluded (sensitive)
	}
}
```

**No suggestedSpec entry** -- setup is read-only, not user-configurable.

**TDD steps:**
1. Write test expecting `observedConfig.setup` to be populated with IPv4/IPv6/DNSCrypt/LinkedIP
2. Run test -- fails (GetSetup not in interface)
3. Add types, interface method, client, mock, controller wiring
4. Run test -- passes
5. Run full suite
6. Commit: `feat: add setup section to observe mode (#76)`

---

### Task 2: Add parental control BlockBypass and Recreation (#77)

**Files:**
- Modify: `api/v1alpha1/nextdnsprofile_observed_types.go` (add fields to ObservedParentalControl, ObservedCategoryEntry + new ObservedRecreation types)
- Modify: `api/v1alpha1/nextdnsprofile_types.go` (add BlockBypass to ParentalControlSpec, Recreation to CategoryEntry)
- Modify: `internal/controller/nextdnsprofile_controller.go` (read blockBypass + recreation in readFullProfile, wire blockBypass in syncWithNextDNS, pass through in buildSuggestedSpec)
- Modify: `internal/nextdns/client.go` (add BlockBypass to ParentalControlConfig)
- Modify: `internal/nextdns/mock_client.go` (update mock)
- Modify: `internal/controller/nextdnsprofile_controller_test.go`

**Type changes:**

Add to `ObservedParentalControl`:
```go
BlockBypass bool                 `json:"blockBypass"`
Recreation *ObservedRecreation  `json:"recreation,omitempty"`
```

New types:
```go
// ObservedRecreation represents observed recreation schedule settings
type ObservedRecreation struct {
	// Times contains the recreation schedule per day of week
	// +optional
	Times *ObservedRecreationTimes `json:"times,omitempty"`
	// Timezone is the timezone for the recreation schedule
	// +optional
	Timezone string `json:"timezone,omitempty"`
}

// ObservedRecreationTimes contains per-day recreation intervals
type ObservedRecreationTimes struct {
	Monday    *ObservedRecreationInterval `json:"monday,omitempty"`
	Tuesday   *ObservedRecreationInterval `json:"tuesday,omitempty"`
	Wednesday *ObservedRecreationInterval `json:"wednesday,omitempty"`
	Thursday  *ObservedRecreationInterval `json:"thursday,omitempty"`
	Friday    *ObservedRecreationInterval `json:"friday,omitempty"`
	Saturday  *ObservedRecreationInterval `json:"saturday,omitempty"`
	Sunday    *ObservedRecreationInterval `json:"sunday,omitempty"`
}

// ObservedRecreationInterval represents a time range
type ObservedRecreationInterval struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
```

Add to `ObservedCategoryEntry`:
```go
Recreation bool `json:"recreation"`
```

Add to `ParentalControlSpec` (spec type):
```go
// BlockBypass prevents bypassing parental controls
// +kubebuilder:default=false
// +optional
BlockBypass *bool `json:"blockBypass,omitempty"`
```

Add to `CategoryEntry` (spec type):
```go
// Recreation indicates if this category allows recreation time exceptions
// +kubebuilder:default=false
// +optional
Recreation *bool `json:"recreation,omitempty"`
```

**readFullProfile changes** (line ~828):
- Add `BlockBypass: pc.BlockBypass` to ObservedParentalControl init
- Add recreation schedule mapping from `pc.Recreation`
- Add `Recreation: cat.Recreation` to category loop

**syncWithNextDNS changes:**
- Add `BlockBypass` to ParentalControlConfig
- Wire `boolValue(profile.Spec.ParentalControl.BlockBypass, false)` into managed mode

**buildSuggestedSpec changes:**
- Add `BlockBypass: boolPtr(observed.ParentalControl.BlockBypass)` to suggested ParentalControlSpec
- Add `Recreation: boolPtr(cat.Recreation)` to category loop

**TDD steps:**
1. Write test with blockBypass + recreation in mock parental control data
2. Run test -- fails (fields don't exist)
3. Add types and controller wiring
4. Run test -- passes
5. Run full suite
6. Commit: `feat: add blockBypass and recreation to parental control (#77)`

---

### Task 3: Add logs.location to settings (#78)

**Files:**
- Modify: `api/v1alpha1/nextdnsprofile_observed_types.go` (add Location to ObservedLogs)
- Modify: `api/v1alpha1/nextdnsprofile_types.go` (add Location to LogsSpec)
- Modify: `internal/controller/nextdnsprofile_controller.go` (read in readFullProfile, pass in buildSuggestedSpec, wire in syncWithNextDNS)
- Modify: `internal/nextdns/client.go` (add Location to SettingsConfig)
- Modify: `internal/nextdns/mock_client.go`
- Modify: `internal/controller/nextdnsprofile_controller_test.go`

**Type changes:**

Add to `ObservedLogs`:
```go
// Location is the log storage location (e.g., "eu", "us", "ch")
Location string `json:"location,omitempty"`
```

Add to `LogsSpec`:
```go
// Location specifies the log storage location (e.g., "eu", "us", "ch")
// +optional
Location string `json:"location,omitempty"`
```

Add to `SettingsConfig`:
```go
Location string
```

**readFullProfile** (line ~899): Add `Location: settings.Logs.Location`

**buildSuggestedSpec**: Add `Location: observed.Settings.Logs.Location`

**syncWithNextDNS** (managed mode): Add `settingsConfig.Location = profile.Spec.Settings.Logs.Location` (if Logs is not nil)

**UpdateSettings** client method: Add `Location: config.Location` to the SettingsLogs in the PATCH body

**TDD steps:**
1. Write test with location "eu" in mock settings
2. Run test -- fails (field doesn't exist)
3. Add types and wiring
4. Run test -- passes
5. Run full suite
6. Commit: `feat: add logs.location to settings (#78)`

---

### Task 4: Regenerate CRDs, update documentation, verify

**Files:**
- Regenerate: `api/v1alpha1/zz_generated.deepcopy.go`
- Regenerate: `config/crd/bases/nextdns.io_nextdnsprofiles.yaml`
- Regenerate: `chart/crds/nextdns.io_nextdnsprofiles.yaml`
- Modify: `docs/README.md` (update spec tables, status tables, observe mode docs)

**Step 1:** `make generate manifests && make sync-helm-crds`

**Step 2:** Update `docs/README.md`:
- Add `setup` to observe mode section (explain it's read-only)
- Add `blockBypass` to ParentalControlSpec table
- Add `recreation` to CategoryEntry table
- Add `location` to LogsSpec table
- Update observe mode example to mention setup data

**Step 3:** `make test` -- all pass

**Step 4:** Commit: `chore: regenerate CRDs and update docs for observe mode completeness`

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. `grep "setup" config/crd/bases/nextdns.io_nextdnsprofiles.yaml` -- setup in CRD
4. `grep "blockBypass" config/crd/bases/nextdns.io_nextdnsprofiles.yaml` -- blockBypass in CRD
5. `grep "location" config/crd/bases/nextdns.io_nextdnsprofiles.yaml` -- location in CRD
6. Observe mode test: setup, blockBypass, recreation, location all populated
7. Managed mode test: blockBypass and location synced to API
