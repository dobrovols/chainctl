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
- `make test` — unit tests with local Go build cache.
- `make bench` — smoke benchmarks against bundle/helm operations.

Ensure your user has sudo privileges when bootstrapping k3s (per preflight requirements).
