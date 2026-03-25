# Critical Fixes (C1, C2) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix managed mode to sync all spec fields (rewrites, performance, web3, logClientsIPs, logDomains) and remove dead code in SyncDenylist.

**Architecture:** Consolidate `UpdateSettings` to use the top-level `Settings.Update` PATCH endpoint (which accepts logs, block page, performance, and web3 in one call). Add `SyncRewrites` using diff-based create/delete. Wire both into the controller's `syncWithNextDNS`. Remove dead code from `SyncDenylist`.

**Tech Stack:** Go, nextdns-go v0.11.0, controller-runtime

**Working directory:** New worktree from `main`

> **For Claude:** REQUIRED SUB-SKILLS (must use in order):
> 1. `superpowers:using-git-worktrees` -- Isolate work in a dedicated worktree
> 2. Choose execution mode (load `superpowers:test-driven-development` alongside whichever is chosen -- all agents/sessions must use TDD):
>    - **Subagent-Driven (this session):** `superpowers:subagent-driven-development` + `superpowers:test-driven-development`
>    - **Parallel Session (separate):** `superpowers:executing-plans` + `superpowers:test-driven-development`
> 3. `superpowers:verification-before-completion` -- Verify all tests pass before claiming done
> 4. `superpowers:requesting-code-review` -- Code review after EACH task
> 5. After ALL tasks: dispatch independent and comprehensive code review on full diff
> 6. `superpowers:finishing-a-development-branch` -- Complete the branch

---

### Task 1: Remove dead code in SyncDenylist (C2)

**Files:**
- Modify: `internal/nextdns/client.go:222-263`

**Step 1: Write a test verifying SyncDenylist works without pre-fetch**

Add to `internal/nextdns/client_test.go`:

```go
func TestSyncDenylist_NoPrefetch(t *testing.T) {
	// Verify SyncDenylist works by creating entries
	// This test exists to confirm the dead code removal doesn't break functionality
	mockClient := NewMockClient()
	ctx := context.Background()

	entries := []DomainEntry{
		{Domain: "bad.com", Active: true},
		{Domain: "worse.com", Active: false},
	}

	err := mockClient.SyncDenylist(ctx, "test-profile", entries)
	require.NoError(t, err)

	result, err := mockClient.GetDenylist(ctx, "test-profile")
	require.NoError(t, err)
	assert.Equal(t, 2, len(result))
}
```

**Step 2: Run test to verify it passes (test is for existing behavior)**

Run: `go test ./internal/nextdns/... -run TestSyncDenylist_NoPrefetch -v`
Expected: PASS

**Step 3: Remove dead code**

In `internal/nextdns/client.go`, replace the `SyncDenylist` function (lines 222-263) with:

```go
func (c *Client) SyncDenylist(ctx context.Context, profileID string, entries []DomainEntry) error {
	start := time.Now()

	// Build the desired denylist
	var denylist []*nextdns.Denylist
	for _, entry := range entries {
		denylist = append(denylist, &nextdns.Denylist{
			ID:     entry.Domain,
			Active: entry.Active,
		})
	}

	// PUT replaces the entire list
	createRequest := &nextdns.CreateDenylistRequest{
		ProfileID: profileID,
		Denylist:  denylist,
	}
	if err := c.client.Denylist.Create(ctx, createRequest); err != nil {
		metrics.RecordAPIRequest("SyncDenylist", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to sync denylist: %w", err)
	}

	metrics.RecordAPIRequest("SyncDenylist", time.Since(start).Seconds(), true)
	return nil
}
```

**Step 4: Run test to verify it still passes**

Run: `go test ./internal/nextdns/... -run TestSyncDenylist -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/nextdns/client.go internal/nextdns/client_test.go
git commit -m "fix: remove dead code in SyncDenylist (C2)

The currentDomains map and pre-fetch LIST call were unused.
SyncDenylist uses a PUT to replace the entire list, matching SyncAllowlist."
```

---

### Task 2: Extend SettingsConfig and UpdateSettings for performance, web3, log drop fields

**Files:**
- Modify: `internal/nextdns/client.go:73-81` (SettingsConfig struct)
- Modify: `internal/nextdns/client.go:439-477` (UpdateSettings method)
- Modify: `internal/nextdns/interface.go` (no interface change needed, UpdateSettings signature unchanged)

