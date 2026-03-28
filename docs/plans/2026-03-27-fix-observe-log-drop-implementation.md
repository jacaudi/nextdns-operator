# Fix Observe Mode Log Drop Inversion -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix observe mode to correctly read and invert the API's `logs.drop` fields into user-friendly `LogClientsIPs`/`LogDomains` booleans in `observedConfig` and `suggestedSpec`.

**Architecture:** Add `LogClientsIPs` and `LogDomains` bools to `ObservedLogs`. In `readFullProfile`, read `settings.Logs.Drop.IP`/`Drop.Domain` and invert them. In `buildSuggestedSpec`, pass the new fields through as `*bool` pointers.

**Tech Stack:** Go, Kubebuilder, controller-runtime, nextdns-go v0.11.0

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

### Task 1: Add LogClientsIPs/LogDomains to ObservedLogs and wire through readFullProfile + buildSuggestedSpec

**Files:**
- Modify: `api/v1alpha1/nextdnsprofile_observed_types.go:111-115`
- Modify: `internal/controller/nextdnsprofile_controller.go:882-887` (readFullProfile)
- Modify: `internal/controller/nextdnsprofile_controller.go:1015-1020` (buildSuggestedSpec)
- Modify: `internal/controller/nextdnsprofile_controller_test.go` (two test updates)

**Step 1: Write failing test -- update TestBuildSuggestedSpec**

In `internal/controller/nextdnsprofile_controller_test.go`, find the `ObservedLogs` setup in the `TestBuildSuggestedSpec` test (line 3058). Update it to include the new fields:

Change:
```go
Logs: &nextdnsv1alpha1.ObservedLogs{Enabled: true, Retention: 30},
```

To:
```go
Logs: &nextdnsv1alpha1.ObservedLogs{Enabled: true, Retention: 30, LogClientsIPs: true, LogDomains: false},
```

Then update the assertions (lines 3134-3135). Change:
```go
assert.Nil(t, suggested.Settings.Logs.LogClientsIPs) // Not available from API
assert.Nil(t, suggested.Settings.Logs.LogDomains)    // Not available from API
```

To:
```go
assert.Equal(t, boolPtr(true), suggested.Settings.Logs.LogClientsIPs)
assert.Equal(t, boolPtr(false), suggested.Settings.Logs.LogDomains)
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/controller/... -run TestBuildSuggestedSpec -v`
Expected: FAIL -- `LogClientsIPs` field doesn't exist on ObservedLogs

**Step 3: Add LogClientsIPs and LogDomains to ObservedLogs**

In `api/v1alpha1/nextdnsprofile_observed_types.go`, replace the `ObservedLogs` struct (lines 111-115):

```go
// ObservedLogs represents observed logging settings
type ObservedLogs struct {
	Enabled       bool `json:"enabled"`
	Retention     int  `json:"retention,omitempty"`
	// LogClientsIPs indicates whether client IPs are logged.
	// Derived from the API's Drop.IP field (inverted: LogClientsIPs = !Drop.IP).
	LogClientsIPs bool `json:"logClientsIPs"`
	// LogDomains indicates whether queried domains are logged.
	// Derived from the API's Drop.Domain field (inverted: LogDomains = !Drop.Domain).
	LogDomains    bool `json:"logDomains"`
}
```

**Step 4: Update readFullProfile to read Drop fields**

In `internal/controller/nextdnsprofile_controller.go`, replace the Logs block in `readFullProfile` (lines 882-887):

```go
	if settings.Logs != nil {
		observed.Settings.Logs = &nextdnsv1alpha1.ObservedLogs{
			Enabled:   settings.Logs.Enabled,
			Retention: settings.Logs.Retention,
		}
		// Invert Drop fields to user-friendly positive semantics:
		// API Drop.IP=true means "don't log IPs" -> LogClientsIPs=false
		if settings.Logs.Drop != nil {
			observed.Settings.Logs.LogClientsIPs = !settings.Logs.Drop.IP
			observed.Settings.Logs.LogDomains = !settings.Logs.Drop.Domain
		} else {
			// Default: log both when Drop is not set
			observed.Settings.Logs.LogClientsIPs = true
			observed.Settings.Logs.LogDomains = true
		}
	}
```

