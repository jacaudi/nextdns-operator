# Cross-Namespace Secret References -- Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow `credentialsRef` to reference a Secret in a different namespace via an optional `namespace` field.

**Architecture:** Add `Namespace` field to `SecretKeySelector`, update `getAPIKey()` to use it, update `findProfilesForSecret()` to match cross-namespace references. Backward compatible -- omitting `namespace` defaults to the CR's namespace.

**Tech Stack:** Go, Kubebuilder, controller-runtime

**Working directory:** `/Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets`

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

### Task 1: Add Namespace field to SecretKeySelector and update getAPIKey

**Files:**
- Modify: `api/v1alpha1/shared_types.go:24-34`
- Modify: `internal/controller/nextdnsprofile_controller.go:289-311`
- Modify: `internal/controller/nextdnsprofile_controller_test.go:189+`

**Step 1: Write failing test for cross-namespace secret retrieval**

Add a new test case to the existing `TestGetAPIKey` table in `internal/controller/nextdnsprofile_controller_test.go` (after line 248). Insert this case into the `tests` slice:

```go
		{
			name: "successful retrieval from different namespace",
			profile: &nextdnsv1alpha1.NextDNSProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile",
					Namespace: "app-namespace",
				},
				Spec: nextdnsv1alpha1.NextDNSProfileSpec{
					Name: "Test Profile",
					CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
						Name:      "shared-secret",
						Namespace: "platform-system",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shared-secret",
					Namespace: "platform-system",
				},
				Data: map[string][]byte{
					"api-key": []byte("cross-ns-api-key"),
				},
			},
			expectError: false,
			expectedKey: "cross-ns-api-key",
		},
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && go test ./internal/controller/... -run TestGetAPIKey -v`
Expected: FAIL -- `Namespace` field doesn't exist on SecretKeySelector

**Step 3: Add Namespace field to SecretKeySelector**

In `api/v1alpha1/shared_types.go`, replace the `SecretKeySelector` struct (lines 24-34):

```go
// SecretKeySelector references a key in a Secret
type SecretKeySelector struct {
	// Name is the name of the Secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the Secret
	// If not set, defaults to the namespace of the referencing resource
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Key is the key within the Secret
	// +kubebuilder:default=api-key
	// +optional
	Key string `json:"key,omitempty"`
}
```

**Step 4: Update getAPIKey to use cross-namespace reference**

In `internal/controller/nextdnsprofile_controller.go`, replace the `getAPIKey` method (lines 289-311):

```go
// getAPIKey retrieves the NextDNS API key from the referenced Secret
func (r *NextDNSProfileReconciler) getAPIKey(ctx context.Context, profile *nextdnsv1alpha1.NextDNSProfile) (string, error) {
	secretName := profile.Spec.CredentialsRef.Name
	secretKey := profile.Spec.CredentialsRef.Key
	if secretKey == "" {
		secretKey = "api-key"
	}

	// Use credentialsRef.namespace if set, otherwise default to the profile's namespace
	secretNamespace := profile.Spec.CredentialsRef.Namespace
	if secretNamespace == "" {
		secretNamespace = profile.Namespace
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: secretNamespace,
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", secretNamespace, secretName, err)
	}

	apiKey, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s/%s", secretKey, secretNamespace, secretName)
	}

	return string(apiKey), nil
}
```

**Step 5: Run test to verify it passes**

Run: `cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && go test ./internal/controller/... -run TestGetAPIKey -v`
Expected: PASS (all cases including cross-namespace)

**Step 6: Commit**

