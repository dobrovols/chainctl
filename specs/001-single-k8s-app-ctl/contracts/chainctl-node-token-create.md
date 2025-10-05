# Contract: `chainctl node token create`

## Synopsis
Create a scoped pre-shared token for joining new nodes.

```
chainctl node token create \
  --role worker|control-plane \
  [--ttl <duration>] \
  [--description <text>] \
  [--output json|text]
```

## Inputs
- `--role` (enum): required scope.
- `--ttl` (duration): optional, default `2h`, max `24h`.
- `--description` (string): optional audit note.
- `--output` (enum): json or text.

## Behavior
1. Generate random secret, hash for storage.
2. Persist token metadata in cluster secret.
3. Print token string to stdout (never logged).
4. Emit telemetry event for audit.

## Outputs
- Text: token value, expiry timestamp, usage instructions.
- JSON: `{ token, tokenID, expiresAt, scope }` (token field only in direct output, not logs).
- Exit codes: `0` success, `60` persistence failure.

## Error Handling
- Storage failure returns `60`.
- TTL exceeding limit results in validation error.

## Observability
- Structured audit log with token ID, role, expiry (without secret).

