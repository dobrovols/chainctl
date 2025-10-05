# Quickstart: chainctl Installer CLI

## Prerequisites
- Ubuntu 22.04 or RHEL 9 host with sudo access.
- `chainctl` binary downloaded and marked executable.
- Encrypted values file created with `chainctl encrypt-values` (passphrase available).
- Removable-media tarball mounted at `/opt/chainctl/bundles/<version>` when offline.
- (Optional) OpenTelemetry exporter configured via `CHAINCTL_OTEL_EXPORTER=stdout|otlp-grpc|otlp-http`.

## Bootstrap & Install (Air-gapped)
1. Mount the bundle:
   ```bash
   sudo mount /dev/sdb1 /opt/chainctl/bundles/v1
   ```
2. Decrypt values when prompted:
   ```bash
   sudo ./chainctl cluster install \
     --bootstrap \
     --airgapped \
     --bundle-path /opt/chainctl/bundles/v1 \
     --values-file /opt/secure/app-values.enc \
     --values-passphrase "$CHAINCTL_VALUES_PASSPHRASE" \
     --output text
   ```
3. Validate success via JSON report (optional):
   ```bash
   ./chainctl cluster install --dry-run --output json | jq .
   ```
4. Store dry-run artifacts for review:
   ```bash
   scripts/capture-dry-run.sh
   ```

## Application Upgrade (Online)
1. Ensure kubeconfig points to target cluster.
2. Run upgrade with rendered diff:
   ```bash
   ./chainctl app upgrade \
     --cluster-endpoint https://cluster.local:6443 \
     --values-file ./app-values.enc \
     --values-passphrase "$CHAINCTL_VALUES_PASSPHRASE" \
     --output text
   ```
3. Review JSON telemetry for auditing:
   ```bash
   ./chainctl app upgrade --dry-run --output json > upgrade-plan.json
   ```

## Joining a Worker Node
1. Create scoped token on controller host:
   ```bash
   ./chainctl node token create --role worker --ttl 4h --output json > token.json
   ```
2. On the new node, join cluster:
   ```bash
   sudo ./chainctl node join \
     --cluster-endpoint https://cluster.local:6443 \
     --role worker \
     --token $(jq -r .token token.json) \
     --labels env=prod,region=us-east
   ```
3. Confirm readiness:
   ```bash
   kubectl get nodes
   ```

## Cluster Upgrade via System Upgrade Controller
1. Mount bundle or ensure internet access to fetch charts.
2. Launch upgrade:
   ```bash
   ./chainctl cluster upgrade \
     --cluster-endpoint https://cluster.local:6443 \
     --k3s-version v1.30.2+k3s1 \
     --bundle-path /opt/chainctl/bundles/v1 \
     --airgapped \
     --drain-timeout 20m \
     --output text
   ```
3. Monitor progress via logs or JSON output. The submitted plan is available as `system-upgrade/chainctl-upgrade`.

## Cleanup & Logs
- Logs written to `~/.chainctl/logs/<timestamp>.jsonl`.
- Set `CHAINCTL_CLUSTER_ID` to hash telemetry per cluster.
- Use `./chainctl support bundle` (future feature) to collect diagnostics if failures occur.
