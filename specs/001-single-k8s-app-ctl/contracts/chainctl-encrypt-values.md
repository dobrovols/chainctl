# Contract: `chainctl encrypt-values`

## Synopsis
Encrypt a plaintext Helm values file for secure usage in installer workflows.

```
chainctl encrypt-values \
  --input <path> \
  --output <path> \
  [--passphrase <passphrase>] \
  [--confirm] \
  [--output json|text]
```

## Inputs
- `--input` (path): required plaintext YAML.
- `--output` (path): required destination for encrypted file.
- `--passphrase` (string): optional; if omitted prompt twice.
- `--confirm` (bool): skips prompt when overwriting existing output.
- `--output` (enum): json or text.

## Behavior
1. Validate input readability and YAML syntax.
2. Generate random salt and nonce; derive key via scrypt.
3. Encrypt with AES-256-GCM; write binary file with header metadata.
4. Produce checksum entry for bundle manifest when requested.

## Outputs
- Text: success message with checksum, instructions to store passphrase securely.
- JSON: `{ outputPath, checksum, createdAt }`.
- Exit codes: `0` success, `70` validation failure, `71` encryption failure.

## Error Handling
- Input parse error returns `70`.
- Encryption failure returns `71` with remediation steps.
- Passphrase mismatch triggers reprompt (max 3 attempts).

## Observability
- Logs contain only metadata (no secrets) with `phase=encrypt-values`.

