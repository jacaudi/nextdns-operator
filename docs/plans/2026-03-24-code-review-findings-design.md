# Code Review Findings -- Design

**Issue:** #64
**Date:** 2026-03-24

## Problem

Full codebase review identified 19 findings across critical, important, and minor severity. These will be addressed in three separate PRs by severity.

## PR 1: Critical Fixes (C1, C2)

### C1: Implement full sync for missing spec fields

The spec accepts fields for rewrites, performance, web3, logClientsIPs, and logDomains, but managed mode never syncs them to the NextDNS API. Users configure these settings with no effect.

**Approach:**

1. **Performance (ECS, CacheBoost, CNAMEFlattening):** The nextdns-go library has `SettingsPerformanceService` with `Get`/`Update`. Extend `SettingsConfig` with performance fields and add a `SettingsPerformance.Update` call in `UpdateSettings`.

2. **LogClientsIPs / LogDomains:** The nextdns-go library exposes `SettingsLogs.Drop.IP` and `SettingsLogs.Drop.Domain` with inverted logic (true = don't log). The spec uses positive logic (true = log them). Invert internally in the client with a clear comment. Wire into existing `UpdateSettings`.

3. **Rewrites:** The nextdns-go library has `Rewrites` service with `Create`/`List`/`Delete` (no Update). Implement a `SyncRewrites` method using diff-based sync: list current, compute adds/deletes, apply. Add to `ClientInterface`.

4. **Web3:** The `Settings` struct has a `Web3` bool. Check if `SettingsService.Update` accepts it. If so, add a call in `UpdateSettings`. If not, it may be a read-only field from the API.

5. **Controller:** Wire all new sync calls into `syncWithNextDNS` in the profile controller.

### C2: Remove dead code in SyncDenylist

Remove unused `currentDomains` map and the pre-fetch LIST call from `SyncDenylist`.

## PR 2: Important Fixes (I1-I8)

| Finding | Approach |
|---------|----------|
| I1: Finalizer names | Standardize all to `nextdns.io/<resource>-finalizer`. Pre-v1.0.0, no migration. |
| I2: Metrics LIST overhead | Move `updateResourceMetrics` to a periodic goroutine or reduce to per-reconcile for the single resource type. |
| I3: math/rand | Replace `math/rand` with `math/rand/v2` in `sync.go`. |
| I4: Deep-copy consistency | Document the pattern; controller-runtime's fake client returns copies, so no bug exists. |
| I5: Retention validation | Add `+kubebuilder:validation:Enum=1h;6h;1d;7d;30d;90d;1y;2y` to `LogsSpec.Retention`. |
| I6: Spec removal behavior | Add doc comment to spec fields explaining that removing a section doesn't revert remote state. |
| I7: ResourceReference/ListReference | Remove `ResourceReference` (only used by `SecretKeySelector` which can use its own type). Keep `ListReference`. |
| I8: ServiceMonitor | Remove `ServiceMonitorConfig` from CoreDNS spec. Track as future feature request. |

## PR 3: Minor Fixes (M1-M9)

| Finding | Approach |
|---------|----------|
| M1: List controller duplication | Extract shared reconcile/deletion helpers. Keep controllers separate but DRY. |
| M2: LoadBalancerIP deprecated | Replace with annotation-based approach. |
| M3: Multus IP warnings | Set a `MultusIPWarning` condition on the CR status. |
| M4: CoreDNS profile resolution | Pass resolved profile as parameter to sub-methods. |
| M5: ConfigImport edge case tests | Add nil/empty/large payload test cases. |
| M6: Metrics tests | Add basic test file for metrics registration. |
| M7: API type tests | Convert structural tests to behavioral tests (defaults, validation). |
| M8: IPv6 in LoadBalancerIP | Extend regex pattern to support IPv6 or remove pattern and validate in controller. |
| M9: formatRetentionString(0) lossy | Map 0 to `"1h"` since sub-day retentions round down to 0 days in the API. |

## Decision: LogClientsIPs / LogDomains inversion

The spec uses positive logic (`logClientsIPs: true` = "yes, log client IPs"). The API uses inverted logic via `Drop.IP: true` = "drop/don't log client IPs". We keep the user-friendly positive logic in the spec and invert in the client layer with a clear comment documenting the inversion.
