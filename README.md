# chainctl

chainctl is a single-binary Go CLI for installing, upgrading, and operating a Kubernetes (k3s) based micro-services platform across online and air-gapped environments. It supports cluster bootstrap, Helm-based application deployments, system-upgrade-controller driven k3s upgrades, node join workflows, and encrypted configuration handling.

## Features
- **Cluster lifecycle**: `chainctl cluster install` bootstraps new k3s clusters or validates existing ones, performing host and cluster preflight checks.
- **Application upgrades**: `chainctl app upgrade` applies Helm releases with structured telemetry and JSON reporting. Supports OCI-hosted Helm charts via `--chart oci://...` and persists execution state to a local JSON file.
- **Cluster upgrades**: `chainctl cluster upgrade` ensures the system-upgrade-controller stack and submits upgrade plans with rollback awareness.
- **Node onboarding**: `chainctl node token` / `chainctl node join` manage scoped pre-shared tokens for multi-node scaling.
- **Air-gapped ready**: Installer reads bundles from removable media tarballs with checksum validation.
- **Security**: Encrypted values files handled via AES-256-GCM; OTEL exporters support hashed cluster IDs.

## Quickstart
See [`specs/001-single-k8s-app-ctl/quickstart.md`](specs/001-single-k8s-app-ctl/quickstart.md) for full instructions.

Basic bootstrap (online):
```bash
chainctl cluster install \
  --bootstrap \
  --values-file /path/to/values.enc \
  --values-passphrase "$CHAINCTL_VALUES_PASSPHRASE"
```

Reuse an existing cluster:
```bash
chainctl cluster install \
  --cluster-endpoint https://cluster.local \
  --values-file values.enc \
  --values-passphrase "$CHAINCTL_VALUES_PASSPHRASE"
```

App install (OCI chart) with state persistence:
```bash
chainctl app install \
  --values-file values.enc \
  --values-passphrase "$CHAINCTL_VALUES_PASSPHRASE" \
  --chart oci://registry.example.com/apps/myapp:1.2.3 \
  --namespace demo \
  --release-name myapp-demo \
  --state-file-name app.json \
  --output json
```

Application upgrade with JSON output (OCI chart + state persistence):
```bash
chainctl app upgrade \
  --cluster-endpoint https://cluster.local \
  --values-file values.enc \
  --values-passphrase "$CHAINCTL_VALUES_PASSPHRASE" \
  --chart oci://registry.example.com/apps/myapp:1.2.4 \
  --namespace demo \
  --release-name myapp-demo \
  --app-version 1.2.4 \
  --state-file-name app.json \
  --output json
```

State file defaults to `$XDG_CONFIG_HOME/chainctl/state/app.json` (or `$HOME/.chainctl/state/app.json`). Override with `--state-file` (absolute path) or `--state-file-name` (filename within managed directory). On failures to write state, the deployment remains and the CLI reports the error.

## Telemetry
Configure OpenTelemetry exporters via `CHAINCTL_OTEL_EXPORTER=stdout|otlp-grpc|otlp-http`. Instance IDs are hashed using `CHAINCTL_CLUSTER_ID` (defaults to hostname). Sample payloads live in [`docs/telemetry/chainctl_samples.json`](docs/telemetry/chainctl_samples.json).

## Development
```bash
make verify          # fmt + lint + tests + benchmarks
scripts/capture-dry-run.sh  # capture dry-run artifacts under artifacts/dry-run/
```

Helpful docs:
- CLI reference: [`docs/cli/commands.md`](docs/cli/commands.md)
- Performance budgets: [`docs/performance/budgets.md`](docs/performance/budgets.md)
- Runbook: [`docs/runbooks/installer.md`](docs/runbooks/installer.md)
- Compliance checklist: [`docs/compliance/p1-p5-checklist.md`](docs/compliance/p1-p5-checklist.md)

## Artifacts
Generated outputs land in `artifacts/`:
- `artifacts/dry-run/` – CLI dry-run logs (via script).
- `artifacts/performance/` – Benchmark baselines (`install_baseline.json`, `upgrade_baseline.json`).

## Tests
- Unit tests: `go test ./test/unit`
- Integration (envtest): `KUBEBUILDER_ASSETS=... go test ./test/integration`
- e2e (requires KIND/kubeconfig): `CHAINCTL_E2E=1 go test ./test/e2e`

## License
This project is distributed under the MIT License.