**Step 5: Update buildSuggestedSpec to pass through the new fields**

In `internal/controller/nextdnsprofile_controller.go`, replace the Logs block in `buildSuggestedSpec` (lines 1015-1020):

```go
		if observed.Settings.Logs != nil {
			suggested.Settings.Logs = &nextdnsv1alpha1.LogsSpec{
				Enabled:       boolPtr(observed.Settings.Logs.Enabled),
				Retention:     formatRetentionString(observed.Settings.Logs.Retention),
				LogClientsIPs: boolPtr(observed.Settings.Logs.LogClientsIPs),
				LogDomains:    boolPtr(observed.Settings.Logs.LogDomains),
			}
		}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/controller/... -run TestBuildSuggestedSpec -v`
Expected: PASS

**Step 7: Update TestReconcile_ObserveMode_Success mock setup and assertions**

In `internal/controller/nextdnsprofile_controller_test.go`, find the mock settings setup (line 2532-2533). Update to include Drop:

Change:
```go
mockNDS.Settings["abc123"] = &sdknextdns.Settings{
    Logs:      &sdknextdns.SettingsLogs{Enabled: true, Retention: 7},
```

To:
```go
mockNDS.Settings["abc123"] = &sdknextdns.Settings{
    Logs: &sdknextdns.SettingsLogs{
        Enabled:   true,
        Retention: 7,
        Drop: &sdknextdns.SettingsLogsDrop{
            IP:     false, // IPs ARE logged
            Domain: false, // Domains ARE logged
        },
    },
```

Then add assertions after the existing `Logs.Enabled` check (after line ~2570):

```go
	assert.True(t, updated.Status.ObservedConfig.Settings.Logs.LogClientsIPs)
	assert.True(t, updated.Status.ObservedConfig.Settings.Logs.LogDomains)
```

And add suggestedSpec assertions (after the existing suggestedSpec Logs assertions):

```go
	assert.Equal(t, boolPtr(true), updated.Status.SuggestedSpec.Settings.Logs.LogClientsIPs)
	assert.Equal(t, boolPtr(true), updated.Status.SuggestedSpec.Settings.Logs.LogDomains)
```

**Step 8: Run all tests**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: All PASS

**Step 9: Commit**

```bash
git add api/v1alpha1/nextdnsprofile_observed_types.go internal/controller/nextdnsprofile_controller.go internal/controller/nextdnsprofile_controller_test.go
git commit -m "fix: read and invert logs.drop fields in observe mode (#75)

ObservedLogs now includes LogClientsIPs and LogDomains, derived from the
API's Drop.IP and Drop.Domain fields with inverted semantics:
- Drop.IP=false (IPs are kept) -> LogClientsIPs=true
- Drop.Domain=true (domains are dropped) -> LogDomains=false

buildSuggestedSpec passes these through to the suggested LogsSpec."
```

---

### Task 2: Regenerate CRDs and verify

**Step 1: Regenerate deepcopy and CRDs**

Run: `make generate manifests && make sync-helm-crds`

**Step 2: Verify CRD contains new fields**

Run: `grep -A5 "logClientsIPs" config/crd/bases/nextdns.io_nextdnsprofiles.yaml | head -10`
Expected: Shows `logClientsIPs` and `logDomains` in the observedConfig schema

**Step 3: Run full test suite**

Run: `make test`
Expected: All PASS

**Step 4: Commit**

```bash
git add api/v1alpha1/zz_generated.deepcopy.go config/crd/ chart/crds/
git commit -m "chore: regenerate CRDs for observe mode log drop fields (#75)"
```

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. `grep logClientsIPs config/crd/bases/nextdns.io_nextdnsprofiles.yaml` -- field in CRD
4. Observe mode test: Drop.IP=false -> LogClientsIPs=true in observedConfig
5. Observe mode test: LogClientsIPs=true -> suggestedSpec.logClientsIPs=true
