# Logging & Telemetry Runbook

Structured logging is emitted by `chainctl` alongside existing telemetry events.

## Capturing Logs
- Pipe CLI output through `tee` to persist JSON logs, e.g.:
  ```bash
  chainctl cluster install ... | tee /tmp/cluster-install.jsonl
  ```
- Each entry includes `workflowId`, `category`, `severity`, and sanitized `command`/`stderrExcerpt` fields.
- Sample sanitized output is available in `docs/examples/logging/cluster_install.jsonl`.

## Centralised Ingestion
- Configure Filebeat/Fluent Bit to watch the JSONL output and ship to ELK/Stackdriver.
- Ensure pipeline treats each line as independent JSON; fields are already normalized.

## Troubleshooting
- If the CLI reports `initialize structured logging` errors, no cluster changes were made; check stdout redirection or file permissions and rerun.
- Secrets detected in command args or stderr are replaced with `***`. Investigate sanitiser gaps if real secrets appear.

See `docs/runbooks/installer.md` for operational walkthroughs that integrate these logging steps.
