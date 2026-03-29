# Fix Reconcile Loop from Status Updates -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Prevent the ~2-second reconcile loop caused by unconditional status updates in both observe and managed modes.

**Architecture:** Capture a snapshot of `profile.Status` before reconciliation begins. After reconciliation logic runs, compare the new status against the snapshot using `apiequality.Semantic.DeepEqual`. Only call `Status().Update()` when something actually changed. This prevents the watch event -> re-queue cycle.

**Tech Stack:** Go, controller-runtime, `k8s.io/apimachinery/pkg/api/equality`

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

### Task 1: Fix observe mode -- skip status update when nothing changed

**Files:**
- Modify: `internal/controller/nextdnsprofile_controller.go`
- Modify: `internal/controller/nextdnsprofile_controller_test.go`

**Step 1: Write failing test**

Add a test that reconciles the same profile twice in observe mode with identical API data and verifies that the second reconciliation does NOT trigger a status update. The test should:

1. Set up a profile in observe mode with mock data
2. Reconcile once (status gets populated)
3. Read the status back, note `LastSyncTime`
4. Reconcile again with identical mock data
5. Read the status back again
6. Assert `LastSyncTime` has NOT changed (same value as before)

```go
func TestReconcile_ObserveMode_SkipsUpdateWhenUnchanged(t *testing.T) {
    // ... setup profile, secret, mock with fixed data ...

    // First reconcile - populates status
    _, err := reconciler.Reconcile(ctx, req)
    require.NoError(t, err)

    first := &nextdnsv1alpha1.NextDNSProfile{}
    err = fakeClient.Get(ctx, nn, first)
    require.NoError(t, err)
    firstSyncTime := first.Status.LastSyncTime
    require.NotNil(t, firstSyncTime)

    // Second reconcile - nothing changed in API
    _, err = reconciler.Reconcile(ctx, req)
    require.NoError(t, err)

    second := &nextdnsv1alpha1.NextDNSProfile{}
    err = fakeClient.Get(ctx, nn, second)
    require.NoError(t, err)

    // LastSyncTime should NOT have changed
    assert.Equal(t, firstSyncTime, second.Status.LastSyncTime,
        "LastSyncTime should not change when observed data is unchanged")
}
```

**Step 2: Run test, verify it fails**

Run: `go test ./internal/controller/... -run TestReconcile_ObserveMode_SkipsUpdateWhenUnchanged -v`
Expected: FAIL -- LastSyncTime changes every reconcile

**Step 3: Implement the fix for observe mode**

In `internal/controller/nextdnsprofile_controller.go`, in `reconcileObserveMode` (around line 740-754):

Currently:
```go
// Update status
profile.Status.ProfileID = profile.Spec.ProfileID
profile.Status.Fingerprint = fingerprint
profile.Status.ObservedConfig = observed
profile.Status.SuggestedSpec = buildSuggestedSpec(observed)
now := metav1.Now()
profile.Status.LastSyncTime = &now
profile.Status.ObservedGeneration = profile.Generation

r.setCondition(...)
r.setCondition(...)
r.setCondition(...)

if err := r.Status().Update(ctx, profile); err != nil {
```

Change to:
```go
// Build the new status without changing LastSyncTime yet
newObserved := observed
newSuggested := buildSuggestedSpec(observed)

// Check if the observed data actually changed
observedChanged := !apiequality.Semantic.DeepEqual(profile.Status.ObservedConfig, newObserved) ||
    !apiequality.Semantic.DeepEqual(profile.Status.SuggestedSpec, newSuggested) ||
    profile.Status.ProfileID != profile.Spec.ProfileID ||
    profile.Status.Fingerprint != fingerprint

// Update status fields
profile.Status.ProfileID = profile.Spec.ProfileID
profile.Status.Fingerprint = fingerprint
profile.Status.ObservedConfig = newObserved
profile.Status.SuggestedSpec = newSuggested
profile.Status.ObservedGeneration = profile.Generation

r.setCondition(profile, ConditionTypeObserveOnly, metav1.ConditionTrue, "ObserveMode", "Profile is in observe-only mode")
r.setCondition(profile, ConditionTypeSynced, metav1.ConditionTrue, "ObserveSuccess", "Remote profile read successfully")
r.setCondition(profile, ConditionTypeReady, metav1.ConditionTrue, "Observed", "Profile observed successfully")

// Only update LastSyncTime and write status if something actually changed
if observedChanged {
    now := metav1.Now()
    profile.Status.LastSyncTime = &now

    metrics.RecordProfileSync(profile.Name, profile.Namespace)

    if err := r.Status().Update(ctx, profile); err != nil {
        logger.Error(err, "Failed to update status")
        return ctrl.Result{}, err
    }

    logger.Info("Successfully observed NextDNS profile",
        "profileID", profile.Spec.ProfileID,
        "profileName", observed.Name)
} else {
    logger.V(1).Info("Observed profile unchanged, skipping status update",
        "profileID", profile.Spec.ProfileID)
}

syncInterval := CalculateSyncInterval(r.SyncPeriod)
return ctrl.Result{RequeueAfter: syncInterval}, nil
```

