# Governance Decisions

## 2025-10-06 · Structured Logging Enforcement
- **Summary**: Adopt workflow-level structured logging for cluster and application commands.
- **Motivation**: Aligns with Constitution v1.2.0 P5 requirements to capture sanitized command context and stderr for centralized troubleshooting.
- **Implications**:
  - Bootstrap and Helm operations now fail fast if structured logging cannot initialize.
  - CLI workflows emit correlation-friendly workflow IDs shared with telemetry.
  - Documentation and runbooks updated to guide operators on capturing JSON logs.
- **Follow-up**: Monitor benchmark budget (<5% per entry) and expand sample log catalog as new workflows are instrumented.
