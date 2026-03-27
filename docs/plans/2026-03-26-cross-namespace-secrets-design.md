# Cross-Namespace Secret References -- Design

**Issue:** #70
**Date:** 2026-03-26

## Problem

`credentialsRef` only supports `name` and `key`, requiring the Secret to be in the same namespace as the CR. In GitOps setups, credentials often live in a different namespace.

## Approach

Add optional `namespace` field to `SecretKeySelector`. When omitted, defaults to the CR's namespace (backward compatible).

## Changes

1. `api/v1alpha1/shared_types.go` -- Add `Namespace` field to `SecretKeySelector`
2. `internal/controller/nextdnsprofile_controller.go` -- `getAPIKey()` uses `credentialsRef.namespace` if set
3. `internal/controller/nextdnsprofile_controller.go` -- `findProfilesForSecret()` matches cross-namespace refs
4. Regenerate CRDs
5. Tests for cross-namespace resolution
6. Update docs

## RBAC

Already cluster-scoped (`ClusterRole` with `secrets: get,list,watch`). No changes needed.
