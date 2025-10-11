# chainctl CLI Reference

## Global Structure
- `chainctl cluster` – install, upgrade, and validate Kubernetes clusters.
- `chainctl app` – install or upgrade the Helm-based application release.
- `chainctl node` – manage join tokens and node onboarding.
- `chainctl secrets` – encrypt configuration values.

## Declarative Configuration
- `--config` accepts a YAML file describing shared defaults, reusable profiles, and per-command flag overrides.
- Discovery precedence: explicit `--config` path → `CHAINCTL_CONFIG` → `./chainctl.yaml` → `$XDG_CONFIG_HOME/chainctl/config.yaml` → `$HOME/.config/chainctl/config.yaml`.
- YAML must not contain secrets (`values-passphrase`, tokens, kubeconfigs); provide sensitive values via runtime flags or secret stores.
- Before command execution, chainctl prints a summary showing each flag, effective value, and source (default, profile, command, runtime) in text or JSON format to allow validation and audit logging.

## Commands
### chainctl app install
```
chainctl app install \
  [--config chainctl.yaml] \
  --values-file values.enc \
  --values-passphrase <passphrase> \
  --namespace demo \
  [--chart oci://registry.example.com/apps/myapp:1.2.3] \
  [--bundle-path /mnt/app-bundle] \
  [--release-name myapp-demo] \
  [--app-version 1.2.3] \
  [--state-file /var/lib/chainctl/state.json] \
  [--state-file-name app.json] \
  [--output json]
```
- Declarative configs can provide defaults for namespace, release name, bundle paths, and chart references. Runtime flags always override YAML values.
- Exactly one of `--chart` (OCI reference) or `--bundle-path` (air-gapped assets) must be supplied. `--state-file` and `--state-file-name` are mutually exclusive.
- Namespace and release defaults are pulled from the profile; flags allow explicit overrides for multi-tenant clusters.
- State is written to the XDG config directory (`$XDG_CONFIG_HOME/chainctl/state/app.json` by default) unless `--state-file` or `--state-file-name` are provided.
- JSON output includes `status`, `action`, `release`, `namespace`, `chart`, `stateFile`, and `timestamp` fields.

### chainctl app upgrade
```
chainctl app upgrade \
  [--config chainctl.yaml] \
  --cluster-endpoint https://cluster.local \
  --values-file values.enc \
  --values-passphrase <passphrase> \
  [--chart oci://registry.example.com/apps/myapp:1.2.4] \
  [--bundle-path /mnt/app-bundle] \
  [--release-name myapp-demo] \
  [--app-version 1.2.4] \
  [--namespace demo] \
  [--state-file /var/lib/chainctl/state.json] \
  [--state-file-name app.json] \
  [--output json]
```
- Declarative configs can specify staging profiles (e.g., namespace overrides) and command-specific defaults; runtime flags can still override individual values.
- Helm upgrade is driven through the resolver: OCI charts are pulled with digest capture; bundle mode reuses local assets.
- CLI rejects conflicting sources, invalid OCI references, and invalid state-file paths before contacting the cluster. Namespace can be supplied via flags or profile.
- On success, state is persisted atomically (0600 file, 0700 directories) and the final path is echoed to the operator.
- JSON output adds `action: "upgrade"` and reuses install fields for parity.

### chainctl cluster install
```
chainctl cluster install \
  [--config chainctl.yaml] \
  [--bootstrap] \
  [--cluster-endpoint https://cluster.local] \
  --values-file /path/to/values.enc \
  --values-passphrase <passphrase> \
  [--bundle-path /mnt/bundle] \
  [--dry-run] \
  [--output json]
```
- Declarative configs can declare shared defaults (namespace, bundle path, dry-run mode) and per-command overrides. YAML discovery summary prints before host validation begins.
- Host preflight (CPU, memory, `br_netfilter`, `overlay`, sudo) enforced.
- Reuse mode loads kubeconfig and validates cluster connectivity.
- Dry-run returns immediately after validations, logging to `artifacts/dry-run/` via script.

### chainctl cluster upgrade
```
chainctl cluster upgrade \
  --cluster-endpoint https://cluster.local \
  --k3s-version v1.30.2+k3s1 \
  [--controller-manifest manifest.yaml] \
  [--bundle-path /mnt/bundle]
```
- Ensures system-upgrade-controller namespace/CRDs.
- Submits plan `system-upgrade/chainctl-upgrade` with target version.
- Supports text or JSON output for plan status.

### chainctl node token
```
chainctl node token create --role worker --ttl 4h --output json
```
- TTL capped at 24h, returns `{ token, tokenID, expiresAt }`.

### chainctl node join
```
chainctl node join \
  --cluster-endpoint https://cluster.local \
  --role worker \
  --token <id.secret> \
  [--output json]
```
- Dry-run friendly; validates token scope/expiry.

### chainctl secrets encrypt-values
```
chainctl secrets encrypt-values \
  --input values.yaml \
  --output values.enc \
  [--passphrase ...] \
  [--format json]
```
- AES-256-GCM with checksum output; passphrase prompt if omitted.

## Telemetry
Set `CHAINCTL_OTEL_EXPORTER=stdout|otlp-grpc|otlp-http` to enable telemetry. New Helm flows emit metadata for `source`, `namespace`, and chart digests alongside phase start/stop events. Instance IDs hashed via `CHAINCTL_CLUSTER_ID` (default hostname).

## State Persistence Summary
- Default path: `$XDG_CONFIG_HOME/chainctl/state/app.json` or `$HOME/.chainctl/state/app.json`.
- `--state-file-name` customises the filename while keeping the managed directory.
- `--state-file` accepts an absolute path; directories are created with 0700 permissions and files saved atomically with 0600 permissions.

## Dry-Run Capture
Run `scripts/capture-dry-run.sh` to collect install/upgrade outputs in `artifacts/dry-run/`, attach files to pull requests for reviewer context.