**Step 1: Write failing test for UpdateSettings with performance and web3**

Add to `internal/nextdns/client_test.go`:

```go
func TestUpdateSettings_FullConfig(t *testing.T) {
	mockClient := NewMockClient()
	ctx := context.Background()

	// Set up a profile
	mockClient.SetProfile("test-profile", "Test", "test.dns.nextdns.io")

	config := &SettingsConfig{
		LogsEnabled:   true,
		LogClientsIPs: true,
		LogDomains:    true,
		LogRetention:  30,
		BlockPageEnable: true,
		Web3:          true,
		// Performance fields
		Ecs:             true,
		CacheBoost:      true,
		CnameFlattening: false,
	}

	err := mockClient.UpdateSettings(ctx, "test-profile", config)
	require.NoError(t, err)

	// Verify settings were stored
	settings := mockClient.Settings["test-profile"]
	require.NotNil(t, settings)
	assert.True(t, settings.Web3)
	require.NotNil(t, settings.Performance)
	assert.True(t, settings.Performance.Ecs)
	assert.True(t, settings.Performance.CacheBoost)
	assert.False(t, settings.Performance.CnameFlattening)
	require.NotNil(t, settings.Logs)
	assert.True(t, settings.Logs.Enabled)
	assert.Equal(t, 30, settings.Logs.Retention)
	// LogClientsIPs=true means Drop.IP=false (inverted)
	require.NotNil(t, settings.Logs.Drop)
	assert.False(t, settings.Logs.Drop.IP)
	assert.False(t, settings.Logs.Drop.Domain)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/nextdns/... -run TestUpdateSettings_FullConfig -v`
Expected: FAIL -- `Ecs`, `CacheBoost`, `CnameFlattening` fields don't exist on SettingsConfig

**Step 3: Extend SettingsConfig with performance fields**

Replace `SettingsConfig` in `internal/nextdns/client.go` (lines 73-81):

```go
// SettingsConfig represents general settings
type SettingsConfig struct {
	LogsEnabled     bool
	LogClientsIPs   bool
	LogDomains      bool
	LogRetention    int
	BlockPageEnable bool
	Web3            bool
	// Performance settings
	Ecs             bool
	CacheBoost      bool
	CnameFlattening bool
}
```

**Step 4: Rewrite UpdateSettings to use single Settings.Update PATCH**

Replace the `UpdateSettings` method in `internal/nextdns/client.go` (lines 439-477):

```go
func (c *Client) UpdateSettings(ctx context.Context, profileID string, config *SettingsConfig) error {
	if config == nil {
		return nil
	}

	start := time.Now()

	// Build the full settings object for a single PATCH call
	// Note: LogClientsIPs and LogDomains use positive logic in the spec
	// (true = log them), but the API uses inverted logic via Drop
	// (true = don't log them). We invert here.
	settings := &nextdns.Settings{
		Logs: &nextdns.SettingsLogs{
			Enabled:   config.LogsEnabled,
			Retention: config.LogRetention,
			Drop: &nextdns.SettingsLogsDrop{
				IP:     !config.LogClientsIPs,
				Domain: !config.LogDomains,
			},
		},
		BlockPage: &nextdns.SettingsBlockPage{
			Enabled: config.BlockPageEnable,
		},
		Performance: &nextdns.SettingsPerformance{
			Ecs:             config.Ecs,
			CacheBoost:      config.CacheBoost,
			CnameFlattening: config.CnameFlattening,
		},
		Web3: config.Web3,
	}

	request := &nextdns.UpdateSettingsRequest{
		ProfileID: profileID,
		Settings:  settings,
	}

	err := c.client.Settings.Update(ctx, request)
	metrics.RecordAPIRequest("UpdateSettings", time.Since(start).Seconds(), err == nil)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}
```

**Step 5: Update MockClient.UpdateSettings to store all fields**

In `internal/nextdns/mock_client.go`, find the `UpdateSettings` method and update it to store performance, web3, and log drop fields in the `Settings` map:

