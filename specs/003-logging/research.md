# Phase 0 Research: Structured Logging & Command Telemetry

## Decision: Extend telemetry writer with dedicated structured logger
- **Rationale**: Reusing the existing `pkg/telemetry` writer keeps JSON output streaming through a single code path, simplifies correlation ID propagation, and avoids duplicate buffering layers while satisfying FR-001/FR-004.
- **Alternatives Considered**:
  - Introduce a brand-new logging package separate from telemetry: rejected to prevent divergence between metrics and logging outputs.
  - Wrap a third-party logging library: rejected to minimise dependencies and keep logs compatible with existing JSON-line consumers.

## Decision: Sanitize commands via targeted redaction helpers
- **Rationale**: Centralising sanitization in `internal/cli/logging` lets bootstrap, helm, and future command runners share redaction logic (e.g., `--token=***`, secrets in env), ensuring FR-002/FR-006 are met without duplicating regex rules.
- **Alternatives Considered**:
  - Manual sanitization inside each caller: rejected due to drift risk and inconsistent secret coverage.
  - Blanket masking of entire command strings: rejected because operators need command context for troubleshooting per clarification.

## Decision: Capture stderr using dual-writer buffer
- **Rationale**: Wrapping external command execution with a `MultiWriter` preserves operator-facing stderr while copying sanitized excerpts into structured logs, satisfying FR-003 without hiding real-time failures.
- **Alternatives Considered**:
  - Redirect stderr solely into buffers: rejected because it obscures live feedback during long helm/bootstrap steps.
  - Logging full stderr output verbatim: rejected due to sensitive data leakage risk and noisy ELK ingestion.

## Decision: Generate workflow correlation IDs per CLI invocation
- **Rationale**: Deriving a UUID per `chainctl` command and threading it through logging helpers gives operators end-to-end traceability (FR-007) while keeping implementation straightforward.
- **Alternatives Considered**:
  - Per-step random IDs without a root workflow ID: rejected because cross-step correlation would require additional inference in log pipelines.
  - Reusing existing phase strings as IDs: rejected since phases can repeat and are not globally unique across sessions.
