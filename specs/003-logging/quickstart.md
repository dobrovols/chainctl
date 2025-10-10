# Quickstart: Verifying Structured Workflow Logs

## Prerequisites
- Build feature branch: `go build ./cmd/chainctl`
- Access to a test cluster or dry-run-capable environment
- Helm registry credentials configured if using OCI charts
- Terminal configured to capture CLI stdout (e.g., `tee /tmp/chainctl-logs.jsonl`)

## Dry-Run Validation
1. Execute `./chainctl cluster install --values-file ./testdata/values.enc --dry-run --output json | tee /tmp/chainctl-logs.jsonl`.
2. Confirm log stream contains start/end workflow entries with fields `timestamp`, `category`, `message`, `severity`.
3. Verify Helm command log reports sanitized `command` string (no secrets, tokens replaced with `***`).
4. Inspect `workflowId` to ensure all entries share the same UUID.

## Failure Path Validation
1. Force a Helm failure (e.g., supply invalid chart path) and rerun command.
2. Check final entry uses `severity="error"` and includes `stderrExcerpt` with truncated diagnostic details.
3. Confirm stderr still prints to console during execution while sanitized snippet appears in JSON log.
4. Simulate structured logging being disabled or unable to initialize (e.g., point the log sink to a read-only directory or unset required logging env vars) and rerun the command; verify chainctl exits before performing Helm/bootstrap actions and surfaces an actionable error instructing operators to re-enable logging.

## Bootstrap Command Logging
1. Set `CHAINCTL_K3S_INSTALL_PATH` to an invalid script to trigger bootstrap failure.
2. Run `./chainctl cluster install --bootstrap --values-file ./testdata/values.enc`.
3. Confirm log entry with `category="command"` shows the shell command invocation and sanitized environment metadata.

## App Install/Upgrade Logging
1. Run `./chainctl app install --values-file ./testdata/values.enc --values-passphrase secret --chart oci://registry.example.com/apps/myapp:1.2.3 --namespace demo --release-name myapp-demo | tee /tmp/chainctl-app-install.jsonl`.
2. Confirm workflow entries `app-install workflow started/completed` appear with matching `workflowId`.
3. Validate the resolve step emits `helm-resolve` command logs without exposing registry credentials.
4. Trigger an upgrade failure (`--chart` pointing to a bad digest) and ensure the `app-upgrade` workflow ends with `severity="error"` and sanitized stderr.

## Cluster Upgrade Logging
1. Execute `./chainctl cluster upgrade --cluster-endpoint https://cluster.demo --k3s-version v1.30.2+k3s1 | tee /tmp/chainctl-upgrade.jsonl`.
2. Review `upgrade-plan` command entries to confirm plan arguments are logged and sanitized.
3. Force a failure (e.g., missing controller manifest) and ensure the workflow logs include `step="upgrade"` with an actionable error message.

## Integration Tip
- Feed `/tmp/chainctl-logs.jsonl` into your ELK ingestion tool (e.g., `filebeat`) and ensure fields map directly without additional transforms.
- Sample sanitized output is available in `docs/examples/logging/cluster_install.jsonl`.

## Cleanup
- Remove temporary log files: `rm /tmp/chainctl-logs.jsonl /tmp/chainctl-app-install.jsonl /tmp/chainctl-upgrade.jsonl`
