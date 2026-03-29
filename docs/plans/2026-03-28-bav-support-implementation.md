# BAV (Bypass Age Verification) Support -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add BAV (Bypass Age Verification) boolean to observe mode, managed mode, and suggestedSpec.

**Architecture:** Same pattern as Web3 -- add to observed types, spec types, settings config, and wire through all three controller paths. Single task since it's one field following an established pattern.

**Tech Stack:** Go, Kubebuilder, controller-runtime, nextdns-go v0.13.0

**Working directory:** New worktree from `main`

---

### Task 1: Add BAV field everywhere and wire through

**Files:**
- Modify: `api/v1alpha1/nextdnsprofile_observed_types.go:139` (add after Web3)
- Modify: `api/v1alpha1/nextdnsprofile_types.go:299-302` (add after Web3)
- Modify: `internal/nextdns/client.go:82,534` (SettingsConfig + UpdateSettings)
- Modify: `internal/controller/nextdnsprofile_controller.go:631-632,649,933,1098` (all 4 controller locations)
- Modify: `internal/nextdns/mock_client.go` (UpdateSettings mock)
- Modify: `internal/controller/nextdnsprofile_controller_test.go` (tests)

**Step 1: Write failing test**

Update `TestReconcile_ObserveMode_Success` mock settings to include `BAV: true` (alongside existing `Web3: false`). Add assertion:
```go
assert.True(t, updated.Status.ObservedConfig.Settings.BAV)
```

Update `TestBuildSuggestedSpec` observed settings to include `BAV: true`. Add assertion:
```go
assert.Equal(t, boolPtr(true), suggested.Settings.BAV)
```

Update `TestSyncWithNextDNS_FullSettings` spec to include `BAV: boolPtr(true)`. Add assertion that mock settings have `BAV: true`.

**Step 2: Run test, verify fails**

Run: `go test ./internal/controller/... -run "TestReconcile_ObserveMode_Success|TestBuildSuggestedSpec|TestSyncWithNextDNS_FullSettings" -v`
Expected: FAIL -- BAV field doesn't exist

**Step 3: Add BAV to all types and wire through**

In each file, add BAV right after Web3 following the same pattern:

- `ObservedSettings`: `BAV bool \`json:"bav"\``
- `SettingsSpec`: `BAV *bool \`json:"bav,omitempty"\`` with `+kubebuilder:default=false` and `+optional`
- `SettingsConfig`: `BAV bool`
- `readFullProfile` (line 933): `BAV: settings.BAV` (alongside `Web3: settings.Web3`)
- `buildSuggestedSpec` (line 1098): `BAV: boolPtr(observed.Settings.BAV)`
- `syncWithNextDNS` (line 631-632): Add `BAV: false` default, then `settingsConfig.BAV = boolValue(profile.Spec.Settings.BAV, false)` at line 649
- `UpdateSettings` client (line 534): `BAV: config.BAV` in the SDK Settings struct
- Mock `UpdateSettings`: `BAV: config.BAV` in stored settings

**Step 4: Run tests, verify pass**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: All PASS

**Step 5: Commit**

```bash
git add api/ internal/
git commit -m "feat: add BAV (Bypass Age Verification) to settings (#78)

Completes the deferred BAV item from #78 now that nextdns-go v0.13.0
exposes Settings.BAV. Wired through observe, managed, and suggestedSpec."
```

---

### Task 2: Regenerate CRDs, update docs, verify

**Step 1:** `make generate manifests && make sync-helm-crds`

**Step 2:** Update `docs/README.md` -- add BAV to SettingsSpec table:
```
| `bav` | *bool | `false` | Bypass Age Verification |
```

**Step 3:** `make test` -- all pass

**Step 4:** Verify: `grep bav config/crd/bases/nextdns.io_nextdnsprofiles.yaml`

**Step 5:** Commit:
```bash
git add api/v1alpha1/zz_generated.deepcopy.go config/crd/ chart/crds/ docs/README.md
git commit -m "chore: regenerate CRDs and update docs for BAV support"
```

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. `grep bav config/crd/bases/nextdns.io_nextdnsprofiles.yaml` -- field in CRD
4. Observe mode: BAV read from API into observedConfig
5. Managed mode: BAV synced to API
6. SuggestedSpec: BAV passed through
