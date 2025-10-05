# chainctl Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-05

## Active Technologies
- Go 1.22 (gofmt + gofumpt enforced) + Cobra CLI, Helm SDK, k3s install scripts, system-upgrade-controller CRDs, OpenTelemetry SDK (001-single-k8s-app-ctl)

## Project Structure
```
cmd/chainctl/        # CLI entrypoints and command wiring
pkg/                 # reusable packages (bootstrap, helm, upgrade, bundle, secrets, tokens, telemetry)
internal/            # validation, config, kubeclient helpers not exported
test/                # unit, integration (envtest/kind), and e2e suites
```

## Commands
- Format & lint: `gofmt ./... && gofumpt ./... && golangci-lint run`
- Unit tests: `go test ./pkg/... ./cmd/...`
- Integration (envtest): `go test ./test/integration/...`
- E2E (kind): `make test-e2e` (to be implemented per plan)
- Benchmarks: `go test -bench . ./pkg/...`

## Code Style
Go 1.22 (gofmt + gofumpt enforced): use wrapped errors, keep exported function docs concise, prefer context-aware operations, follow noun-verb CLI naming.

## Recent Changes
- 001-single-k8s-app-ctl: Added Go 1.22 (gofmt + gofumpt enforced) + Cobra CLI, Helm SDK, k3s install scripts, system-upgrade-controller CRDs, OpenTelemetry SDK

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
