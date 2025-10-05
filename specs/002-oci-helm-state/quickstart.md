# Quickstart: OCI Helm Install/Update with State Persistence

## Prerequisites
- kubectl context pointing at a test cluster (kind or k3s) with access to target namespace.
- OCI registry credentials (if required) available via `helm registry login` or chainctl config.
- `chainctl` built from feature branch (`go build ./cmd/chainctl`).
- State directory writable under `$XDG_CONFIG_HOME/chainctl/state` or `$HOME/.chainctl/state`.

## Install Flow (OCI chart)
1. Run `chainctl app install --chart oci://registry.example.com/apps/myapp:1.2.3 --namespace demo --release-name myapp-demo --app-version 1.2.3`.
2. Observe progress output within 5 seconds and completion within 10 minutes.
3. Verify success message includes release, namespace, and state file path.
4. Inspect state file: `cat ~/.config/chainctl/state/app.json` and confirm JSON fields match command inputs.

5. Re-run install with overrides: `chainctl app install --chart ... --namespace demo --state-file-name custom.json` (or `--state-file /tmp/custom-state.json`) and confirm the state file is written to the requested location.

## Update Flow (OCI chart)
1. Run `chainctl app update --chart oci://registry.example.com/apps/myapp:1.2.4 --namespace demo --release-name myapp-demo --app-version 1.2.4`.
2. Optionally specify `--state-file /tmp/custom-state.json` to reuse a custom location during updates.
3. Confirm CLI detects prior state, applies chart update, and reports new version.
4. Validate state JSON reflects `lastAction: "update"` with updated timestamp and version.

## Local Bundle Flow
1. Execute `chainctl app install --bundle-path /tmp/myapp-bundle --namespace demo --release-name myapp-demo` without `--chart`.
2. Command should reuse bundle assets and write state referencing bundle path.

## Error Handling Checks
- Run install with both `--chart` and `--bundle-path` to confirm CLI exits with validation error before contacting Kubernetes.
- Set state directory to read-only and rerun update; expect Helm changes to apply but CLI to exit non-zero with guidance to fix permissions.

## Cleanup
- Delete Helm release: `helm uninstall myapp-demo -n demo`.
- Remove state file if desired: `rm ~/.config/chainctl/state/app.json`.
