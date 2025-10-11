# Declarative Configuration Runbook

This runbook describes how operators author and consume declarative `chainctl.yaml` files to standardise CLI executions.

## Authoring
1. Start from the sample in [`docs/examples/config/chainctl.yaml`](../examples/config/chainctl.yaml) and commit the file to version control.
2. Populate `defaults` with non-secret flag values that apply to multiple commands (namespaces, bundle paths, `dry-run`, `output`).
3. Define reusable `profiles` when teams share overrides (e.g., staging namespaces). Reference profiles from command sections via the `profiles` list.
4. For each command (e.g., `chainctl cluster install`), list supported flags under `flags`. Use only flags exposed by the CLI; blocked flags return actionable errors during validation.
5. Never include secrets (`values-passphrase`, tokens, kubeconfigs). Provide sensitive values at runtime via environment variables, CI secret stores, or prompt injection.

## Distribution & Discovery
- Operators may pass `--config /path/to/chainctl.yaml` explicitly.
- Auto-discovery order:
  1. `CHAINCTL_CONFIG`
  2. `./chainctl.yaml`
  3. `$XDG_CONFIG_HOME/chainctl/config.yaml`
  4. `$HOME/.config/chainctl/config.yaml`
- Once resolved, chainctl prints a summary table (text) or JSON block (when `--output json`) that lists every flag, value, and origin (default/profile/command/runtime). Review before executing destructive commands.

## Override Rules
1. Load defaults (YAML `defaults`).
2. Apply profiles referenced by the command (`profiles` list).
3. Apply command-specific flags (`commands[<cmd>].flags`).
4. Apply runtime CLI overrides (`--flag value`).

The summary notes all overrides so auditors can confirm provenance.

## Telemetry & Logging
- Structured logs include `category="config"` events with the resolved command, source path, profiles, and individual flag sources. Centralised logging systems can filter on this category for compliance reporting.
- Telemetry metadata attaches the same information to the workflow entries for observability pipelines.

## Troubleshooting
- `ERR_CONFIG_NOT_FOUND`: confirm the file exists or provide an explicit `--config` path.
- `ERR_CONFIG_UNKNOWN_COMMAND`: the YAML references a command not currently enabled; remove or update after upgrading chainctl.
- `ERR_CONFIG_UNKNOWN_FLAG`: remove or fix the flag name; use `chainctl <cmd> --help` to enumerate valid flags.
- `ERR_CONFIG_SECRET_BLOCKED`: relocate sensitive values (passphrases, tokens) to secret storage and inject at runtime.

Store approved `chainctl.yaml` files in a central repo and review diffs like code. The merged summary emitted by chainctl should be captured alongside automation logs for traceability.
