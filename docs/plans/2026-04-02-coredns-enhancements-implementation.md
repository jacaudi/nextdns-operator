# CoreDNS Enhancements -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add deviceName warning for plain DNS, remove ServiceMonitor from CRD, and add PDB support to CoreDNS.

**Architecture:** Three independent changes in one branch. Each is a separate task with its own commit. CRDs regenerated once at the end.

**Tech Stack:** Go, Kubebuilder, controller-runtime

**Working directory:** New worktree from `main`

> **For Claude:** REQUIRED SUB-SKILLS (must use in order):
> 1. `superpowers:using-git-worktrees`
> 2. `superpowers:subagent-driven-development` + `superpowers:test-driven-development`
> 3. `superpowers:verification-before-completion`
> 4. `superpowers:requesting-code-review` after EACH task
> 5. Final comprehensive code review on full diff
> 6. `superpowers:finishing-a-development-branch`

---

### Task 1: Warn when deviceName used with plain DNS (#93)

**Files:**
- Modify: `internal/controller/nextdnscoredns_controller.go` (add warning after profile resolution)
- Modify: `internal/controller/nextdnscoredns_controller_test.go` (add test)

**Step 1: Write failing test**

Add test that creates a CoreDNS CR with `protocol: dns` and `deviceName: "MyDevice"`, reconciles, and asserts a `DeviceNameIgnored` warning condition is set on the CR status.

```go
func TestNextDNSCoreDNSReconciler_Reconcile_DeviceNameIgnoredWithPlainDNS(t *testing.T) {
    // Setup profile (Ready, ProfileID set)
    // Setup CoreDNS with protocol: dns, deviceName: "MyDevice"
    // Reconcile
    // Assert: condition DeviceNameIgnored = True
    // Assert: reconcile still succeeds (doesn't block)
}
```

**Step 2: Run test, verify fails**

**Step 3: Add warning logic**

In `nextdnscoredns_controller.go`, add a new condition constant:
```go
ConditionTypeDeviceNameIgnored = "DeviceNameIgnored"
```

After the Multus validation block (around line 170, before resource reconciliation), add:
```go
// Warn if deviceName is used with plain DNS protocol
if coreDNS.Spec.Upstream != nil &&
    coreDNS.Spec.Upstream.Protocol == nextdnsv1alpha1.ProtocolDNS &&
    coreDNS.Spec.Upstream.DeviceName != "" {
    logger.Info("WARNING: deviceName is ignored with plain DNS protocol; use DoT or DoH for device identification")
    r.setCondition(coreDNS, ConditionTypeDeviceNameIgnored, metav1.ConditionTrue, "ProtocolLimitation",
        "deviceName is ignored with plain DNS protocol; use DoT or DoH for device identification")
} else {
    r.setCondition(coreDNS, ConditionTypeDeviceNameIgnored, metav1.ConditionFalse, "NotApplicable",
        "deviceName is not used with plain DNS or is not set")
}
```

Wait -- setting a condition to False when not applicable is noisy. Better to only set the condition when there IS a warning, and remove it otherwise. Actually, simplest: only set the True condition, don't touch it when not applicable.

Revised: Only set the condition when the warning applies. Don't clear it (if user fixes the config, the next reconcile won't set it, and the condition will remain stale -- but this is acceptable for a warning).

Actually, best practice: always set it. True when warning applies, False when not. This keeps status clean.

**Step 4: Run test, verify passes**
**Step 5: Run full suite**
**Step 6: Commit**

```bash
git commit --no-gpg-sign -m "feat: warn when deviceName is used with plain DNS protocol (#93)

Sets DeviceNameIgnored condition when spec.upstream.protocol is dns
and spec.upstream.deviceName is set. Device identification only works
with DoT (SNI) and DoH (URL path).

Closes #93"
```

---

### Task 2: Remove ServiceMonitor from CRD (#94)

**Files:**
- Modify: `api/v1alpha1/nextdnscoredns_types.go` (remove ServiceMonitorConfig struct and field)
- Modify: `internal/controller/nextdnscoredns_controller_test.go` (remove any ServiceMonitor references in tests)
- Modify: `docs/README.md` (update metrics docs)

**Step 1: Remove types**

In `api/v1alpha1/nextdnscoredns_types.go`:
- Delete `ServiceMonitorConfig` struct (lines 132-155)
- Remove `ServiceMonitor *ServiceMonitorConfig` field from `CoreDNSMetricsConfig` (lines 164-166)

**Step 2: Check for compilation errors**

Run: `go build ./...`
Fix any references to the removed types.

**Step 3: Update docs**

In `docs/README.md`, remove ServiceMonitor field references from the CoreDNS field table. Add a note that ServiceMonitor is configured via Helm values.

**Step 4: Run full suite**
**Step 5: Commit**

```bash
git commit --no-gpg-sign -m "chore: remove ServiceMonitor from CoreDNS CRD (#94)

ServiceMonitor was defined in the CRD but never reconciled by the
controller. It was only functional via the Helm chart's bjw-s library.
Removing from CRD to avoid user confusion. Configure ServiceMonitor
through Helm values instead.

Closes #94"
```

