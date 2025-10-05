# Installer Runbook

## Purpose
Guide operators through installing, upgrading, and troubleshooting the chainctl-managed application on k3s clusters.

## Quick Commands
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
- Application upgrade:
  ```bash
  chainctl app upgrade --cluster-endpoint https://cluster.local \
    --values-file <encrypted-values> \
    --values-passphrase <passphrase> --output json
  ```
- Cluster upgrade plan:
  ```bash
  chainctl cluster upgrade --cluster-endpoint https://cluster.local \
    --k3s-version v1.30.2+k3s1
  ```

## Dry-Run Workflow
1. Execute `scripts/capture-dry-run.sh`.
2. Review artifacts under `artifacts/dry-run/` for plan differences.
3. Attach the generated files to the pull request template for reviewer context.

## Troubleshooting
| Symptom | Cause | Resolution |
|---------|-------|------------|
| `requires sudo privileges` during dry-run | Preflight host check requires elevated privileges | Re-run via `sudo` or adjust preflight overrides in Dev environments |
| Cluster validation failure | Kubeconfig missing or cluster unreachable | Ensure `KUBECONFIG` points to valid cluster and network access is available |
| Helm upgrade rollback triggered | Application deployment unhealthy | Inspect Helm release history (`helm status chainapp`), run `chainctl cluster install --dry-run` to inspect changes |
| System-upgrade plan missing | CRD not installed | Verify controller CRDs via `kubectl get crd plans.upgrade.cattle.io` |

## Observability Checklist
- Exporter configured via `CHAINCTL_OTEL_EXPORTER` (`stdout`, `otlp-grpc`, `otlp-http`).
- Logs and metrics hashed via `CHAINCTL_CLUSTER_ID`.
- Attach telemetry samples from `artifacts/dry-run/` and `docs/telemetry/chainctl_samples.json` to incident reports.

## Disaster Recovery
1. Disable upgrade plan by deleting the `Plan` resource (`kubectl delete plan -n system-upgrade chainctl-upgrade`).
2. Restore previous Helm release (`helm rollback chainapp <revision>`).
3. Re-run `chainctl cluster install --dry-run` to confirm steady state before reapplying.