```go
func (m *MockClient) UpdateSettings(ctx context.Context, profileID string, config *SettingsConfig) error {
	m.recordCall("UpdateSettings")
	if m.UpdateSettingsError != nil {
		return m.UpdateSettingsError
	}

	m.Settings[profileID] = &sdknextdns.Settings{
		Logs: &sdknextdns.SettingsLogs{
			Enabled:   config.LogsEnabled,
			Retention: config.LogRetention,
			Drop: &sdknextdns.SettingsLogsDrop{
				IP:     !config.LogClientsIPs,
				Domain: !config.LogDomains,
			},
		},
		BlockPage: &sdknextdns.SettingsBlockPage{
			Enabled: config.BlockPageEnable,
		},
		Performance: &sdknextdns.SettingsPerformance{
			Ecs:             config.Ecs,
			CacheBoost:      config.CacheBoost,
			CnameFlattening: config.CnameFlattening,
		},
		Web3: config.Web3,
	}
	return nil
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/nextdns/... -run TestUpdateSettings_FullConfig -v`
Expected: PASS

**Step 7: Run all tests**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: all PASS

**Step 8: Commit**

```bash
git add internal/nextdns/client.go internal/nextdns/mock_client.go internal/nextdns/client_test.go
git commit -m "feat: extend UpdateSettings to sync performance, web3, and log drop fields (C1)

Consolidates settings sync into a single Settings.Update PATCH call.
LogClientsIPs/LogDomains use positive logic in spec but API uses
inverted Drop struct -- inversion is done in the client layer."
```

---

### Task 3: Add SyncRewrites to client

**Files:**
- Modify: `internal/nextdns/interface.go` (add SyncRewrites to interface)
- Modify: `internal/nextdns/client.go` (add SyncRewrites implementation)
- Modify: `internal/nextdns/mock_client.go` (add SyncRewrites mock)

**Step 1: Write failing test for SyncRewrites**

Add to `internal/nextdns/client_test.go`:

```go
func TestSyncRewrites(t *testing.T) {
	mockClient := NewMockClient()
	ctx := context.Background()

	// Set up existing rewrites
	mockClient.Rewrites["test-profile"] = []*sdknextdns.Rewrites{
		{ID: "rw1", Name: "old.example.com", Content: "1.2.3.4"},
		{ID: "rw2", Name: "keep.example.com", Content: "5.6.7.8"},
	}

	// Desired state: keep one, add one, remove one
	desired := []RewriteEntry{
		{Name: "keep.example.com", Content: "5.6.7.8"},
		{Name: "new.example.com", Content: "9.10.11.12"},
	}

	err := mockClient.SyncRewrites(ctx, "test-profile", desired)
	require.NoError(t, err)

	result, err := mockClient.GetRewrites(ctx, "test-profile")
	require.NoError(t, err)
	assert.Equal(t, 2, len(result))

	// Verify the correct rewrites exist
	names := make(map[string]string)
	for _, rw := range result {
		names[rw.Name] = rw.Content
	}
	assert.Equal(t, "5.6.7.8", names["keep.example.com"])
	assert.Equal(t, "9.10.11.12", names["new.example.com"])
	assert.Empty(t, names["old.example.com"])
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/nextdns/... -run TestSyncRewrites -v`
Expected: FAIL -- SyncRewrites undefined

**Step 3: Add RewriteEntry type and SyncRewrites to interface**

In `internal/nextdns/client.go`, add the `RewriteEntry` type near the other config types:

```go
// RewriteEntry represents a DNS rewrite for syncing to NextDNS
type RewriteEntry struct {
	Name    string
	Content string
}
```

In `internal/nextdns/interface.go`, add to `ClientInterface`:

```go
	// Rewrite operations
	SyncRewrites(ctx context.Context, profileID string, entries []RewriteEntry) error
```

**Step 4: Implement SyncRewrites in client.go**

Add to `internal/nextdns/client.go`:

