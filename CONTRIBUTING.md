# Contributing

Thanks for helping improve the NextDNS Kubernetes Operator! This guide consolidates the local development flow. For configuration, CRD reference, and architecture, see the [full documentation](docs/README.md).

## Prerequisites

- Go 1.26+ (see [`go.mod`](go.mod))
- [Task](https://taskfile.dev) (`brew install go-task`) — all workflows are wired into [`Taskfile.yml`](Taskfile.yml); run `task --list` to see every target
- Docker (for container builds) and a Kubernetes cluster + `kubectl` (for install/deploy)

`controller-gen` is installed automatically into `bin/` the first time a codegen task runs.

## Development flow

```bash
task test            # manifests + generate + fmt + vet, then go test ./... -coverprofile cover.out
task build           # build the operator binary into bin/manager
task run             # run the controller locally against your kubeconfig
task fmt             # go fmt ./...
task vet             # go vet ./...
```

`test`, `build`, and `run` all depend on `manifests`, `generate`, `fmt`, and `vet`, so generated code and manifests are kept in sync automatically when you run them.

## Regenerating CRDs and RBAC (important)

Changes to the API types (`api/**`) or controller RBAC markers (`internal/**`) require regenerating derived artifacts, and **CI fails if the committed files are out of sync**. Run the relevant task and commit the results:

```bash
task generate            # DeepCopy methods (api/v1alpha1/zz_generated.deepcopy.go)
task manifests           # CRDs + ClusterRole -> config/crd/bases, config/rbac/role.yaml, chart/crds/
task sync-helm-crds      # copy generated CRDs into the Helm chart (chart/crds/)
task generate-helm-rbac  # regenerate the Helm RBAC template from the kubebuilder role
```

If you touched API types or RBAC markers, run `task manifests`, `task sync-helm-crds`, and `task generate-helm-rbac`, then commit every changed file under `config/`, `chart/`, and `api/`. CI runs a diff check and will fail the PR if anything is stale.

## Deploying to a cluster

```bash
task install     # apply CRDs (config/crd/bases) into the cluster
task deploy      # apply the controller manager + RBAC (config/manager, config/rbac)
task undeploy    # remove the controller
task uninstall   # remove the CRDs
```

See [Local Development](README.md#local-development) in the README for the same quick reference.

## Commit messages

Releases are automated with [semantic-release](https://github.com/semantic-release/semantic-release), so commits must follow [Conventional Commits](https://www.conventionalcommits.org/):

| Type | Effect |
|------|--------|
| `fix: ...` | Patch release |
| `feat: ...` | Minor release |
| `feat!: ...` / `BREAKING CHANGE:` | Major release |
| `chore(deps): ...`, `fix(deps): ...` | Patch release |
| `chore:`, `docs:`, `ci:`, `test:` | No release |

## Pull requests

- Open PRs against `main` from a topic branch.
- Run `task test` locally and make sure it passes.
- If you changed API types or RBAC, regenerate and commit CRDs/RBAC (see above) so the CI sync checks pass.
- Include tests for behavior changes and update the relevant page under `docs/` when behavior or configuration changes.
