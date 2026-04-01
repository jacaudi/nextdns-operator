# CoreDNS ProfileID Validation -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Prevent CoreDNS from generating an invalid Corefile when the referenced profile has no ProfileID.

**Architecture:** Add a single validation check after profile resolution and readiness check, before Corefile generation. Follows the existing error handling pattern (set condition, update status, requeue).

**Tech Stack:** Go, controller-runtime

**Working directory:** New worktree from `main`

---

### Task 1: Add ProfileID validation with TDD

**Files:**
- Modify: `internal/controller/nextdnscoredns_controller.go:126` (add check after profile resolved)
- Modify: `internal/controller/nextdnscoredns_controller_test.go` (add test)

**Step 1: Write failing test**

Add to `internal/controller/nextdnscoredns_controller_test.go` after `TestNextDNSCoreDNSReconciler_Reconcile_ProfileNotReady`:

```go
func TestNextDNSCoreDNSReconciler_Reconcile_ProfileReadyButNoProfileID(t *testing.T) {
	scheme := newCoreDNSTestScheme()
	ctx := context.Background()

	// Profile is Ready but has no ProfileID (race condition)
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-profile",
			Namespace: "default",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Test Profile",
		},
		Status: nextdnsv1alpha1.NextDNSProfileStatus{
			// ProfileID is empty -- first sync hasn't set it yet
			Conditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Synced",
					Message:            "Profile synced",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-coredns",
			Namespace:  "default",
			Finalizers: []string{CoreDNSFinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{
				Name: "test-profile",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, coreDNS).
		WithStatusSubresource(profile, coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-coredns", Namespace: "default"},
	})
	require.NoError(t, err)

	// Should requeue, not proceed to build Corefile
	assert.True(t, result.RequeueAfter > 0, "Should requeue when ProfileID is empty")

	// Verify condition is set
	updated := &nextdnsv1alpha1.NextDNSCoreDNS{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns", Namespace: "default"}, updated)
	require.NoError(t, err)

	assert.False(t, updated.Status.Ready)

	// Find the Ready condition
	var readyCondition *metav1.Condition
	for i := range updated.Status.Conditions {
		if updated.Status.Conditions[i].Type == ConditionTypeReady {
			readyCondition = &updated.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, readyCondition)
	assert.Equal(t, metav1.ConditionFalse, readyCondition.Status)
	assert.Equal(t, "ProfileNotReady", readyCondition.Reason)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/controller/... -run TestNextDNSCoreDNSReconciler_Reconcile_ProfileReadyButNoProfileID -v`
Expected: FAIL -- the controller proceeds to build the Corefile with empty ProfileID instead of requeueing

**Step 3: Add the validation check**

In `internal/controller/nextdnscoredns_controller.go`, after line 126 (`r.setCondition(coreDNS, ConditionTypeProfileResolved, metav1.ConditionTrue, ...)`), add:

```go
	// Verify profile has a ProfileID (may be empty if first sync hasn't completed)
	if profile.Status.ProfileID == "" {
		logger.Info("Referenced NextDNSProfile has no ProfileID yet", "profile", profile.Name)
		r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "ProfileNotReady",
			"Referenced profile does not have a ProfileID yet")
		coreDNS.Status.Ready = false
		if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/controller/... -run TestNextDNSCoreDNSReconciler_Reconcile_ProfileReadyButNoProfileID -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/controller/nextdnscoredns_controller.go internal/controller/nextdnscoredns_controller_test.go
git commit --no-gpg-sign -m "fix: validate ProfileID before building CoreDNS Corefile (#91)

Adds a check for empty ProfileID after profile resolution. Without this,
a race condition where the profile is Ready but ProfileID hasn't been set
would generate an invalid Corefile with empty upstream URLs.

Closes #91"
```

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. Test: Profile Ready + empty ProfileID -> requeue with condition, no Corefile generated
4. Test: Profile Ready + valid ProfileID -> normal reconciliation (existing tests)
