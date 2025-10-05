# Phase 1 Data Model: OCI Helm State

## Entities

### ChartSource
- **Attributes**:
  - `type` (`enum[oci, bundle]`)
  - `reference` (string; OCI URL or filesystem path)
  - `credentialsRef` (optional string; name of registry auth context)
  - `digest` (optional string; populated after pull)
- **Rules**:
  - Exactly one source per command invocation.
  - OCI references must match `oci://<registry>/<repo>:<tag>` format.

### ReleaseOptions
- **Attributes**:
  - `releaseName` (string; default derived from profile when unset)
  - `appVersion` (string; optional, surfaces in telemetry/state)
  - `namespace` (string; required for cluster install/update)
  - `dryRun` (bool; existing flag support reused)
  - `confirm` (bool; existing UX expectation)
- **Rules**:
  - Namespace must be non-empty and validated before execution.
  - Release name sanitized to Helm naming constraints.

### ExecutionStateRecord
- **Attributes**:
  - `release` (string; Helm release identifier)
  - `namespace` (string)
  - `chart` (object; embeds `ChartSource` summary)
  - `version` (string; application version or chart tag)
  - `lastAction` (`enum[install, update]`)
  - `timestamp` (RFC3339 string)
  - `clusterEndpoint` (string; optional for audit)
- **Rules**:
  - Written after successful action; append-only semantics via overwrite with latest record.
  - Write failures return explicit error without undoing deployment.

### StateFileConfig
- **Attributes**:
  - `directory` (string; defaults to managed config directory, overrideable via `--state-file`)
  - `fileName` (string; defaults to `app.json`, overrideable via `--state-file-name`)
  - `absolutePath` (string; populated when operator supplies `--state-file`)
  - `permissions` (string; expected `0600` file, `0700` directories)
- **Rules**:
  - `absolutePath` and `fileName` overrides are mutually exclusive.
  - Overrides validated before Helm operations; invalid inputs block execution.
  - When `absolutePath` supplied, directory must exist or be creatable by CLI.

## Relationships
- `ExecutionStateRecord.chart` references `ChartSource` to snapshot origin.
- CLI flag parsing populates `ReleaseOptions`, which combined with `ChartSource` drives Helm operations.
- `StateFileConfig` governs where `ExecutionStateRecord` is written and is derived from CLI overrides plus defaults.

## State Transitions
1. **Initialize State**: If file missing, create the current record before action using defaults or overrides.
2. **Install Action**: On success, write/overwrite the current record with release details.
3. **Update Action**: Same as install; `lastAction` reflects `update`.
4. **Failure Paths**: If Helm fails, state file unchanged; if write fails, log and return error while leaving deployment intact.

## Validation Hooks
- Pre-flight ensures mutually exclusive source selection, namespace presence, and valid state override combinations.
- State writer enforces directory creation with `0700` permissions and atomic write via temp file + rename.
- JSON schema (see contracts) validated in unit tests using gojsonschema.
