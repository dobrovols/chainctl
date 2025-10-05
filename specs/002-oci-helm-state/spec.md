# Feature Specification: CLI Support for OCI Helm Charts and Persistent State

**Feature Branch**: `002-oci-helm-state`  
**Created**: 2025-10-05  
**Status**: Draft  
**Input**: User description: "need to support helm chart reference in oci url format for install and udate application help charts in addition to local bundle; need ability to specify help release name, application version, k8s namespace to be used as additional options for command line; application should save state after executing command line to the json file: release, namespace, chart ref, version, last action, etc"

## Clarifications
### Session 2025-10-05
- Q: When the CLI completes an install/update but fails to write the JSON state file (for example, directory is read-only), how should it respond? → A: Leave the deployment as-is, report the write error, and require the operator to resolve it manually
- Q: When the JSON state file is missing before an install or update, how should the CLI behave? → A: Automatically recreate the state file during the command before proceeding
- Q: When an operator provides both a local bundle path and an OCI Helm chart reference in the same command, what is the expected behavior? → A: Treat it as an error and ask the operator to choose exactly one source

## User Scenarios & Testing *(mandatory)*

### Primary User Story
As a platform operator, I want the CLI to install or update an application using either a local bundle or an OCI-hosted Helm chart while recording what was deployed, so that I can manage releases consistently across clusters.

### Acceptance Scenarios
1. **Given** an operator provides an OCI registry URL and optional release name, version, and namespace, **When** they run the install action, **Then** the CLI applies the chart from the registry with the provided options and confirms the deployment details.
2. **Given** an operator performs an update action with a different chart reference or version, **When** the command finishes, **Then** the CLI writes a JSON state record capturing the release name, namespace, chart reference, version, last action, and completion timestamp.
3. **Given** an operator supplies custom state file name and/or full path overrides, **When** the command completes, **Then** the CLI writes the state record to the specified location and reflects the override in its confirmation output.

### Edge Cases
- If both a local bundle path and an OCI chart reference are supplied in the same command, the CLI rejects the request and instructs the operator to choose a single source before retrying.
- If writing the state file fails because the directory is read-only or existing contents conflict, the CLI leaves the deployment unchanged, reports the error, and instructs the operator to resolve the issue before retrying.
- If a custom state file override points to an unwritable directory or violates naming rules, the CLI reports the validation error before attempting the Helm action.

## Requirements *(mandatory)*

### Functional Requirements
- **FR-001**: System MUST accept Helm chart references provided as OCI URLs for install and update commands.
- **FR-002**: System MUST continue to accept local bundle inputs when no OCI reference is supplied.
- **FR-003**: System MUST allow operators to override the Helm release name via a command-line option, defaulting to the existing naming convention when unset.
- **FR-004**: System MUST allow operators to specify the application version used for the chart operation via a command-line option.
- **FR-005**: System MUST allow operators to target a Kubernetes namespace via a command-line option, validating that the namespace value is provided before execution.
- **FR-006**: System MUST record the release name, namespace, chart reference, application version, last action performed, and completion timestamp in a JSON state file immediately after each successful install or update.
- **FR-007**: System MUST clearly communicate the location of the updated state file to the operator after each action.
- **FR-008**: System MUST avoid overwriting the previous state file on failed actions and instead present an error indicating that the state is unchanged.
- **FR-009**: System MUST automatically recreate the state file when it is missing before running an install or update and proceed with the requested action.
- **FR-010**: If the state file cannot be written after a successful install or update, the system MUST keep the deployment in place, report the write failure to the operator, and provide guidance to resolve the issue before reattempting.

### Non-Functional Requirements
- **NFR-001**: CLI MUST emit status output within 5 seconds of command start and complete Helm-driven actions within 10 minutes (aligned with P4).
- **NFR-002**: State file creation MUST enforce directory permissions of 0700 and file permissions of 0600, logging any deviation (aligned with P5).
- **NFR-003**: Structured logs and telemetry MUST include chart source type, action, namespace, and state file path (when not sensitive) to support operator observability (aligned with P5).
- **NFR-004**: Overrides MUST be validated without interactive prompts when `--non-interactive` is set, preserving automation workflows (aligned with P3).

### Key Entities *(include if feature involves data)*
- **Chart Reference**: Represents the source of the application chart, including whether it is an OCI URL or local bundle path.
- **Release Configuration**: Captures user-provided overrides such as release name, application version, and namespace targeted by the command.
- **Execution State Record**: JSON artifact storing the most recent action details (release name, namespace, chart reference, version, last action, timestamp) for audit and reuse.
- **State File Configuration**: Captures default directory, override file name, and optional absolute path specified by the operator, ensuring persistence aligns with security constraints.

---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [ ] No implementation details (languages, frameworks, APIs)
- [ ] Focused on user value and business needs
- [ ] Written for non-technical stakeholders
- [ ] All mandatory sections completed

### Requirement Completeness
- [ ] No [NEEDS CLARIFICATION] markers remain
- [ ] Requirements are testable and unambiguous  
- [ ] Success criteria are measurable
- [ ] Scope is clearly bounded
- [ ] Dependencies and assumptions identified

---

## Execution Status
*Updated by main() during processing*

- [ ] User description parsed
- [ ] Key concepts extracted
- [ ] Ambiguities marked
- [ ] User scenarios defined
- [ ] Requirements generated
- [ ] Entities identified
- [ ] Review checklist passed

---
