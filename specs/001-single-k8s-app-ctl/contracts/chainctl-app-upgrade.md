# Contract: `chainctl app upgrade`

## Synopsis
Upgrade the deployed micro-services application Helm release on an existing k3s cluster.

```
chainctl app upgrade \
  --cluster-endpoint <url> \
  --values-file <path> \
  [--values-passphrase <passphrase>] \
  [--bundle-path <path>] \
  [--airgapped] \
  [--dry-run] \
  [--output json|text]
```

## Inputs
- `--cluster-endpoint` (string): required reachable API endpoint.
- `--values-file` (path): required encrypted values file.
- `--values-passphrase` (string): optional for non-interactive use.
- `--bundle-path` (path): required for air-gapped upgrades.
- `--airgapped` (bool): indicates offline assets should be used.
- `--dry-run` (bool): renders diff only.
- `--output` (enum): `json` or `text` (default text).

## Preconditions
- Cluster credentials available via kubeconfig.
- Values file decrypts successfully.
- Helm release exists.

## Behavior
1. Validate cluster compatibility with target chart version.
2. Decrypt values file and render Helm manifest diff; display in dry-run.
3. Apply Helm upgrade with rollout monitoring.
4. Verify application health checks (deployments ready, jobs complete).
5. Emit telemetry and summary output.

## Outputs
- Text summary showing chart version, status, duration, follow-up actions.
- JSON includes `helmRevision`, `durationSeconds`, `healthChecks` array.
- Exit codes: `0` success, `31` validation failure, `32` upgrade failure.

## Error Handling
- On health check failure, attempt Helm rollback and report error.
- Missing bundle assets produce code `20` with missing artifact list.
- Decrypt error returns `11` and halts upgrade.

## Observability
- Logs include helm diff digest, success/failure events.
- Metrics: `chainctl_upgrade_duration_seconds`, `chainctl_upgrade_rollbacks_total`.

