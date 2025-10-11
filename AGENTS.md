# chainctl Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-05

## Active Technologies
- Go 1.24 (gofmt + gofumpt enforced) + Cobra CLI, Helm SDK, k3s install scripts, system-upgrade-controller CRDs, OpenTelemetry SDK (001-single-k8s-app-ctl)
- Go 1.24 + Cobra CLI (`spf13/cobra`), Helm SDK (`helm.sh/helm/v3`), existing `pkg/bundle`, `pkg/helm`, `pkg/telemetry` (002-oci-helm-state)
- Local filesystem JSON state file under user config directory (002-oci-helm-state)
- Go 1.24 + `spf13/cobra`, `helm.sh/helm/v3`, existing `pkg/telemetry`, `pkg/bootstrap`, `pkg/helm`, OpenTelemetry exporters (003-logging)
- N/A (logs streamed to stdout/stderr for aggregation) (003-logging)
- Go 1.24 + `spf13/cobra`, `helm.sh/helm/v3`, internal `pkg/state`, `pkg/telemetry`, `internal/config`, YAML parsing via `gopkg.in/yaml.v3` (004-declarative-config-file)
- Local filesystem YAML files under repo or XDG config directories (read-only at runtime) (004-declarative-config-file)

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
Go 1.24 (gofmt + gofumpt enforced): use wrapped errors, keep exported function docs concise, prefer context-aware operations, follow noun-verb CLI naming.

## Recent Changes
- 004-declarative-config-file: Added Go 1.24 + `spf13/cobra`, `helm.sh/helm/v3`, internal `pkg/state`, `pkg/telemetry`, `internal/config`, YAML parsing via `gopkg.in/yaml.v3`
- 003-logging: Added Go 1.24 + `spf13/cobra`, `helm.sh/helm/v3`, existing `pkg/telemetry`, `pkg/bootstrap`, `pkg/helm`, OpenTelemetry exporters
- 003-logging: Added Go 1.24 + `spf13/cobra`, `helm.sh/helm/v3`, existing `pkg/telemetry`, `pkg/bootstrap`, `pkg/helm`, OpenTelemetry exporters

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