```bash
cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && git add api/v1alpha1/shared_types.go internal/controller/nextdnsprofile_controller.go internal/controller/nextdnsprofile_controller_test.go && git commit -m "feat: add namespace field to SecretKeySelector for cross-namespace secrets (#70)

Allows credentialsRef to reference a Secret in a different namespace.
When namespace is omitted, defaults to the CR's namespace (backward compatible).

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Update findProfilesForSecret for cross-namespace matching

**Files:**
- Modify: `internal/controller/nextdnsprofile_controller.go:1280-1304`
- Modify: `internal/controller/nextdnsprofile_controller_test.go:654+`

**Step 1: Write failing test for cross-namespace secret watch**

Add a new test after the existing `TestFindProfilesForSecret` (around line 654). The test creates a profile in namespace `app` that references a secret in namespace `platform`, and verifies that changing the secret triggers reconciliation of the profile:

```go
func TestFindProfilesForSecret_CrossNamespace(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	// Profile in "app" namespace references secret in "platform" namespace
	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cross-ns-profile",
			Namespace: "app",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Cross NS Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name:      "shared-creds",
				Namespace: "platform",
			},
		},
	}

	// Profile in "platform" namespace with same-namespace ref (should also match)
	localProfile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-profile",
			Namespace: "platform",
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name: "Local Profile",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "shared-creds",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, localProfile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-creds",
			Namespace: "platform",
		},
	}

	requests := reconciler.findProfilesForSecret(ctx, secret)

	// Should find both: cross-ns profile and local profile
	assert.Equal(t, 2, len(requests))

	names := make(map[string]string)
	for _, req := range requests {
		names[req.Name] = req.Namespace
	}
	assert.Equal(t, "app", names["cross-ns-profile"])
	assert.Equal(t, "platform", names["local-profile"])
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && go test ./internal/controller/... -run TestFindProfilesForSecret_CrossNamespace -v`
Expected: FAIL -- only finds localProfile (1 result, not 2), because the current code only lists profiles in the secret's namespace

**Step 3: Update findProfilesForSecret for cross-namespace matching**

In `internal/controller/nextdnsprofile_controller.go`, replace the `findProfilesForSecret` function (lines 1280-1304):

```go
// findProfilesForSecret returns reconcile requests for profiles referencing the secret.
// Matches both same-namespace references (credentialsRef.namespace empty) and
// cross-namespace references (credentialsRef.namespace explicitly set).
func (r *NextDNSProfileReconciler) findProfilesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	// List ALL profiles across all namespaces to catch cross-namespace references
	var profiles nextdnsv1alpha1.NextDNSProfileList
	if err := r.List(ctx, &profiles); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, profile := range profiles.Items {
		ref := profile.Spec.CredentialsRef
		refNamespace := ref.Namespace
		if refNamespace == "" {
			refNamespace = profile.Namespace
		}
		if ref.Name == secret.Name && refNamespace == secret.Namespace {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      profile.Name,
					Namespace: profile.Namespace,
				},
			})
		}
	}
	return requests
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && go test ./internal/controller/... -run TestFindProfilesForSecret -v`
Expected: PASS (all secret tests including cross-namespace)

**Step 5: Run full test suite**

Run: `cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: All PASS

**Step 6: Commit**

```bash
cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && git add internal/controller/nextdnsprofile_controller.go internal/controller/nextdnsprofile_controller_test.go && git commit -m "feat: support cross-namespace secret matching in findProfilesForSecret (#70)

Lists all profiles cluster-wide to catch cross-namespace credentialsRef
references. Both same-namespace and explicit namespace references match.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Regenerate CRDs, update docs, verify

**Files:**
- Regenerate: `api/v1alpha1/zz_generated.deepcopy.go`
- Regenerate: `config/crd/bases/nextdns.io_nextdnsprofiles.yaml`
- Regenerate: `chart/crds/nextdns.io_nextdnsprofiles.yaml`
- Modify: `docs/README.md:574-575`

**Step 1: Regenerate deepcopy and CRDs**

Run: `cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && make generate manifests && make sync-helm-crds`

**Step 2: Verify CRD contains namespace field**

Run: `grep -A5 "credentialsRef:" config/crd/bases/nextdns.io_nextdnsprofiles.yaml | head -15`
Expected: Shows `namespace` field in schema

**Step 3: Update docs**

In `docs/README.md`, find the credentialsRef rows in the spec fields table (lines 574-575). Add the namespace field between name and key:

Change from:
```
| `credentialsRef.name` | string | Yes | | Name of the Secret containing the API key |
| `credentialsRef.key` | string | No | `api-key` | Key within the Secret |
```

To:
```
| `credentialsRef.name` | string | Yes | | Name of the Secret containing the API key |
| `credentialsRef.namespace` | string | No | CR's namespace | Namespace of the Secret (for cross-namespace references) |
| `credentialsRef.key` | string | No | `api-key` | Key within the Secret |
```

**Step 4: Run full test suite**

Run: `cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && make test`
Expected: All PASS

**Step 5: Commit**

```bash
cd /Users/acaudill/Projects/nextdns-operator/.worktrees/cross-ns-secrets && git add api/v1alpha1/zz_generated.deepcopy.go config/crd/ chart/crds/ docs/README.md && git commit -m "chore: regenerate CRDs and update docs for cross-namespace secrets (#70)

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. `grep namespace config/crd/bases/nextdns.io_nextdnsprofiles.yaml | grep -c credentialsRef` -- CRD has namespace in credentialsRef schema
4. Cross-namespace test: profile in namespace A reads secret from namespace B
5. Same-namespace test: omitting namespace defaults to CR's namespace (backward compatible)
6. Secret watch test: changing a secret triggers reconciliation for cross-namespace profiles
