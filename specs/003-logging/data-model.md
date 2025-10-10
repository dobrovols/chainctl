# Phase 1 Data Model: Structured Workflow Logging

## Entities

### WorkflowLogContext
- **Attributes**:
  - `workflowId` (UUID string; generated per CLI invocation)
  - `step` (string; high-level action such as `bootstrap`, `helm`, `upgrade-apply`)
  - `category` (enum: `workflow`, `command`, `diagnostic`)
  - `severity` (enum: `info`, `warn`, `error`)
  - `correlationId` (string; mirrors `workflowId` for root events, may append suffix for nested commands)
- **Rules**:
  - `severity` defaults to `info` on start, escalates to `error` on failure entries.
  - `step` MUST match CLI workflow enumerations to avoid drift across metrics/logs.

### StructuredLogEntry
- **Attributes**:
  - `timestamp` (RFC3339 string; required)
  - `category` (string; copied from context)
  - `message` (string; human-readable summary of the action or failure)
  - `severity` (string; info/warn/error)
  - `workflowId` (string; UUID traced across entries)
  - `step` (string; optional for command-level logs)
  - `command` (string; sanitized command or helm invocation)
  - `stderrExcerpt` (string; optional sanitized snippet present only on failures)
  - `metadata` (object of string key/value; includes `phase`, `duration_ms`, `mode`, etc.)
- **Rules**:
  - `stderrExcerpt` MUST be absent when command succeeds.
  - `metadata` keys MUST be lowercase with hyphen separators to stay consistent across emitters.
  - Entries MUST include required fields (`timestamp`, `category`, `message`, `severity`).

### CommandInvocation
- **Attributes**:
  - `rawCommand` ([]string; process arguments prior to sanitization)
  - `sanitizedCommand` (string; produced by sanitization helper)
  - `env` (map[string]string; captured for redaction, not logged verbatim)
  - `exitCode` (int; 0 success, non-zero failure)
  - `stderrBuffer` (bytes; capped excerpt stored for logging)
- **Rules**:
  - Sanitization MUST mask tokens/passwords defined by allow/deny list before stringifying.
  - `stderrBuffer` truncated to configured limit (default 4KB) to avoid flooding logs.
  - `sanitizedCommand` included in log entry when populated, otherwise omitted.

## Relationships
- `WorkflowLogContext` is created once per CLI invocation and embedded in each `StructuredLogEntry`.
- Each `CommandInvocation` produces at least one `StructuredLogEntry` (`category=command`).
- Workflow steps (bootstrap, helm) emit start/finish entries referencing the same `workflowId` and `step` for correlation.

## State Transitions
1. **Start Workflow**: Generate `workflowId`, emit `severity=info` start entry with `category=workflow`.
2. **Execute Command**: Log sanitized command before execution, stream stdout/stderr, buffer stderr; on completion emit success/failure entry.
3. **Handle Failure**: On non-zero exit, include truncated `stderrExcerpt`, escalate severity to `error`, populate `metadata.exit_code`.
4. **Complete Workflow**: Emit final `category=workflow` entry summarising outcome and duration; ensure correlation continuity.

## Validation Hooks
- Unit tests ensure sanitization masks known secret patterns and leaves non-sensitive tokens intact.
- JSON schema validation ensures emitted entries satisfy required fields and data types.
- Integration tests inspect CLI output to verify workflow start/end brackets command logs and correlate via shared `workflowId`.
