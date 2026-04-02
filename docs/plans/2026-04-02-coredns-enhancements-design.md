# CoreDNS Enhancements -- Design

**Issues:** #93, #94, #95
**Date:** 2026-04-02

## #93: Warn when deviceName used with plain DNS

Add warning condition `DeviceNameIgnored` when `protocol: dns` + `deviceName` is set. Log warning, don't block reconciliation.

## #94: Remove ServiceMonitor from CRD

Remove `ServiceMonitorConfig` struct and `ServiceMonitor` field. ServiceMonitor stays Helm-only via bjw-s. Update docs.

## #95: Add PodDisruptionBudget support

Add `PodDisruptionBudget *CoreDNSPDBConfig` to deployment config with `MinAvailable` and `MaxUnavailable` fields. Reconcile PDB when replicas > 1. Default `MaxUnavailable: 1` when enabled without explicit values. Owner reference for GC.

## Execution

All three in one branch/PR. Regen CRDs once. Update docs for all changes.
