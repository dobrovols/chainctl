# Data Model: chainctl Installer CLI

## InstallationProfile
- **Fields**
  - `Mode` (`enum: bootstrap|reuse`)
  - `ClusterEndpoint` (`string`, optional when bootstrap) – validated URL/IP.
  - `K3sVersion` (`semver`) – must fall within supported window 1.28–1.30.
  - `AirgappedBundlePath` (`filepath`) – absolute mount point, must exist/readable.
  - `EncryptedValuesPath` (`filepath`) – must exist; validated via checksum manifest.
  - `ValuesPassphrase` (`[]byte`, write-only) – never persisted, only held in memory.
  - `HelmReleaseName` (`string`) – defaults to `chainapp` if empty.
  - `HelmNamespace` (`string`) – must be DNS-compliant; default `chain-system`.
  - `UpgradeStrategy` (`UpgradePlan`, optional) – required when performing upgrades.
- **Relationships**
  - References `BundleManifest` for offline assets when `Airgapped`.
  - References `SecretPayload` for decrypted values at runtime.
- **Validation Rules**
  - `AirgappedBundlePath` required when offline flag set.
  - `ValuesPassphrase` must decrypt file before use; failure aborts run.
  - When `Mode=bootstrap`, `ClusterEndpoint` derived post-install.

## BundleManifest
- **Fields**
  - `Version` (`semver`)
  - `Images` (`[]ImageRecord`)
  - `HelmCharts` (`[]ChartRecord`)
  - `Binaries` (`[]BinaryRecord`)
  - `Checksums` (`map[string]string`)
- **Relationships**
  - Bound to `InstallationProfile.AirgappedBundlePath`.
- **Validation Rules**
  - Checksums verified before extraction.
  - Missing images trigger rollback path.

## SecretPayload
- **Fields**
  - `DecryptedValues` (`map[string]any`)
  - `SourceFile` (`filepath`)
  - `Timestamp` (`time.Time`)
- **Lifecycle**
  - Created on decrypt, wiped on completion or error.
  - Never written to disk.

## TokenRecord
- **Fields**
  - `TokenID` (`uuid`)
  - `Scope` (`enum: worker|control-plane`)
  - `ExpiresAt` (`time.Time`)
  - `HashedSecret` (`[]byte`)
  - `CreatedBy` (`string`, operator ID)
- **Relationships**
  - Associated with `NodeEnrollmentRequest`.
- **Validation Rules**
  - TTL max 24h; expired tokens not accepted.
  - One-time use enforced by mark consumed flag.

## NodeEnrollmentRequest
- **Fields**
  - `Hostname` (`string`)
  - `Role` (`enum: worker|control-plane`)
  - `JoinToken` (`string`, transient)
  - `Labels` (`map[string]string`)
  - `Taints` (`[]corev1.Taint`)
- **Lifecycle**
  - Created when operator runs `chainctl node join`.
  - Validated against `TokenRecord` before kube join.

## UpgradePlan
- **Fields**
  - `TargetK3sVersion` (`semver`)
  - `AppChartVersion` (`semver`)
  - `SystemUpgradeSettings` (`UpgradeSettings`)
  - `HealthChecks` (`[]HealthCheck`)
- **Validation Rules**
  - Cannot downgrade versions.
  - `HealthChecks` must include deploy and workload probes.

## TelemetryEnvelope
- **Fields**
  - `Phase` (`enum: preflight|bootstrap|helm|upgrade|join|verify`)
  - `Duration` (`time.Duration`)
  - `Outcome` (`enum: success|warning|failure`)
  - `ClusterID` (`string`, hashed)
  - `Metadata` (`map[string]string`)
- **Usage**
  - Emitted for P5 observability compliance, serialised to JSON logs.

