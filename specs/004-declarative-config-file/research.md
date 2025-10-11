# Phase 0 Research: Declarative YAML Configuration

## Decision: Represent YAML with `defaults` plus per-command sections
- **Rationale**: A top-level `defaults` map establishes shared flag values, while a `commands` map keyed by command path (e.g., `cluster install`) captures command-specific overrides, directly supporting FR-007 and clarifying operator expectations from Question 2.
- **Alternatives Considered**:
  - Separate YAML file per command: rejected to avoid configuration sprawl and support shared baselines.
  - Deeply nested environment-level hierarchy: rejected as unnecessary complexity for current scope.

## Decision: Reuse Cobra flag definitions as schema source
- **Rationale**: Leveraging existing `cmd/chainctl/app` flag definitions ensures validation stays aligned with live CLI options, lowers duplication risk, and enables automatic detection of unsupported keys per FR-002.
- **Alternatives Considered**:
  - Maintaining independent schema definitions: rejected because it would drift from code and require double maintenance.
  - Generating schema from documentation: rejected due to incomplete coverage and lack of type fidelity.

## Decision: Parse YAML via `gopkg.in/yaml.v3` with in-memory caching
- **Rationale**: The project already uses the Go toolchain; `yaml.v3` offers mature support for strict decoding and helpful error messages. Caching parsed files per path for the life of a process keeps parsing under the 200ms target and avoids repeated filesystem reads.
- **Alternatives Considered**:
  - Using `sigs.k8s.io/yaml`: rejected because we need strict duplicate key detection and custom error contexts not readily available.
  - Persisting a compiled JSON representation on disk: rejected to prevent stale cache files and additional IO management.

## Decision: Enforce secret-free configs via validation hook
- **Rationale**: Secret prohibition (Question 3) is implemented by scanning declared flag names against known sensitive categories (e.g., token, password) and rejecting files containing them, guiding operators to external secret stores in accordance with FR-005.
- **Alternatives Considered**:
  - Allowing secrets with log redaction flags: rejected due to governance requirements and higher leak risk.
  - Relying on documentation only: rejected because automated enforcement provides consistent safety.
