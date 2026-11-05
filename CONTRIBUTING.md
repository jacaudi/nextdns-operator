# Contributing

Thanks for helping improve the NextDNS Kubernetes Operator! This guide consolidates the local development flow. For configuration, CRD reference, and architecture, see the [full documentation](docs/README.md).

## Prerequisites

- Go 1.27+ (see [`go.mod`](go.mod))
- [Task](https://taskfile.dev) (`brew install go-task`). Every workflow is wired into [`taskfile.yml`](taskfile.yml) and [`.taskfiles/`](.taskfiles/); run `task --list` to see every target
- The linters `task ci` runs: [golangci-lint](https://golangci-lint.run) v2.15+, [actionlint](https://github.com/rhysd/actionlint), [yamllint](https://github.com/adrienverge/yamllint), [hadolint](https://github.com/hadolint/hadolint), and [Helm](https://helm.sh)
- Docker (for container builds) and a Kubernetes cluster + `kubectl` (for install/deploy)

`controller-gen` is installed automatically into `bin/` the first time a codegen task runs. It is not reinstalled when `CONTROLLER_TOOLS_VERSION` changes, so after switching to a branch that pins a different version, delete `bin/controller-gen` first.

## Development flow

```bash
task ci              # everything CI runs: lint, tests + coverage floor, generated-artifact check. Run before pushing.
task test            # tests only (race detector)
task lint            # static checks only (golangci-lint, actionlint, yamllint, hadolint, helm lint + template)
task go:fmt          # format Go sources
task build           # build the operator binary into bin/manager
task run             # run the controller locally against your kubeconfig
```

CI runs exactly `task ci`, so a green local run is the best predictor of a green PR.

## Regenerating CRDs and RBAC (important)

Changes to the API types (`api/**`) or controller RBAC markers (`internal/**`) require regenerating derived artifacts, and **`task ci` fails if the committed files are out of sync** (`task verify-generated`). Regenerate and commit the results:

```bash
task generate            # DeepCopy methods (api/v1alpha1/zz_generated.deepcopy.go)
task manifests           # CRDs + ClusterRole -> config/crd/bases, config/rbac/role.yaml, and copies the CRDs into chart/crds/
task generate-helm-rbac  # regenerate the Helm RBAC template from the kubebuilder role
```

If you touched API types or RBAC markers, run `task generate manifests generate-helm-rbac`, then commit every changed file under `api/`, `config/`, and `chart/`.

## Deploying to a cluster

```bash
task install     # apply CRDs (config/crd/bases) into the cluster
task deploy      # apply the controller manager + RBAC (config/manager, config/rbac)
task undeploy    # remove the controller
task uninstall   # remove the CRDs
```

See [Local Development](README.md#local-development) in the README for the same quick reference.

## Commit messages

Releases are automated with [release-please](https://github.com/googleapis/release-please), so commits must follow [Conventional Commits](https://www.conventionalcommits.org/) with a scope, e.g. `fix(chart): ...`:

| Type | Effect |
|------|--------|
| `fix(scope): ...` | Patch release |
| `feat(scope): ...` | Minor release |
| `feat(scope)!: ...` / `BREAKING CHANGE:` footer | Major release |
| `deps`, `perf`, `refactor`, `revert` | Listed in the changelog |
| `docs`, `chore`, `ci`, `build`, `test`, `style` | Hidden from the changelog |

The changelog sections are defined in [`release-please-config.json`](release-please-config.json).

## Pull requests

- Open PRs against `main` from a topic branch.
- Run `task ci` locally and make sure it passes.
- If you changed API types or RBAC, regenerate and commit CRDs/RBAC (see above).
- Include tests for behavior changes and update the relevant page under `docs/` when behavior or configuration changes.