Add the import for `apiequality`:
```go
apiequality "k8s.io/apimachinery/pkg/api/equality"
```

**Step 4: Run test, verify it passes**

Run: `go test ./internal/controller/... -run TestReconcile_ObserveMode_SkipsUpdateWhenUnchanged -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/controller/
git commit --no-gpg-sign -m "fix: skip status update in observe mode when data is unchanged (#87)

Compare observedConfig before and after reading from the API.
Only update LastSyncTime and write status when something actually changed.
Prevents the watch event -> re-queue cycle that caused ~2-second reconcile loops."
```

---

### Task 2: Fix managed mode -- skip status update when nothing changed

**Files:**
- Modify: `internal/controller/nextdnsprofile_controller.go`
- Modify: `internal/controller/nextdnsprofile_controller_test.go`

**Step 1: Write failing test**

Similar to observe mode -- reconcile twice with identical spec and verify LastSyncTime doesn't change on the second run. Note: managed mode always syncs to the API (pushes spec), but the status update should be skippable when the status fields haven't changed.

Actually, in managed mode the sync to the API is idempotent, and the status fields (conditions, counts, etc.) should be identical on re-reconciliation. The fix here is the same pattern: capture status before, compare after, skip update if unchanged.

```go
func TestReconcile_ManagedMode_SkipsUpdateWhenUnchanged(t *testing.T) {
    // ... setup profile with security settings, secret, mock ...

    // First reconcile
    _, err := reconciler.Reconcile(ctx, req)
    require.NoError(t, err)

    first := &nextdnsv1alpha1.NextDNSProfile{}
    err = fakeClient.Get(ctx, nn, first)
    require.NoError(t, err)
    firstSyncTime := first.Status.LastSyncTime

    // Second reconcile - same spec, same API state
    _, err = reconciler.Reconcile(ctx, req)
    require.NoError(t, err)

    second := &nextdnsv1alpha1.NextDNSProfile{}
    err = fakeClient.Get(ctx, nn, second)
    require.NoError(t, err)

    assert.Equal(t, firstSyncTime, second.Status.LastSyncTime)
}
```

**Step 2: Run test, verify it fails**

**Step 3: Implement the fix for managed mode**

In the managed mode reconcile path (around line 222-245), apply the same pattern:

```go
// Capture status before changes
statusBefore := profile.Status.DeepCopy()

// ... existing status updates (AggregatedCounts, ReferencedResources, conditions) ...

// Check if status actually changed (excluding LastSyncTime which we haven't set yet)
statusChanged := !apiequality.Semantic.DeepEqual(statusBefore.AggregatedCounts, profile.Status.AggregatedCounts) ||
    !apiequality.Semantic.DeepEqual(statusBefore.ReferencedResources, profile.Status.ReferencedResources) ||
    !apiequality.Semantic.DeepEqual(statusBefore.Conditions, profile.Status.Conditions) ||
    statusBefore.ProfileID != profile.Status.ProfileID ||
    statusBefore.Fingerprint != profile.Status.Fingerprint

if statusChanged {
    now := metav1.Now()
    profile.Status.LastSyncTime = &now
}

// Always update on first sync (no LastSyncTime yet)
if statusChanged || profile.Status.LastSyncTime == nil {
    if err := r.Status().Update(ctx, profile); err != nil {
        logger.Error(err, "Failed to update status")
        return ctrl.Result{}, err
    }
}
```

**Step 4: Run test, verify it passes**

**Step 5: Run full test suite**

**Step 6: Commit**

```bash
git add internal/controller/
git commit --no-gpg-sign -m "fix: skip status update in managed mode when data is unchanged (#87)

Same pattern as observe mode -- compare status before and after,
only write when something changed. Prevents watch-triggered re-queues."
```

---

### Task 3: Verify and clean up

**Step 1: Run full test suite**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`

**Step 2: Verify no regressions -- existing observe and managed tests still pass**

Run: `go test ./internal/controller/... -run "TestReconcile_ObserveMode|TestSyncWithNextDNS" -v`

**Step 3: Commit any cleanups needed**

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. Observe mode test: second reconcile with same data does NOT update LastSyncTime
4. Managed mode test: second reconcile with same spec does NOT update LastSyncTime
5. First reconcile always updates (LastSyncTime was nil)
6. Changed data triggers update (e.g., new field in observedConfig)
