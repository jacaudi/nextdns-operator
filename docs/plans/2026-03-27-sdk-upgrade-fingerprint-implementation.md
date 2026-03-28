# SDK Upgrade + Fingerprint Fix -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Upgrade nextdns-go SDK to v0.12.0 and use the real API fingerprint instead of a hardcoded DNS endpoint.

**Architecture:** Bump the SDK dependency, then replace both fingerprint assignment sites (managed + observe mode) with the value from `Profile.Fingerprint`. In observe mode, `readFullProfile` already calls `GetProfile` -- return the fingerprint as a second value. Update the mock to wire the fingerprint through.

**Tech Stack:** Go, nextdns-go v0.12.0, controller-runtime

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

### Task 1: Upgrade nextdns-go SDK to v0.12.0

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Bump the SDK version**

Run: `go get github.com/jacaudi/nextdns-go@v0.12.0 && go mod tidy`

**Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

**Step 3: Run full test suite**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: All PASS (no breaking changes in v0.12.0)

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): upgrade nextdns-go to v0.12.0

Adds Profile.Fingerprint field needed to fix #74."
```

---

### Task 2: Wire fingerprint through mock and fix controller

**Files:**
- Modify: `internal/nextdns/mock_client.go:776-783` (SetProfile to store Fingerprint)
- Modify: `internal/controller/nextdnsprofile_controller.go:493-508` (managed mode fingerprint)
- Modify: `internal/controller/nextdnsprofile_controller.go:720-722` (observe mode fingerprint)
- Modify: `internal/controller/nextdnsprofile_controller.go:748-757` (readFullProfile return)
- Modify: `internal/controller/nextdnsprofile_controller_test.go` (assertions)

**Step 1: Write failing test -- assert fingerprint comes from API, not DNS endpoint**

In `internal/controller/nextdnsprofile_controller_test.go`, find `TestReconcile_ObserveMode_Success`. The mock setup calls `mockNDS.SetProfile("abc123", "Remote Profile", "abc123.dns.nextdns.io")`. Change the fingerprint argument to the real API fingerprint:

```go
mockNDS.SetProfile("abc123", "Remote Profile", "fp04d207c439ee4858")
```

Then find the assertion for `status.fingerprint` (around line where it checks `updated.Status.Fingerprint`). If there isn't one, add:

```go
assert.Equal(t, "fp04d207c439ee4858", updated.Status.Fingerprint)
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/controller/... -run TestReconcile_ObserveMode_Success -v`
Expected: FAIL -- fingerprint is still `abc123.dns.nextdns.io`

**Step 3: Update mock SetProfile to store Fingerprint**

In `internal/nextdns/mock_client.go`, find `SetProfile` (line 776). It currently ignores the fingerprint param:

```go
func (m *MockClient) SetProfile(profileID, name, fingerprint string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.Profiles[profileID] = &nextdns.Profile{
        Name: name,
    }
}
```

Change to:

```go
func (m *MockClient) SetProfile(profileID, name, fingerprint string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.Profiles[profileID] = &nextdns.Profile{
        Name:        name,
        Fingerprint: fingerprint,
    }
}
```

**Step 4: Update readFullProfile to return fingerprint**

In `internal/controller/nextdnsprofile_controller.go`, change `readFullProfile` signature to return the fingerprint as well:

Change:
```go
func (r *NextDNSProfileReconciler) readFullProfile(ctx context.Context, client nextdns.ClientInterface, profileID string) (*nextdnsv1alpha1.ObservedConfig, error) {
```

To:
```go
func (r *NextDNSProfileReconciler) readFullProfile(ctx context.Context, client nextdns.ClientInterface, profileID string) (*nextdnsv1alpha1.ObservedConfig, string, error) {
```

Update the body -- after `profile, err := client.GetProfile(ctx, profileID)`, capture the fingerprint. Update all return statements to include the fingerprint string (empty string for errors, `profile.Fingerprint` for success).

The function currently ends with `return observed, nil`. Change to:
```go
return observed, profile.Fingerprint, nil
```

Note: `profile` is the variable from `client.GetProfile` at the top of the function. It may be shadowed later -- use a separate variable name if needed. Check if `profile` is reused elsewhere in the function.

All error returns become: `return nil, "", fmt.Errorf(...)`

**Step 5: Update observe mode caller**

In `reconcileObserveMode`, the call at line 709:
```go
observed, err := r.readFullProfile(ctx, client, profile.Spec.ProfileID)
```

Change to:
```go
observed, fingerprint, err := r.readFullProfile(ctx, client, profile.Spec.ProfileID)
```

Then at line 722, change:
```go
profile.Status.Fingerprint = profile.Spec.ProfileID + ".dns.nextdns.io"
```

To:
```go
profile.Status.Fingerprint = fingerprint
```

**Step 6: Update managed mode fingerprint**

In `syncWithNextDNS`, around line 493-508. The managed mode adopts or creates a profile. For the adopt path (line 494), `GetProfile` is already called but the result is discarded with `_`. Change:

```go
_, err := client.GetProfile(ctx, profile.Spec.ProfileID)
```

To:
```go
existingProfile, err := client.GetProfile(ctx, profile.Spec.ProfileID)
```

Then at line 508, change:
```go
profile.Status.Fingerprint = profile.Status.ProfileID + ".dns.nextdns.io"
```

To:
```go
if existingProfile != nil {
    profile.Status.Fingerprint = existingProfile.Fingerprint
}
```

For the create path (new profiles), `CreateProfile` returns only a profile ID, not a Profile struct. After creating, call `GetProfile` to get the fingerprint:

```go
newProfileID, err := client.CreateProfile(ctx, profile.Spec.Name)
if err != nil {
    return fmt.Errorf("failed to create profile: %w", err)
}
profile.Status.ProfileID = newProfileID
// Fetch the newly created profile to get its fingerprint
newProfile, err := client.GetProfile(ctx, newProfileID)
if err != nil {
    logger.Error(err, "Failed to get fingerprint for new profile", "profileID", newProfileID)
} else {
    profile.Status.Fingerprint = newProfile.Fingerprint
}
```

Note: The `existingProfile` variable only exists in the adopt path. For the create path, we need a separate call. Restructure so the fingerprint is set correctly in both paths.

**Step 7: Run test to verify it passes**

Run: `go test ./internal/controller/... -run TestReconcile_ObserveMode_Success -v`
Expected: PASS

**Step 8: Run full test suite**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: All PASS

**Step 9: Commit**

```bash
git add internal/nextdns/mock_client.go internal/controller/nextdnsprofile_controller.go internal/controller/nextdnsprofile_controller_test.go
git commit -m "fix: use real API fingerprint instead of hardcoded DNS endpoint (#74)

The Profile.Fingerprint field from nextdns-go v0.12.0 now provides the
actual API fingerprint. Replaces the constructed '{id}.dns.nextdns.io'
with the authoritative value from GetProfile.

readFullProfile now returns the fingerprint as a second value.
Managed mode fetches fingerprint on both adopt and create paths."
```

---

### Task 3: Close SDK issue and verify

**Step 1: Close the SDK issue**

Run: `gh issue close 35 --repo jacaudi/nextdns-go --comment "Fixed in v0.12.0. Consumed by nextdns-operator in PR fixing #74."`

**Step 2: Run full test suite one more time**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`

---

## Verification

1. `go build ./...` -- compiles with new SDK
2. `go test ./...` -- all pass
3. `grep "nextdns-go" go.mod` -- shows v0.12.0
4. `grep "dns.nextdns.io" internal/controller/nextdnsprofile_controller.go` -- returns 0 matches (no more hardcoded endpoints)
5. Observe mode test: fingerprint comes from API, not constructed
