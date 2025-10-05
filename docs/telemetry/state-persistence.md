# Telemetry Notes: Application Install/Upgrade

## Signals
- **Event stream**: `telemetry.Event` emitted for phase `helm` with:
  - `metadata.source`: `oci` or `bundle` depending on resolver outcome.
  - `metadata.namespace`: Helm namespace targeted by the command.
  - `metadata.digest`: OCI chart digest when available.
- **Success JSON output** includes the persisted `stateFile` path for audit pipelines.

## Recommended Collection
1. Set `CHAINCTL_OTEL_EXPORTER=stdout` during dry-run to capture structured events alongside CLI output.
2. When shipping to OTLP collectors, tag traces/metrics with `CHAINCTL_CLUSTER_ID` for anonymised aggregation.
3. Preserve the emitted state record (`state.Record`) for post-upgrade verification or rollbacks.

## Example
```
CHAINCTL_OTEL_EXPORTER=stdout chainctl app upgrade \
  --cluster-endpoint https://cluster.local \
  --values-file values.enc \
  --values-passphrase <pass> \
  --chart oci://registry.example.com/apps/myapp:1.2.3 \
  --namespace demo \
  --output json
```
Sample stdout telemetry event:
```
{"timestamp":"2025-10-05T18:43:21.349425Z","phase":"helm","outcome":"success","duration":6708,"metadata":{"mode":"reuse","namespace":"demo","source":"oci","digest":"sha256:abc123"}}
```

## Alerting Tips
- Trigger alerts when `outcome` is `failure` for phase `helm`.
- Watch for repeated state write failures (CLI exits non-zero with `state file could not be written`).
- Record the `stateFile` path from JSON output for downstream auditing pipelines.
