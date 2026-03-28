# SDK Upgrade + Fingerprint Fix -- Design

**Issue:** #74, nextdns-go#35
**Date:** 2026-03-27

## Problem

`status.fingerprint` contains a hardcoded DNS endpoint (`{id}.dns.nextdns.io`) instead of the actual API fingerprint (`fp04d207c439ee4858`). The nextdns-go SDK v0.11.0 didn't expose the fingerprint field.

## Approach

Upgrade nextdns-go to v0.12.0 (which adds `Fingerprint` to `Profile` struct), then replace the hardcoded construction with the real API value.

## Changes

1. `go.mod` -- Bump nextdns-go v0.11.0 -> v0.12.0
2. Controller managed mode (~line 508) -- Use `profile.Fingerprint` from GetProfile response
3. Controller observe mode (~line 722) -- Same replacement
4. Mock client -- Wire `SetProfile` fingerprint param through to `GetProfile` response
5. Tests -- Assert fingerprint matches API value
6. Close jacaudi/nextdns-go#35
