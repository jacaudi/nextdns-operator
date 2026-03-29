# Fix Reconcile Loop from Status Updates -- Design

**Issue:** #87
**Date:** 2026-03-29

## Problem

`LastSyncTime = metav1.Now()` on every reconcile causes status to always change, triggering a watch event that immediately re-queues -- creating a ~2-second reconcile loop instead of using the configured sync interval.

## Fix

Compare meaningful status fields before and after reconciliation. Only call `Status().Update()` when something actually changed. Use `apiequality.Semantic.DeepEqual` for comparison.

## Scope

Both observe and managed modes.

### Observe mode
- Compare new `observedConfig` against current before writing
- Only update `LastSyncTime` when observed data actually changed
- If nothing changed, skip `Status().Update()` entirely and return `RequeueAfter: syncInterval`

### Managed mode
- Sync always pushes to API, so that's fine
- Only update `LastSyncTime` after a successful sync
- Capture status before reconcile, compare after, skip update if unchanged
