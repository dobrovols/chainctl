# Contract: `chainctl cluster install`

## Synopsis
Install the micro-services application on a new or existing k3s cluster, optionally bootstrapping k3s first.

```
chainctl cluster install \
  [--bootstrap] \
  [--cluster-endpoint <url>] \
  [--k3s-version <semver>] \
  [--values-file <path>] \
  [--values-passphrase <passphrase>] \
  [--bundle-path <path>] \
  [--airgapped] \
  [--dry-run] \
  [--output json|text]
```

## Inputs
- `--bootstrap` (bool): provision k3s before installing; defaults to false.
- `--cluster-endpoint` (string): required when reusing cluster; mutually exclusive with `--bootstrap`.
- `--k3s-version` (semver): optional override; validated against supported range.
- `--values-file` (path): required; must point to encrypted values file.
- `--values-passphrase` (string): optional; if omitted prompt securely.
- `--bundle-path` (path): required when `--airgapped`; must contain bundle manifest `bundle.yaml`.
- `--airgapped` (bool): toggles offline mode; enforces bundle usage.
- `--dry-run` (bool): executes preflight and renders plan without applying changes.
- `--output` (enum): human-readable text or JSON (default text).

## Preconditions
- Host passes preflight checks (CPU, RAM, kernel modules, SELinux).
- When `--bootstrap`, k3s not already installed.
- Encrypted values file decrypts successfully with passphrase.

## Behavior
1. Run preflight checks; report findings.
2. Resolve bundle assets (online download or air-gapped mount); verify checksums.
3. If `--bootstrap`, install k3s and wait for API ready.
4. Decrypt values file; load Helm chart values.
5. Apply Helm release; wait for workloads ready condition.
6. Emit telemetry envelope per phase.
7. Write summary to stdout or JSON.

## Outputs
- Text mode: table summarising phase durations, status, next steps.
- JSON mode: structured object `{ phases: [...], clusterEndpoint, releaseVersion }`.
- Exit codes: `0` success, `10` preflight failure, `20` bundle validation failure, `30` Helm failure.

## Error Handling
- Preflight failure aborts before any mutation; returns code `10` with remediation list.
- Bundle checksum mismatch triggers rollback and code `20`.
- Helm upgrade failure triggers automatic rollback (if possible) and returns code `30`.
- Passphrase failure prompts retry up to 3 times before aborting with code `11`.

## Observability
- Structured logs at info level for each phase with correlation ID.
- Metrics: `chainctl_phase_duration_seconds`, `chainctl_install_success_total`.

