# Phase 1 Data Model: Declarative Configuration Profiles

## Entities

### ConfigurationProfile
- **Attributes**:
  - `metadata.name` (string; optional human-readable identifier)
  - `metadata.description` (string; optional summary for operators)
  - `defaults` (map[string]any; shared flag values applied to all commands)
  - `commands` (map[string]CommandSection; keyed by command path such as `cluster install`)
  - `sourcePath` (string; absolute path of the loaded YAML file)
- **Rules**:
  - `defaults` keys MUST map to valid CLI flags for every command that consumes them; invalid keys trigger validation errors before execution.
  - `commands` keys MUST match registered Cobra command paths; unknown commands are rejected.
  - `sourcePath` populated after discovery to aid logging, never user-specified within YAML.

### CommandSection
- **Attributes**:
  - `flags` (map[string]FlagValue; explicit flag overrides for the command)
  - `profiles` ([]string; optional list referencing named reusable flag groups)
  - `disabled` (bool; optional control to block execution via config)
- **Rules**:
  - `flags` keys MUST exist on the targeted command; duplicates resolved by last writer semantics during YAML parsing.
  - `profiles` entries MUST reference known reusable sets defined under `defaults` or top-level `profiles`.
  - `disabled` defaults to `false`; when `true`, command execution aborts with guidance.

### FlagValue
- **Attributes**:
  - `value` (any; string, bool, numeric, list depending on flag type)
  - `source` (enum: `default`, `command`, `runtime`; assigned during resolution)
- **Rules**:
  - Secret-attributed flags (token/password) MUST NOT appear with `source` equal to `default` or `command`; validation fails before runtime merge.
  - Values MUST coerce cleanly into the destination flag type; type mismatches yield actionable errors.

### ResolvedInvocation
- **Attributes**:
  - `commandPath` (string; Cobra command full path)
  - `flags` (map[string]FlagValue; final flag set after merging runtime flags)
  - `overrides` ([]string; ordered list describing precedence decisions)
  - `warnings` ([]string; informational messages surfaced to operators)
- **Rules**:
  - `flags` map MUST include values for all required CLI flags either from YAML or runtime.
  - `overrides` records each precedence step, including runtime flags that superseded YAML values to support auditing.
  - `warnings` highlight ignored keys, deprecated flags, or disabled commands discovered during resolution.

## Relationships
- `ConfigurationProfile` aggregates multiple `CommandSection` entries plus shared `defaults`.
- Each `CommandSection` references zero or more reusable flag group names defined in the same `ConfigurationProfile`.
- `ResolvedInvocation` derives from combining `ConfigurationProfile.defaults`, the relevant `CommandSection.flags`, and runtime flag overrides, annotating each contributing `FlagValue`.

## State Transitions
1. **Discovery**: `ConfigurationProfile` created after locating YAML path via `internal/config/locator`.
2. **Parsing & Validation**: YAML decoded into `ConfigurationProfile`; validation rejects unrecognized commands, flags, or secret usage before proceeding.
3. **Resolution**: On command execution, `ResolvedInvocation` builds by merging defaults → command section → runtime flags while tracking `overrides`.
4. **Execution Feedback**: After command completes, `warnings` and effective configuration summary surfaced, and telemetry/logging record `commandPath`, `sourcePath`, and override chain.

## Validation Hooks
- Unit tests ensure secret detection, unknown command/flag errors, and precedence ordering behave deterministically.
- Integration tests load sample YAML files to verify that `ResolvedInvocation` matches expected flag sets for install/upgrade flows.
- Schema validation ensures YAML structure matches `ConfigurationProfile` expectations before merging, preventing runtime panics.
