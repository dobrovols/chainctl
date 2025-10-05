# CLI Contract: chainctl app install/update

## Commands
- `chainctl app install`
- `chainctl app update`

Both commands share flag parsing; `update` defaults to pulling current state to determine release/namespace when flags omitted.

## Flags
| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--chart` | string | Conditional* | OCI chart reference `oci://registry/repo:tag`; mutually exclusive with `--bundle-path`. |
| `--bundle-path` | string | Conditional* | Path to local Helm bundle directory; mutually exclusive with `--chart`. |
| `--release-name` | string | Optional | Overrides Helm release name; defaults to profile or derived from chart. |
| `--app-version` | string | Optional | Application version recorded in telemetry/state. |
| `--namespace` | string | Required | Target Kubernetes namespace for release. |
| `--cluster-endpoint` | string | Required | Kubernetes API endpoint (existing flag reused). |
| `--values-file` | string | Required | Existing encrypted values file (reused). |
| `--values-passphrase` | string | Optional | Passphrase for encrypted values. |
| `--output` | string | Optional | `text` (default) or `json`. |
| `--dry-run` | bool | Optional | Executes validation without applying changes. |
| `--confirm` | bool | Optional | Required when running non-interactive destructive updates. |
| `--state-file-name` | string | Optional | Overrides the default state file name within the managed config directory. |
| `--state-file` | string | Optional | Absolute path for the state JSON file; mutually exclusive with `--state-file-name`. |

\* Exactly one of `--chart` or `--bundle-path` must be provided.  
\* `--state-file` and `--state-file-name` cannot be used together; the CLI validates the selection before running.

## Success Output (text)
```
Install completed successfully for release myapp-demo in namespace demo.
State written to /Users/alex/.config/chainctl/state/app.json
```

## Success Output (json)
```json
{
  "status": "success",
  "action": "install",
  "release": "myapp-demo",
  "namespace": "demo",
  "chart": "oci://registry.example.com/apps/myapp:1.2.3",
  "stateFile": "/Users/alex/.config/chainctl/state/app.json",
  "timestamp": "2025-10-05T12:34:56Z"
}
```

## Error Scenarios
1. **Mutually exclusive flags**
   - Exit code: 1
   - Message: `exactly one of --chart or --bundle-path must be provided`
2. **State write failure**
   - Exit code: 1
   - Message: `deployment applied, but state file could not be written: <detail>`
   - Guidance: instruct to resolve permissions or re-run with a writable `--state-file`/`--state-file-name`.
3. **OCI pull failure**
   - Exit code: 1
   - Message: `unable to pull Helm chart from OCI: <detail>`
   - CLI should suggest `helm registry login` when 401/403.
4. **Invalid state override**
   - Exit code: 1
   - Message: `state file override is invalid: <detail>`
   - Guidance: prompt operator to choose a valid filename or path before retrying.

## Validation Rules
- `--chart` value must match regex `^oci://[^\s]+:[^\s]+$`.
- When `--output json`, ensure `Content-Type` semantics align (single-line JSON via encoder).
- Namespace validated via existing kube validation helper; create if `--create-namespace` flag added later.
- `--state-file` must point to a writable directory and will create the file with 0600 permissions; `--state-file-name` must pass filesystem-safe naming rules.

## Telemetry
- Emit telemetry phase `helm` with attributes `{ "source": "oci" | "bundle", "namespace": <value>, "action": "install|update" }`.
- On success, log state file path at info level.
