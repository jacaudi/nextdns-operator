# Remove ConfigMap Import Feature -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the deprecated ConfigMap Import feature (`configImportRef`) from the operator.

**Architecture:** Pure deletion in dependency order -- remove consumers (controller) before producers (types/package). The ConfigMap watch stays because `configMapRef` (export) still uses owner references.

**Tech Stack:** Go, Kubebuilder, controller-runtime

> **For Claude:** REQUIRED SUB-SKILLS (must use in order):
> 1. `superpowers:using-git-worktrees` -- Isolate work in a dedicated worktree
> 2. Choose execution mode:
>    - **Subagent-Driven (this session):** `superpowers:subagent-driven-development`
>    - **Parallel Session (separate):** `superpowers:executing-plans`
> 3. `superpowers:verification-before-completion` -- Verify all tests pass before claiming done
> 4. `superpowers:requesting-code-review` -- Code review after EACH task
> 5. After ALL tasks: dispatch independent and comprehensive code review on full diff
> 6. `superpowers:finishing-a-development-branch` -- Complete the branch

---

### Task 1: Remove ConfigMap Import logic from controller

**Files:**
- Modify: `internal/controller/nextdnsprofile_controller.go`

**Step 1: Remove the `configimport` package import**

Remove line 24: `"github.com/jacaudi/nextdns-operator/internal/configimport"`

**Step 2: Remove `ConditionTypeConfigImported` constant**

Remove lines 42-43:
```go
// ConditionTypeConfigImported indicates whether config import from ConfigMap succeeded
ConditionTypeConfigImported = "ConfigImported"
```

**Step 3: Remove the deprecation warning block**

Remove lines 135-138 (the `if profile.Spec.ConfigImportRef != nil` block that logs the deprecation warning).

**Step 4: Remove the import logic block**

Remove the entire `if profile.Spec.ConfigImportRef != nil` block at lines 180-211. This includes:
- `configimport.ReadAndParse` call
- Error handling and condition setting
- `configimport.MergeIntoSpec` call
- `ConfigImportResourceVersion` update
- Success condition and logging

**Step 5: Remove configImportRef section from `findProfilesForConfigMap`**

In `findProfilesForConfigMap` (line 1349), remove lines 1372-1390 (the "Check configImportRef" section). Keep lines 1349-1370 (owner reference check for configMapRef export). The function becomes simpler:

```go
func (r *NextDNSProfileReconciler) findProfilesForConfigMap(ctx context.Context, obj client.Object) []reconcile.Request {
	configMap, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}

	var requests []reconcile.Request

	// Check owner references (output ConfigMap from configMapRef)
	for _, ref := range configMap.OwnerReferences {
		if ref.Kind == "NextDNSProfile" && ref.APIVersion == nextdnsv1alpha1.GroupVersion.String() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      ref.Name,
					Namespace: configMap.Namespace,
				},
			})
		}
	}

	return requests
}
```

**Step 6: Verify build**

Run: `go build ./...`
Expected: PASS

**Step 7: Remove ConfigImport-related test assertions and test functions**

In `internal/controller/nextdnsprofile_controller_test.go`:
- Remove the `ConditionTypeConfigImported` assertion at line 866
- Remove `TestFindProfilesForImportConfigMap` test (line 2469) -- this tests the configImportRef path. Keep any test that only tests the owner-reference (configMapRef) path.

**Step 8: Run tests**

Run: `go test ./internal/controller/... -v 2>&1 | tail -5`
Expected: PASS

**Step 9: Commit**

```bash
git add internal/controller/
git commit -m "feat: remove ConfigMap Import logic from controller (#68)

Remove configimport package usage, ConditionTypeConfigImported,
deprecation warning, import logic block, and configImportRef
section from findProfilesForConfigMap. ConfigMap watch retained
for configMapRef (export) owner reference tracking."
```

---

### Task 2: Remove ConfigImportRef from API types

**Files:**
- Modify: `api/v1alpha1/nextdnsprofile_types.go`

**Step 1: Remove `ConfigImportRef` struct**

Remove the struct definition (around lines 32-43):
```go
// ConfigImportRef references a ConfigMap containing profile configuration JSON
type ConfigImportRef struct {
	Name string `json:"name"`
	Key  string `json:"key,omitempty"`
}
```

**Step 2: Remove `ConfigImportRef` field from `NextDNSProfileSpec`**

Remove the field (around line 136):
```go
ConfigImportRef *ConfigImportRef `json:"configImportRef,omitempty"`
```

**Step 3: Remove `ConfigImportResourceVersion` from `NextDNSProfileStatus`**

Remove the field (around line 422):
```go
ConfigImportResourceVersion string `json:"configImportResourceVersion,omitempty"`
```

**Step 4: Verify build**

Run: `go build ./...`
Expected: PASS (deepcopy will be stale but types compile)

**Step 5: Commit**

```bash
git add api/v1alpha1/nextdnsprofile_types.go
git commit -m "feat: remove ConfigImportRef type and fields from CRD spec/status (#68)"
```

---

### Task 3: Delete `internal/configimport/` package

**Files:**
- Delete: `internal/configimport/` (entire directory)

**Step 1: Delete the directory**

```bash
rm -rf internal/configimport/
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: PASS

**Step 3: Run full test suite**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: All PASS (configimport tests gone with the directory)

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: delete internal/configimport package (#68)

Removes reader, merge, validate, types and all associated tests."
```

---

### Task 4: Update documentation and samples

**Files:**
- Modify: `docs/README.md`
- Modify: `config/samples/nextdns_v1alpha1_nextdnsprofile.yaml`

**Step 1: Remove ConfigMap Import section from docs**

In `docs/README.md`, remove the entire ConfigMap Import section (starts at "### ConfigMap Import" with the deprecation notice, through all the import examples and field tables).

**Step 2: Remove configImportRef from field reference tables**

Remove any rows referencing `configImportRef`, `ConfigImportResourceVersion`, or `ConfigImported` condition from the field tables.

**Step 3: Remove configImport troubleshooting**

Remove any troubleshooting entries related to ConfigMap Import.

**Step 4: Clean up sample manifest**

In `config/samples/nextdns_v1alpha1_nextdnsprofile.yaml`, remove the commented-out `configImportRef` example.

**Step 5: Commit**

```bash
git add docs/README.md config/samples/
git commit -m "docs: remove ConfigMap Import documentation and examples (#68)"
```

---

### Task 5: Regenerate deepcopy, CRDs, sync Helm chart

**Step 1: Regenerate**

```bash
make generate manifests
make sync-helm-crds
```

**Step 2: Run full test suite**

```bash
make test
```

**Step 3: Verify configImportRef is gone from CRDs**

```bash
grep -c configImport config/crd/bases/nextdns.io_nextdnsprofiles.yaml
```
Expected: 0

**Step 4: Commit**

```bash
git add api/v1alpha1/zz_generated.deepcopy.go config/crd/ chart/crds/
git commit -m "chore: regenerate CRDs after ConfigMap Import removal (#68)"
```

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. `grep -r configImport internal/` -- returns nothing
4. `grep -r ConfigImport internal/` -- returns nothing
5. `grep configImportRef config/crd/bases/nextdns.io_nextdnsprofiles.yaml` -- returns nothing
6. `grep configImportRef docs/README.md` -- returns nothing
