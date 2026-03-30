# Makefile to Taskfile Migration -- Design

**Date:** 2026-03-29

## Problem

Replace GNU Make with go-task (Taskfile.yml) for a cleaner, YAML-based build system.

## Approach

1:1 task mapping -- same names, same commands. Add smart caching via `sources`/`generates` where applicable.

## Changes

1. Create `Taskfile.yml` with all existing targets
2. Delete `Makefile`
3. Update CI workflows (`make` -> `task`)
4. Update README.md (`make` -> `task`)
