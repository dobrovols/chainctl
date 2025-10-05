# Research Log: chainctl Single-Binary Installer CLI

## k3s Bootstrap Strategy
- **Decision**: Use k3s install script via `INSTALL_K3S_EXEC` with explicit `--cluster-init` and local-path StorageClass configuration, wrapped by Go helpers.
- **Rationale**: Aligns with upstream k3s automation, minimizes maintenance while allowing idempotent reruns; local-path is default SC per requirements.
- **Alternatives Considered**:
  - Custom k3s binaries + manual systemd units (rejected: higher maintenance, brittle upgrades).
  - Relying on distro packages (rejected: inconsistent versions across Ubuntu/RHEL).

## Air-gapped Bundle Mounting
- **Decision**: Expect operators to mount a removable-media tarball under `/opt/chainctl/bundles/<version>` with manifest file.
- **Rationale**: Keeps offline assets immutable, simplifies checksum validation, works for USB or ISO media.
- **Alternatives Considered**:
  - On-the-fly OCI registry mirroring (rejected: added complexity, conflicts with offline requirement).
  - Self-extracting binary (rejected: large binary size, complicated updates).

## Encrypted Values Handling
- **Decision**: Require AES-256-GCM encrypted YAML using `chainctl encrypt-values` command, decrypt passphrase captured via `--values-passphrase` or prompt.
- **Rationale**: Provides strong confidentiality, can be automated via env var injection; matches CLI ergonomics.
- **Alternatives Considered**:
  - PGP-encrypted files (rejected: require external tooling, key management overhead).
  - Plaintext with file permissions (rejected: violates security expectations).

## Pre-shared Join Token Lifecycle
- **Decision**: Generate tokens via `chainctl node token create --ttl <duration> --scope worker|control-plane`, store hashed token on controller, rotate after use.
- **Rationale**: Supports automation, scoped least privilege, enforce expiry and revocation.
- **Alternatives Considered**:
  - Manual approval workflow (rejected: slower scaling, harder for air-gapped sites).
  - Long-lived static tokens (rejected: security risk).

## System Upgrade Controller Usage
- **Decision**: Deploy system-upgrade-controller via Helm chart from bundle; upgrades orchestrated with surge=1, drain timeout configurable, health checks gating completion.
- **Rationale**: Controller already proven for k3s upgrades; configuration matches availability expectations.
- **Alternatives Considered**:
  - Direct k3supgrade CLI per node (rejected: reinvents scheduling, harder rollback).
  - Cluster API integration (rejected: scope creep).

## Performance & Telemetry Verification
- **Decision**: Instrument critical phases with OpenTelemetry metrics/logs, add Go benchmarks for installer workflows using fake clients, record runtime metrics in JSON output.
- **Rationale**: Required by constitution P4/P5, provides automation-friendly observability.
- **Alternatives Considered**:
  - Manual timing via shell scripts (rejected: flaky, not CI-friendly).
  - Ignoring telemetry (rejected: violates P5).

