# BAV (Bypass Age Verification) Support -- Design

**Issue:** #78 (partial -- BAV was deferred pending SDK update)
**Date:** 2026-03-28

## Problem

The NextDNS API returns a `bav` boolean in settings (Bypass Age Verification). SDK v0.13.0 now exposes it. The operator doesn't read or sync it.

## Approach

Same pattern as Web3: add to observed types, spec types, settings config, and wire through all three paths (observe, managed, suggestedSpec).
