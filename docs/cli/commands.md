# chainctl CLI Reference

## Global Structure
- `chainctl cluster` – install, upgrade, and validate Kubernetes clusters.
- `chainctl app` – upgrade the Helm-based micro-services release.
- `chainctl node` – manage join tokens and node onboarding.
- `chainctl secrets` – encrypt configuration values.

## Commands
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

### chainctl app upgrade
```
chainctl app upgrade \
  --cluster-endpoint https://cluster.local \
  --values-file values.enc \
  --values-passphrase <passphrase> \
  [--output json]
```
- Helm upgrade using instrumented installer; telemetry spans emitted when exporter enabled.

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
Set `CHAINCTL_OTEL_EXPORTER=stdout|otlp-grpc|otlp-http` to enable telemetry. Instance IDs hashed via `CHAINCTL_CLUSTER_ID` (default hostname).

## Dry-Run Capture
Run `scripts/capture-dry-run.sh` to collect install/upgrade outputs in `artifacts/dry-run/`, attach files to pull requests for reviewer context.

