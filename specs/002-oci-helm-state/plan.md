# Implementation Plan: CLI Support for OCI Helm Charts and Persistent State

**Branch**: `002-oci-helm-state` | **Date**: 2025-10-05 | **Spec**: specs/002-oci-helm-state/spec.md  
**Input**: Feature specification from `/specs/002-oci-helm-state/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (CLI built in Go)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file.
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

## Summary
Enable `chainctl app` commands to consume Helm charts from OCI registries in addition to local bundles, expose release/version/namespace flags, and persist post-action deployment state in a JSON file with operator-controlled file name or full-path overrides for auditing and subsequent executions.

## Technical Context
**Language/Version**: Go 1.24  
**Primary Dependencies**: Cobra CLI (`spf13/cobra`), Helm SDK (`helm.sh/helm/v3`), existing `pkg/bundle`, `pkg/helm`, `pkg/telemetry`  
**Storage**: Local filesystem JSON state file under user config directory with optional filename/path overrides  
**Testing**: Go `testing` with gomock/fakes, envtest/kind integration harness (existing `test/integration`), future e2e make target  
**Target Platform**: Operator workstations (Linux/macOS) interacting with Kubernetes clusters (v1.28–1.30)  
**Project Type**: CLI with layered pkg/internal structure  
**Performance Goals**: Helm install/update completes within 10 minutes; CLI feedback within 5 seconds per P4  
**Constraints**: Maintain <512 MB memory footprint, support `--dry-run`/`--confirm`, namespace scoping, JSON output parity  
**Scale/Scope**: Single application release per invocation; supports clusters up to dozens of namespaces per run

## Constitution Check
- **P1 · Go Craftsmanship**: Plan adds cohesive packages (`pkg/state`, optional `internal/state`) with gofmt/gofumpt + golangci-lint gating; new exported APIs return wrapped errors and include interface seams for testing. PASS
- **P2 · Test Rigor**: Defines unit tests for CLI flag parsing, Helm installer adapter, and state persistence; integration tests via envtest/kind cover OCI pull + state file writes; e2e smoke to ensure repeated install/update idempotency. PASS
- **P3 · Operator UX**: Extends `chainctl app upgrade` (and future install) with noun-verb alignment, namespace flag, JSON output parity, and explicit error messaging when conflicting sources or state-writes fail; keep `--dry-run`/`--confirm` hooks ready. PASS
- **P4 · Performance Budgets**: Uses Helm OCI pull caching and avoids redundant downloads; state writes buffered to single JSON dump; telemetry timers validate sub-10 minute runtime. PASS
- **P5 · Operational Safety**: Emits structured logs around chart resolution/state persistence, updates telemetry metrics, documents rollback guidance, and ensures state recreation/retention adheres to secure file permissions. PASS

## Project Structure

### Documentation (this feature)
```
specs/002-oci-helm-state/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── app-upgrade-cli.md
│   └── state-schema.json
└── tasks.md   (planned via /tasks)
```

### Source Code (repository root)
```
cmd/chainctl/
└── app/
    ├── app.go
    ├── install.go            # new (installs using OCI/local bundle)
    └── upgrade.go            # updated flags, shared workflow

pkg/
├── helm/
│   ├── client.go             # extend for OCI chart references
│   └── resolver.go           # new helper for OCI/local source resolution
├── bundle/
├── state/                    # new package for JSON persistence
└── upgrade/

internal/
├── config/
├── validation/
├── kubeclient/
└── state/                    # optional helpers for locating state directories

test/
├── unit/app/                 # new CLI flag/state tests
├── integration/app/          # envtest/kind scripts for OCI pull + state write
└── fixtures/state/           # sample state JSONs
```

**Structure Decision**: Extend existing CLI/app modules while introducing `pkg/state` for reusable persistence and a Helm resolver abstraction to keep `cmd` layer thin; integration tests live under `test/` aligned with constitution’s multi-tier coverage.

## Phase 0: Outline & Research
- Investigated Helm SDK support for OCI (`helm.sh/helm/v3/pkg/action` & `registry`), confirming login/pull flows and caching.
- Determined operator-friendly location for state file using XDG config directory fallback to `$HOME/.chainctl/state.json` with `0700` permissions.
- Reviewed existing `pkg/bundle` interactions to ensure mutually exclusive bundle/OCI selection with clear error messages.

Output written to `research.md` summarizing decisions, rationale, and alternatives.

## Phase 1: Design & Contracts
- Derived data entities (ChartSource, ReleaseOptions, ExecutionStateRecord, StateFileConfig) in `data-model.md` including validation rules and relationships.
- Authored CLI contract `contracts/app-upgrade-cli.md` capturing command flags, behaviors, and error conditions; defined JSON schema in `contracts/state-schema.json` for persistent state.
- Documented quickstart manual flow in `quickstart.md` covering install/update, state validation, and error recovery.
- Updated agent context via `.specify/scripts/bash/update-agent-context.sh codex` with new technologies (Helm OCI, state persistence).

## Phase 2: Task Planning Approach
**Task Generation Strategy**:
- Use `tasks-template.md` to enumerate tasks from contracts and data model: CLI flag parsing (including state override options) → implementation, Helm resolver tests → implementation, state persistence tests → implementation, integration/e2e runs.
- Mark parallelizable items (`pkg/state` logic vs. CLI wiring) with `[P]`.

**Ordering Strategy**:
1. Establish state persistence tests and schema validations.
2. Implement Helm resolver with unit tests.
3. Wire CLI flags and behaviors, then update telemetry/logging.
4. Add integration/e2e validations and documentation updates.

**Estimated Output**: ~24 ordered tasks with dependency annotations.

## Phase 3+: Future Implementation
- Phase 3 (`/tasks`): Generate actionable tasks with explicit test-first ordering.
- Phase 4: Execute tasks ensuring lint/tests/benchmarks per constitution.
- Phase 5: Validate via unit/integration/e2e plus quickstart walkthrough before PR.

## Complexity Tracking
| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| *None* | | |

## Progress Tracking
**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [ ] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [ ] Complexity deviations documented

---
*Based on Constitution v1.1.0 - See `/memory/constitution.md`*
