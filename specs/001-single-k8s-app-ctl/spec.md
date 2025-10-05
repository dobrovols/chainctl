# Feature Specification: Single-k8s-app-ctl CLI

**Feature Branch**: `001-single-k8s-app-ctl`  
**Created**: 2025-10-05  
**Status**: Draft  
**Input**: User description: "single-binary CLI to install and upgrade a Kubernetes-based Go micro-services application on Linux (Ubuntu/RHEL), using k3s as the Kubernetes distribution and local-path as the default StorageClass. It can: - bootstrap k3s (optional) or reuse an existing cluster, - install/upgrade the application via Helm, - optionally upgrade k3s using system-upgrade-controller, - work online or air-gapped, - scale from single-node to multi-node (join new nodes)."

## Execution Flow (main)
1. Operator downloads or updates the `chainctl` binary on an Ubuntu or RHEL host and selects whether to bootstrap a new k3s cluster or target an existing one.
2. CLI validates host prerequisites (kernel modules, cgroups, SELinux settings) and gathers configuration inputs such as cluster address, Helm values, encrypted values file path and passphrase, plus the mount point for the air-gapped tarball bundle.
3. When bootstrap is requested, CLI installs k3s with local-path StorageClass and waits for the control plane to become ready.
4. CLI installs or upgrades the micro-services application via Helm, ensuring dependencies and secrets are loaded before the release is applied.
5. Optional flow upgrades the underlying k3s distribution using system-upgrade-controller while preserving application uptime guarantees.
6. CLI can join additional nodes to the cluster, verify they register successfully, and rebalance workloads as needed.
7. Upon completion, CLI produces status, logs, and next-step guidance suitable for both online and disconnected environments.

## ⚡ Quick Guidelines
- Focus on day-1 through day-2 operations for operators who need a dependable installer with rollback awareness.
- Maintain parity between online and air-gapped experiences; no steps may assume internet access by default.
- Ensure command ergonomics follow constitution P3 (noun-verb syntax, dry-run/confirm flags, JSON parity).
- Any k3s automation must respect performance budgets (complete installs under 10 minutes on reference hardware).

### Section Requirements
- Document how cluster bootstrap, application deployment, and upgrade paths interact so planning can assign responsibilities across phases.
- Capture environmental assumptions (Ubuntu 22.04+/RHEL 9+, local-path StorageClass, air-gapped media) needed for task generation and testing.
- Highlight compliance checkpoints tied to constitution principles (test rigor, operational safety) to guide subsequent plans.

### For AI Generation
- Outstanding clarifications must be logged now to unblock planning: 

## Clarifications

### Session 2025-10-05
- Q: How should operators supply the air-gapped artifact bundle when running installer flows offline? → A: Prebuilt tarball on removable media
- Q: What authorization process governs adding new nodes to production clusters through chainctl? → A: Pre-shared join token
- Q: How should operators provide secrets, licenses, and Helm overrides when running chainctl online and offline? → A: Encrypted values file decrypted at runtime

## User Scenarios & Testing *(mandatory)*

### Primary User Story
An operations engineer managing a Go micro-services platform on Ubuntu servers uses `chainctl` to bootstrap or reuse a k3s cluster, deploy the application via Helm, and optionally upgrade both the app and k3s itself, whether connected to the internet or working in an air-gapped facility.

### Acceptance Scenarios
1. **Given** a clean Ubuntu 22.04 host with no Kubernetes components, **When** the operator runs `chainctl cluster install --bootstrap`, **Then** k3s is provisioned with local-path StorageClass and the micro-services release is installed and reported healthy within the supported time budget.
2. **Given** an existing multi-node k3s cluster reachable from the operator’s workstation, **When** they run `chainctl app upgrade --cluster <endpoint> --values <file>`, **Then** the Helm release upgrades in place, preserves user data, and emits JSON-formatted status along with human-readable progress.
3. **Given** a cluster running an older k3s patch version, **When** the operator invokes `chainctl cluster upgrade --controller`, **Then** system-upgrade-controller schedules the upgrade safely, reports per-node progress, and the CLI confirms application availability before exiting.

### Edge Cases
- What happens when the host lacks sufficient CPU/RAM or required kernel modules for k3s? CLI must preflight and provide actionable remediation steps.
- How does the system behave if the Helm upgrade fails mid-apply or exceeds the 10-minute budget? CLI must support rollback and clearly communicate next actions.
- How are conflicting StorageClasses or existing Helm releases handled when reusing a cluster?
- How does the CLI respond if the removable-media tarball is missing images or fails checksum verification?
- What happens when the encrypted values file cannot be decrypted (wrong passphrase or corrupted)?

## Requirements *(mandatory)*

### Functional Requirements
- **FR-001**: System MUST provide a single statically linked binary runnable on Ubuntu 22.04+ and RHEL 9+ without additional package dependencies.
- **FR-002**: System MUST detect whether k3s is present and, if absent and `--bootstrap` is requested, install k3s configured with local-path StorageClass as the default.
- **FR-003**: System MUST validate compatibility (version range, storage class, CPU/RAM) before reusing an existing k3s cluster.
- **FR-004**: System MUST install or upgrade the micro-services application via Helm, ensuring idempotent re-runs and surfacing success/failure signals in both human-readable and JSON formats.
- **FR-005**: System MUST optionally upgrade k3s via system-upgrade-controller while monitoring node drains and confirming post-upgrade workload health.
- **FR-006**: System MUST support air-gapped mode by sourcing binaries, container images, Helm charts, and dependencies from a user-specified tarball mounted from removable media.
- **FR-007**: System MUST support joining additional nodes (worker or control-plane) using pre-shared join tokens with configurable expiration and scope, and verify successful registration and workload scheduling.
- **FR-008**: System MUST collect and surface structured logs, preflight reports, and benchmark metrics required by constitution principles P2–P5.
- **FR-009**: System MUST accept secrets, licenses, and Helm overrides via an encrypted values file on disk, decrypted at runtime with an operator-supplied passphrase for both online and offline executions.

### Key Entities *(include if feature involves data)*
- **Installation Profile**: Describes desired state (bootstrap vs reuse, air-gapped flag, target k3s version, Helm values, encrypted values file reference), informs which flows CLI executes.
- **Target Cluster**: Represents the Kubernetes endpoint (local or remote), stores connection details, compatibility checks, and upgrade history.
- **Air-gapped Artifact Bundle**: Prebuilt tarball containing container images, Helm charts, binaries, and checksums, mounted from removable media for offline execution.
- **Node Enrollment Request**: Metadata for each new node (hostname, role, pre-shared token details) required to join and validate multi-node scaling operations.

---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [ ] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous  
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status
*Updated by main() during processing*

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [ ] Review checklist passed

---
