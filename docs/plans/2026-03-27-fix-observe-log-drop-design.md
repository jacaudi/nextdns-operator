# Fix Observe Mode Log Drop Inversion -- Design

**Issue:** #75
**Date:** 2026-03-27

## Problem

`readFullProfile` discards `settings.Logs.Drop.IP` and `settings.Logs.Drop.Domain` from the API response. `ObservedLogs` has no fields for them. `buildSuggestedSpec` produces nil `LogClientsIPs`/`LogDomains`.

## Approach

Add `LogClientsIPs` and `LogDomains` bools to `ObservedLogs` using user-friendly positive semantics (matching the spec). Invert the API's `Drop` bools in `readFullProfile`. Pass through in `buildSuggestedSpec`.

## Changes

1. `api/v1alpha1/nextdnsprofile_observed_types.go` -- Add `LogClientsIPs` and `LogDomains` to `ObservedLogs`
2. `internal/controller/nextdnsprofile_controller.go` -- Read `Drop` fields in `readFullProfile`, invert to positive semantics
3. `internal/controller/nextdnsprofile_controller.go` -- Pass through in `buildSuggestedSpec`, remove "not available" comments
4. Tests -- Update observe mode test to populate Drop in mock, assert correct inversion
5. Regenerate CRDs

## Inversion Logic

| API Field | API Value | ObservedLogs Field | ObservedLogs Value |
|-----------|-----------|--------------------|--------------------|
| `Drop.IP = false` | Don't drop IPs (log them) | `LogClientsIPs` | `true` |
| `Drop.IP = true` | Drop IPs (don't log) | `LogClientsIPs` | `false` |
| `Drop.Domain = false` | Don't drop domains (log them) | `LogDomains` | `true` |
| `Drop.Domain = true` | Drop domains (don't log) | `LogDomains` | `false` |