```go
// SyncRewrites synchronizes DNS rewrites for a profile using diff-based create/delete.
// The NextDNS API does not support update for rewrites, so we delete removed entries
// and create new ones.
func (c *Client) SyncRewrites(ctx context.Context, profileID string, entries []RewriteEntry) error {
	start := time.Now()

	// Get current rewrites
	listRequest := &nextdns.ListRewritesRequest{
		ProfileID: profileID,
	}
	current, err := c.client.Rewrites.List(ctx, listRequest)
	if err != nil {
		metrics.RecordAPIRequest("SyncRewrites", time.Since(start).Seconds(), false)
		return fmt.Errorf("failed to list rewrites: %w", err)
	}

	// Build desired set keyed by name+content
	type rewriteKey struct{ Name, Content string }
	desired := make(map[rewriteKey]bool, len(entries))
	for _, e := range entries {
		desired[rewriteKey{e.Name, e.Content}] = true
	}

	// Build current set and find entries to delete
	currentSet := make(map[rewriteKey]bool, len(current))
	for _, rw := range current {
		key := rewriteKey{rw.Name, rw.Content}
		currentSet[key] = true
		if !desired[key] {
			// Delete entries not in desired state
			deleteReq := &nextdns.DeleteRewritesRequest{
				ProfileID: profileID,
				ID:        rw.ID,
			}
			if err := c.client.Rewrites.Delete(ctx, deleteReq); err != nil {
				metrics.RecordAPIRequest("SyncRewrites", time.Since(start).Seconds(), false)
				return fmt.Errorf("failed to delete rewrite %s: %w", rw.Name, err)
			}
		}
	}

	// Create entries not in current state
	for _, e := range entries {
		key := rewriteKey{e.Name, e.Content}
		if !currentSet[key] {
			createReq := &nextdns.CreateRewritesRequest{
				ProfileID: profileID,
				Rewrites: &nextdns.Rewrites{
					Name:    e.Name,
					Content: e.Content,
				},
			}
			if _, err := c.client.Rewrites.Create(ctx, createReq); err != nil {
				metrics.RecordAPIRequest("SyncRewrites", time.Since(start).Seconds(), false)
				return fmt.Errorf("failed to create rewrite %s: %w", e.Name, err)
			}
		}
	}

	metrics.RecordAPIRequest("SyncRewrites", time.Since(start).Seconds(), true)
	return nil
}
```

**Step 5: Add SyncRewrites to MockClient**

In `internal/nextdns/mock_client.go`, add:

```go
func (m *MockClient) SyncRewrites(ctx context.Context, profileID string, entries []RewriteEntry) error {
	m.recordCall("SyncRewrites")
	if m.SyncRewritesError != nil {
		return m.SyncRewritesError
	}

	// Build desired set
	type rewriteKey struct{ Name, Content string }
	desired := make(map[rewriteKey]bool, len(entries))
	for _, e := range entries {
		desired[rewriteKey{e.Name, e.Content}] = true
	}

	// Filter current to keep only desired, track what exists
	currentSet := make(map[rewriteKey]bool)
	var kept []*sdknextdns.Rewrites
	for _, rw := range m.Rewrites[profileID] {
		key := rewriteKey{rw.Name, rw.Content}
		if desired[key] {
			kept = append(kept, rw)
			currentSet[key] = true
		}
	}

	// Add new entries
	for _, e := range entries {
		key := rewriteKey{e.Name, e.Content}
		if !currentSet[key] {
			kept = append(kept, &sdknextdns.Rewrites{
				ID:      fmt.Sprintf("rw-%s", e.Name),
				Name:    e.Name,
				Content: e.Content,
			})
		}
	}

	m.Rewrites[profileID] = kept
	return nil
}
```

Also add `SyncRewritesError error` field to the MockClient struct.

**Step 6: Run test to verify it passes**

Run: `go test ./internal/nextdns/... -run TestSyncRewrites -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/nextdns/client.go internal/nextdns/interface.go internal/nextdns/mock_client.go internal/nextdns/client_test.go
git commit -m "feat: add SyncRewrites with diff-based create/delete (C1)

Rewrites API only supports Create/List/Delete (no Update).
SyncRewrites computes the diff and applies creates/deletes."
```

---

### Task 4: Wire all missing syncs into controller

**Files:**
- Modify: `internal/controller/nextdnsprofile_controller.go:559-575` (settings sync section)
- Modify: `internal/controller/nextdnsprofile_controller.go` (add rewrites sync after settings)

**Step 1: Write failing test for performance/web3/log fields sync**

Add to `internal/controller/nextdnsprofile_controller_test.go` a test that verifies settings are fully synced. Find the existing managed mode reconcile test and add a new test:

