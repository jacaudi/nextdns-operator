# Remove ConfigMap Import Feature -- Design

**Issue:** #68
**Date:** 2026-03-26

## Problem

The ConfigMap Import feature (`spec.configImportRef`) is deprecated in favor of observe mode (shipped in v0.9.0). It should be removed to reduce maintenance surface and avoid user confusion.

## Approach

Pure deletion. No new code, no behavioral changes to remaining features.

## What to Remove

1. **`internal/configimport/` directory** (8 files) -- reader, merge, validate, types + all tests
2. **API types** (`api/v1alpha1/nextdnsprofile_types.go`):
   - `ConfigImportRef` struct
   - `spec.configImportRef` field
   - `status.configImportResourceVersion` field
3. **Controller** (`internal/controller/nextdnsprofile_controller.go`):
   - `configimport` package import
   - `ConditionTypeConfigImported` constant
   - Deprecation warning block
   - Import logic block (ReadAndParse + MergeIntoSpec)
   - `findProfilesForConfigMap` handler + ConfigMap watch registration in SetupWithManager
4. **Controller tests** -- `TestReconcile_ConfigImport` and `TestReconcile_ConfigImport_MissingConfigMap`
5. **Documentation** (`docs/README.md`) -- ConfigMap Import section, condition references, troubleshooting
6. **Sample manifest** -- remove commented configImportRef example
7. **Regenerate** deepcopy, CRDs, sync Helm chart

## What Stays

- Observe mode (replacement feature)
- `configMapRef` (connection details export -- unrelated)
- All other controller logic
