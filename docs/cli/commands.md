# chainctl CLI Reference

## Global Structure
- `chainctl cluster` – install, upgrade, and validate Kubernetes clusters.
- `chainctl app` – install or upgrade the Helm-based application release.
- `chainctl node` – manage join tokens and node onboarding.
- `chainctl secrets` – encrypt configuration values.

## Commands
### chainctl app install
```
chainctl app install \
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
- Exactly one of `--chart` (OCI reference) or `--bundle-path` (air-gapped assets) must be supplied. `--state-file` and `--state-file-name` are mutually exclusive.
- Namespace and release defaults are pulled from the profile; flags allow explicit overrides for multi-tenant clusters.
- State is written to the XDG config directory (`$XDG_CONFIG_HOME/chainctl/state/app.json` by default) unless `--state-file` or `--state-file-name` are provided.
- JSON output includes `status`, `action`, `release`, `namespace`, `chart`, `stateFile`, and `timestamp` fields.

### chainctl app upgrade
```
chainctl app upgrade \
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
- Helm upgrade is driven through the resolver: OCI charts are pulled with digest capture; bundle mode reuses local assets.
- CLI rejects conflicting sources, invalid OCI references, and invalid state-file paths before contacting the cluster. Namespace can be supplied via flags or profile.
- On success, state is persisted atomically (0600 file, 0700 directories) and the final path is echoed to the operator.
- JSON output adds `action: "upgrade"` and reuses install fields for parity.

### chainctl cluster install
```
chainctl cluster install \
  [--bootstrap] \
  [--cluster-endpoint https://cluster.local] \
  --values-file /path/to/values.enc \
  --values-passphrase <passphrase> \
  [--bundle-path /mnt/bundle] \
  [--dry-run] \
  [--output json]
```
- Host preflight (CPU, memory, `br_netfilter`, sudo) enforced.
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