```go
func TestSyncWithNextDNS_FullSettings(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-full-settings",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name:      "Full Settings Profile",
			ProfileID: "abc123",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
			Settings: &nextdnsv1alpha1.SettingsSpec{
				Logs: &nextdnsv1alpha1.LogsSpec{
					Enabled:       boolPtr(true),
					LogClientsIPs: boolPtr(true),
					LogDomains:    boolPtr(false),
					Retention:     "30d",
				},
				BlockPage: &nextdnsv1alpha1.BlockPageSpec{
					Enabled: boolPtr(true),
				},
				Performance: &nextdnsv1alpha1.PerformanceSpec{
					ECS:             boolPtr(true),
					CacheBoost:      boolPtr(false),
					CNAMEFlattening: boolPtr(true),
				},
				Web3: boolPtr(true),
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	mockNDS := nextdns.NewMockClient()
	mockNDS.SetProfile("abc123", "Full Settings Profile", "abc123.dns.nextdns.io")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		SyncPeriod: 5 * time.Minute,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockNDS, nil
		},
	}

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-full-settings", Namespace: "default"},
	})
	require.NoError(t, err)

	// Verify UpdateSettings was called
	assert.True(t, mockNDS.WasMethodCalled("UpdateSettings"))

	// Verify settings were stored correctly in mock
	settings := mockNDS.Settings["abc123"]
	require.NotNil(t, settings)
	assert.True(t, settings.Web3)
	require.NotNil(t, settings.Performance)
	assert.True(t, settings.Performance.Ecs)
	assert.False(t, settings.Performance.CacheBoost)
	assert.True(t, settings.Performance.CnameFlattening)
	require.NotNil(t, settings.Logs)
	require.NotNil(t, settings.Logs.Drop)
	// LogClientsIPs=true -> Drop.IP=false (inverted)
	assert.False(t, settings.Logs.Drop.IP)
	// LogDomains=false -> Drop.Domain=true (inverted)
	assert.True(t, settings.Logs.Drop.Domain)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/controller/... -run TestSyncWithNextDNS_FullSettings -v`
Expected: FAIL -- performance/web3/log drop fields not passed to UpdateSettings

**Step 3: Update settings sync in controller**

In `internal/controller/nextdnsprofile_controller.go`, replace the settings sync block (around lines 559-575):

```go
	// Sync settings (logs, block page, performance, web3)
	if profile.Spec.Settings != nil {
		settingsConfig := &nextdns.SettingsConfig{
			LogsEnabled:     true,
			LogClientsIPs:   false,
			LogDomains:      true,
			BlockPageEnable: true,
		}
		if profile.Spec.Settings.Logs != nil {
			settingsConfig.LogsEnabled = boolValue(profile.Spec.Settings.Logs.Enabled, true)
			settingsConfig.LogClientsIPs = boolValue(profile.Spec.Settings.Logs.LogClientsIPs, false)
			settingsConfig.LogDomains = boolValue(profile.Spec.Settings.Logs.LogDomains, true)
			settingsConfig.LogRetention = parseRetentionDays(profile.Spec.Settings.Logs.Retention)
		}
		if profile.Spec.Settings.BlockPage != nil {
			settingsConfig.BlockPageEnable = boolValue(profile.Spec.Settings.BlockPage.Enabled, true)
		}
		if profile.Spec.Settings.Performance != nil {
			settingsConfig.Ecs = boolValue(profile.Spec.Settings.Performance.ECS, true)
			settingsConfig.CacheBoost = boolValue(profile.Spec.Settings.Performance.CacheBoost, true)
			settingsConfig.CnameFlattening = boolValue(profile.Spec.Settings.Performance.CNAMEFlattening, true)
		}
		settingsConfig.Web3 = boolValue(profile.Spec.Settings.Web3, false)
		if err := client.UpdateSettings(ctx, profileID, settingsConfig); err != nil {
			return fmt.Errorf("failed to update settings: %w", err)
		}
	}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/controller/... -run TestSyncWithNextDNS_FullSettings -v`
Expected: PASS

**Step 5: Write failing test for rewrites sync**

