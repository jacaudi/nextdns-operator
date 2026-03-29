# CI/CD Pipeline Optimization -- Design

**Date:** 2026-03-29

## Changes

1. Rename `docker` job to `container` in ci-cd.yml
2. Remove `validate-versions` job
3. Run `container` and `helm` in parallel (both depend only on `release`)
4. Rename `docker-amd64`/`docker-arm64` to `container-amd64`/`container-arm64` in pr.yml
5. Update comments throughout
