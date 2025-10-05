# Installer Runbook

## Purpose
Guide operators through installing, upgrading, and troubleshooting the chainctl-managed application on k3s clusters.

## Quick Commands
- Set installer script source before bootstrap:
  ```bash
  export CHAINCTL_K3S_INSTALL_URL="https://raw.githubusercontent.com/k3s-io/k3s/v1.30.2%2Bk3s1/install.sh"
  export CHAINCTL_K3S_INSTALL_SHA256="<sha256>"
  ```
  *or* point `CHAINCTL_K3S_INSTALL_PATH` to a pre-downloaded script and set the corresponding SHA256.

- Bootstrap + install (online):
  ```bash
  chainctl cluster install --bootstrap \
    --values-file <encrypted-values> \
    --values-passphrase <passphrase>
  ```
- Install on existing cluster:
  ```bash
  chainctl cluster install --cluster-endpoint https://cluster.local \
    --values-file <encrypted-values> \
    --values-passphrase <passphrase>
  ```
- Application install (OCI chart):
  ```bash
  chainctl app install \
    --values-file values.enc \
    --values-passphrase <passphrase> \
    --chart oci://registry.example.com/apps/myapp:1.2.3 \
    --namespace demo \
    --release-name myapp-demo
  ```
- Application upgrade (bundle fallback):
  ```bash
  chainctl app upgrade \
    --cluster-endpoint https://cluster.local \
    --values-file values.enc \
    --values-passphrase <passphrase> \
    --bundle-path /mnt/app-bundle \
    --state-file /var/lib/chainctl/state.json \
    --output json
  ```
  - Use `--state-file-name custom.json` to keep the managed directory while changing the filename.

- Cluster upgrade plan:
  ```bash
  chainctl cluster upgrade --cluster-endpoint https://cluster.local \
    --k3s-version v1.30.2+k3s1
  ```

## State Management Checklist
1. Default state path: `$XDG_CONFIG_HOME/chainctl/state/app.json`. If unset, falls back to `$HOME/.chainctl/state/app.json`.
2. Set `--state-file` when storing records in a shared location; ensure the directory is writable (chainctl will create it with 0700 permissions).
3. `--state-file-name` changes only the filename inside the managed directory—useful for multi-environment hosts.
4. After each install/upgrade, run:
   ```bash
   cat $(chainctl app upgrade ... --output json | jq -r '.stateFile')
   ```
   to confirm the persisted record (release, namespace, chart reference, version, lastAction, timestamp).
5. On failure, the previous state file is preserved; remediate permissions and rerun with the same overrides.

## Dry-Run Workflow
1. Execute `scripts/capture-dry-run.sh`.
2. Review artifacts under `artifacts/dry-run/` for plan differences and state file previews.
3. Attach the generated files to the pull request template for reviewer context.

## Troubleshooting
| Symptom | Cause | Resolution |
|---------|-------|------------|
| `state file could not be written` after success output | Directory read-only or conflicting overrides | Ensure parent directory is writable, remove conflicting `--state-file`/`--state-file-name`, then rerun command. Deployment remains applied. |
| `exactly one of --chart or --bundle-path must be provided` | Mutually exclusive inputs specified | Drop one flag; OCI has precedence only when explicitly requested |
| Helm upgrade rollback triggered | Application deployment unhealthy | Inspect Helm release history (`helm status <release>`), verify state record for last action and chart reference |
| System-upgrade plan missing | CRD not installed | Verify controller CRDs via `kubectl get crd plans.upgrade.cattle.io` |

## Observability Checklist
- Exporter configured via `CHAINCTL_OTEL_EXPORTER` (`stdout`, `otlp-grpc`, `otlp-http`).
- Helm telemetry now includes `source`, `namespace`, and OCI `digest` attributes.
- Hash cluster identifiers via `CHAINCTL_CLUSTER_ID` for anonymised metrics.
- Attach telemetry samples from `docs/telemetry/state-persistence.md` and `artifacts/dry-run/` to incident reports.

## Disaster Recovery
1. Disable upgrade plan by deleting the `Plan` resource (`kubectl delete plan -n system-upgrade chainctl-upgrade`).
2. Restore previous Helm release (`helm rollback <release> <revision>`).
3. Re-run `chainctl app install --dry-run` or `chainctl app upgrade --dry-run` with the same state overrides to confirm steady state.
4. If state file corruption is suspected, remove or back up the JSON record before rerunning the command; chainctl will recreate it automatically.