```go
func TestSyncWithNextDNS_Rewrites(t *testing.T) {
	scheme := newTestScheme()
	ctx := context.Background()

	profile := &nextdnsv1alpha1.NextDNSProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-rewrites",
			Namespace:  "default",
			Finalizers: []string{FinalizerName},
		},
		Spec: nextdnsv1alpha1.NextDNSProfileSpec{
			Name:      "Rewrites Profile",
			ProfileID: "abc123",
			CredentialsRef: nextdnsv1alpha1.SecretKeySelector{
				Name: "nextdns-secret",
			},
			Rewrites: []nextdnsv1alpha1.RewriteEntry{
				{From: "app.example.com", To: "192.168.1.1"},
				{From: "api.example.com", To: "192.168.1.2"},
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nextdns-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}

	mockNDS := nextdns.NewMockClient()
	mockNDS.SetProfile("abc123", "Rewrites Profile", "abc123.dns.nextdns.io")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(profile, secret).
		WithStatusSubresource(profile).
		Build()

	reconciler := &NextDNSProfileReconciler{
		Client: fakeClient,
		Scheme: scheme,
		SyncPeriod: 5 * time.Minute,
		ClientFactory: func(apiKey string) (nextdns.ClientInterface, error) {
			return mockNDS, nil
		},
	}

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-rewrites", Namespace: "default"},
	})
	require.NoError(t, err)

	// Verify SyncRewrites was called
	assert.True(t, mockNDS.WasMethodCalled("SyncRewrites"))

	// Verify rewrites were stored
	rewrites := mockNDS.Rewrites["abc123"]
	require.Equal(t, 2, len(rewrites))
}
```

**Step 6: Run test to verify it fails**

Run: `go test ./internal/controller/... -run TestSyncWithNextDNS_Rewrites -v`
Expected: FAIL -- SyncRewrites never called

**Step 7: Add rewrites sync to controller**

In `internal/controller/nextdnsprofile_controller.go`, add after the settings sync block (before the denylist sync):

```go
	// Sync rewrites
	if len(profile.Spec.Rewrites) > 0 {
		rewriteEntries := make([]nextdns.RewriteEntry, 0, len(profile.Spec.Rewrites))
		for _, rw := range profile.Spec.Rewrites {
			if rw.Active == nil || *rw.Active {
				rewriteEntries = append(rewriteEntries, nextdns.RewriteEntry{
					Name:    rw.From,
					Content: rw.To,
				})
			}
		}
		if err := client.SyncRewrites(ctx, profileID, rewriteEntries); err != nil {
			return fmt.Errorf("failed to sync rewrites: %w", err)
		}
	}
```

**Step 8: Run tests to verify they pass**

Run: `go test ./internal/controller/... -run "TestSyncWithNextDNS_Rewrites|TestSyncWithNextDNS_FullSettings" -v`
Expected: PASS

**Step 9: Run full test suite**

Run: `go test ./... 2>&1 | grep -E "^ok|^FAIL"`
Expected: all PASS

**Step 10: Commit**

```bash
git add internal/controller/nextdnsprofile_controller.go internal/controller/nextdnsprofile_controller_test.go
git commit -m "feat: wire performance, web3, log drop, and rewrites sync into controller (C1)

All spec fields now sync to the NextDNS API in managed mode:
- Performance (ECS, CacheBoost, CNAMEFlattening)
- Web3
- LogClientsIPs / LogDomains (inverted to API Drop fields)
- Rewrites (diff-based create/delete)"
```

---

### Task 5: Regenerate CRDs and run full verification

**Files:**
- Regenerate: `api/v1alpha1/zz_generated.deepcopy.go`
- Regenerate: `config/crd/bases/*.yaml`
- Regenerate: `chart/crds/*.yaml`

**Step 1: Regenerate**

Run: `make generate manifests && make sync-helm-crds`

**Step 2: Run full test suite**

Run: `make test`
Expected: all packages PASS

**Step 3: Commit**

```bash
git add api/v1alpha1/zz_generated.deepcopy.go config/crd/ chart/crds/
git commit -m "chore: regenerate CRDs for critical fixes"
```

---

## Verification

1. `go build ./...` -- compiles
2. `go test ./...` -- all pass
3. `grep -c "SyncRewrites" internal/nextdns/interface.go` -- returns 1
4. Verify `UpdateSettings` uses single PATCH call with performance, web3, log drop
5. Verify `SyncDenylist` no longer pre-fetches the current list
6. Verify controller syncs rewrites, performance, web3, logClientsIPs, logDomains
