# Contract: `chainctl node join`

## Synopsis
Provision join configuration for a new node and verify successful registration.

```
chainctl node join \
  --cluster-endpoint <url> \
  --role worker|control-plane \
  --token <token> \
  [--labels key=value,...] \
  [--taints key=value:effect,...] \
  [--output json|text]
```

## Inputs
- `--cluster-endpoint` (string): required.
- `--role` (enum): required; determines scope validation.
- `--token` (string): required pre-shared join token.
- `--labels` (list): optional; validated as key=value.
- `--taints` (list): optional; validated as key=value:effect.
- `--output` (enum): json or text.

## Preconditions
- Token exists, unexpired, scope matches requested role.
- Node system prerequisites satisfied (validated by local script executed remotely or instructions displayed).

## Behavior
1. Validate token with controller (hashed comparison, TTL check).
2. Generate kube join command or service file for node.
3. Execute join (if run on node) or output instructions (if run centrally with `--output-script`).
4. Monitor Kubernetes node status until Ready.
5. Emit telemetry envelope with join duration.

## Outputs
- Text: success message with node name, labels, next steps.
- JSON: `{ nodeName, role, durationSeconds, taints, labels }`.
- Exit codes: `0` success, `50` token invalid, `51` node join failure, `52` timeout.

## Error Handling
- Invalid token returns `50` with reason.
- Join timeout triggers instructions to collect diagnostics.

## Observability
- Logs with `phase=join` and hashed node identifier.