---

### Task 3: Add PodDisruptionBudget support (#95)

**Files:**
- Modify: `api/v1alpha1/nextdnscoredns_types.go` (add PDB config type and field)
- Modify: `internal/controller/nextdnscoredns_controller.go` (add PDB reconciliation + RBAC)
- Modify: `internal/controller/nextdnscoredns_controller_test.go` (add PDB test)

**Step 1: Write failing test**

Add test that creates a CoreDNS CR with `deployment.replicas: 2` and `deployment.podDisruptionBudget.maxUnavailable: 1`, reconciles, and asserts a PDB resource was created with the correct maxUnavailable value.

**Step 2: Run test, verify fails**

**Step 3: Add PDB types**

In `api/v1alpha1/nextdnscoredns_types.go`, add after `CoreDNSDeploymentConfig`:

```go
// CoreDNSPDBConfig configures PodDisruptionBudget for CoreDNS
type CoreDNSPDBConfig struct {
    // MinAvailable is the minimum number of pods that must be available
    // Mutually exclusive with MaxUnavailable
    // +optional
    MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`

    // MaxUnavailable is the maximum number of pods that can be unavailable
    // Mutually exclusive with MinAvailable. Defaults to 1 if neither is set.
    // +optional
    MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}
```

Add to `CoreDNSDeploymentConfig`:
```go
// PodDisruptionBudget configures disruption budget for HA deployments
// +optional
PodDisruptionBudget *CoreDNSPDBConfig `json:"podDisruptionBudget,omitempty"`
```

Remove the TODO comment at line 68.

Add import for `intstr`:
```go
"k8s.io/apimachinery/pkg/util/intstr"
```

**Step 4: Add RBAC marker**

In `nextdnscoredns_controller.go`, add:
```go
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
```

**Step 5: Add PDB reconciliation**

Add `reconcilePDB` method to the controller. Call it from `reconcileWorkload` or directly from `Reconcile` after workload reconciliation:

```go
func (r *NextDNSCoreDNSReconciler) reconcilePDB(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, profile *nextdnsv1alpha1.NextDNSProfile) error {
    // Skip if PDB not configured
    if coreDNS.Spec.Deployment == nil || coreDNS.Spec.Deployment.PodDisruptionBudget == nil {
        return nil
    }

    // Skip for DaemonSet mode (PDB not meaningful)
    if coreDNS.Spec.Deployment.Mode == nextdnsv1alpha1.DeploymentModeDaemonSet {
        return nil
    }

    // Build PDB spec
    pdbConfig := coreDNS.Spec.Deployment.PodDisruptionBudget
    pdb := &policyv1.PodDisruptionBudget{
        ObjectMeta: metav1.ObjectMeta{
            Name:      r.getResourceName(coreDNS, profile) + "-pdb",
            Namespace: coreDNS.Namespace,
        },
        Spec: policyv1.PodDisruptionBudgetSpec{
            Selector: &metav1.LabelSelector{
                MatchLabels: r.getLabels(coreDNS, profile),
            },
        },
    }

    // Set min/max
    if pdbConfig.MinAvailable != nil {
        pdb.Spec.MinAvailable = pdbConfig.MinAvailable
    } else if pdbConfig.MaxUnavailable != nil {
        pdb.Spec.MaxUnavailable = pdbConfig.MaxUnavailable
    } else {
        // Default: maxUnavailable = 1
        one := intstr.FromInt(1)
        pdb.Spec.MaxUnavailable = &one
    }

    // Set owner reference
    ctrl.SetControllerReference(coreDNS, pdb, r.Scheme)

    // Create or update
    // ... standard create-or-update pattern
}
```

**Step 6: Run test, verify passes**
**Step 7: Run full suite**
**Step 8: Commit**

```bash
git commit --no-gpg-sign -m "feat: add PodDisruptionBudget support for CoreDNS HA (#95)

Creates a PDB when spec.deployment.podDisruptionBudget is configured.
Defaults to maxUnavailable=1 when neither min nor max is specified.
Only applies to Deployment mode (not DaemonSet).

Closes #95"
```

---

### Task 4: Regenerate CRDs, update docs, verify

**Step 1:** `task manifests && task generate && task sync-helm-crds`
**Step 2:** `./hack/generate-helm-rbac.sh` (PDB RBAC added)
**Step 3:** Update `docs/README.md`:
  - Remove ServiceMonitor rows from field tables
  - Add PDB fields to deployment config table
  - Add note about deviceName + plain DNS
  - Add note about ServiceMonitor being Helm-only
**Step 4:** `go test ./...` -- all pass
**Step 5:** Commit

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. DeviceName + plain DNS -> DeviceNameIgnored condition set
4. ServiceMonitor gone from CRD schema
5. PDB created when configured with replicas > 1
6. PDB not created for DaemonSet mode
7. Helm RBAC template includes PDB permissions
