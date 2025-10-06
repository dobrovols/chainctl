# chainctl Development Environment

This project targets Go 1.24 and Linux hosts (Ubuntu 22.04+, RHEL 9+) for execution. Development requires the following:

## Tooling
- Go 1.24 (install via `asdf` or `goenv` to match CI).
- Docker + KIND and/or k3s for integration/e2e testing.
- `golangci-lint` and `gofumpt` (installed via `go install`).
- Helm 3, kubectl, and system-upgrade-controller binaries pinned per constitution guardrails.

## Environment Variables
- `CHAINCTL_VALUES_PASSPHRASE` for non-interactive encryption flows.
- `KUBECONFIG` pointing to target clusters in reuse mode.

## Air-gapped Setup
- Mount removable-media tarball under `/opt/chainctl/bundles/<version>` with checksum manifest.
- Ensure encrypted values file is accessible (see `chainctl encrypt-values`).

## Command Reference
- `make fmt` — run gofmt/gofumpt.
- `make lint` — golangci-lint (requires binary installed).
- `make test` — unit + integration tests (skips e2e when `CHAINCTL_SKIP_E2E=1`).
- `make bench` — smoke benchmarks against bundle/helm operations.

For end-to-end coverage:

```bash
CHAINCTL_E2E=1 \
CHAINCTL_E2E_SUDO=1 \
go test ./test/e2e/...
```

Optional variables such as `CHAINCTL_E2E_CLUSTER_REUSE`, `CHAINCTL_CLUSTER_ENDPOINT`,
and `CHAINCTL_JOIN_TOKEN` control reuse-mode validation and node join flows. See
`test/e2e/README.md` for the complete matrix.

Ensure your user has sudo privileges when bootstrapping k3s (per preflight requirements).

## Continuous Integration

GitHub Actions runs two jobs:

1. `build-and-test` — linting, unit, and integration suites with e2e skipped.
2. `e2e-tests` — provisions kernel modules via sudo, sets the e2e environment,
   and executes `go test ./test/e2e/...`.

The e2e job relies on the sudo-aware helpers in `test/e2e` and skips scenarios
that require real kubeconfig/token inputs when they are not available.
