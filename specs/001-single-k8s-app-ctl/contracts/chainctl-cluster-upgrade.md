# Contract: `chainctl cluster upgrade`

## Synopsis
Orchestrate a k3s version upgrade using system-upgrade-controller while monitoring application availability.

```
chainctl cluster upgrade \
  --cluster-endpoint <url> \
  --k3s-version <semver> \
  [--controller-manifest <path>] \
  [--airgapped --bundle-path <path>] \
  [--drain-timeout <duration>] \
  [--surge <int>] \
  [--dry-run] \
  [--output json|text]
```

## Inputs
- `--cluster-endpoint` (string): required.
- `--k3s-version` (semver): required target version.
- `--controller-manifest` (path): optional override; defaults to embedded chart.
- `--airgapped` + `--bundle-path` (path): offline asset source.
- `--drain-timeout` (duration): optional, default `15m`.
- `--surge` (int): optional, default `1`.
- `--dry-run` (bool): simulate upgrade plan only.
- `--output` (enum): json or text.

## Preconditions
- Cluster running supported base version.
- system-upgrade-controller chart accessible (bundle or online).
- All nodes have upgrade tokens/permissions configured.

## Behavior
1. Validate target version > current and within supported window.
2. Deploy/ensure system-upgrade-controller with desired settings.
3. Submit upgrade plan CRD with surge/drain parameters.
4. Stream node upgrade progress; enforce health checks after each node.
5. Confirm application workloads healthy post-upgrade.
6. Emit telemetry and summary output.

## Outputs
- Text timeline of node upgrades and duration.
- JSON object with `nodes` array (name, status, duration), `success` boolean.
- Exit codes: `0` success, `40` validation failure, `41` upgrade failure, `42` timeout.

## Error Handling
- If any node fails upgrade, mark plan failed, capture logs, instruct rollback.
- Timeout triggers abort and returns `42`.
- Missing bundle assets -> code `20`.

## Observability
- Metrics: `chainctl_cluster_upgrade_node_duration_seconds{node=...}`.
- Structured logs with `phase=upgrade` `node=...` per step.

