# Implementation Plan: Declarative YAML Configuration for CLI Flags

**Branch**: `004-declarative-config-file` | **Date**: 2025-10-10 | **Spec**: specs/004-declarative-config-file/spec.md  
**Input**: Feature specification from `/specs/004-declarative-config-file/spec.md`

## Summary
Declarative configuration lets operators describe complete `chainctl` command invocations in YAML, enforce precedence between configuration sources, and remove secrets from shared artifacts. The plan introduces a reusable configuration loader, command-level wiring to consume the resolved flag set, validation feedback for unsupported keys, and documentation that guides teams on adopting YAML-driven workflows without breaking existing flag-based usage.

## Technical Context
**Language/Version**: Go 1.24  
**Primary Dependencies**: `spf13/cobra`, `helm.sh/helm/v3`, internal `pkg/state`, `pkg/telemetry`, `internal/config`, YAML parsing via `gopkg.in/yaml.v3`  
**Storage**: Local filesystem YAML files under repo or XDG config directories (read-only at runtime)  
**Testing**: `go test ./pkg/... ./cmd/...`, envtest-based flows in `test/integration`, future e2e in `make test-e2e` covering declarative config runs  
**Target Platform**: cross-platform CLI (macOS/Linux) executed by cluster operators  
**Project Type**: single (monolithic CLI with shared packages)  
**Performance Goals**: Config discovery and parsing MUST complete in <200ms per invocation and avoid new allocations that push process memory beyond existing 512MB ceiling.  
**Constraints**: YAML MUST exclude secrets, obey precedence `CHAINCTL_CONFIG` → `./chainctl.yaml` → `$XDG_CONFIG_HOME/chainctl/config.yaml` → `~/.config/chainctl/config.yaml`, align with structured logging, preserve existing flag UX.  
**Scale/Scope**: Must support all current `chainctl` commands (install/upgrade/app flows) with tens of flags per command and shared defaults for multiple operator teams.

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **P1 · Go Craftsmanship**: All new Go code will live in `internal/config` for discovery and a new `pkg/config` facade for consumers, both covered by gofmt/gofumpt, golangci-lint, and clear exported contracts. Plan includes docstrings outlining expected error semantics and package boundaries.
- **P2 · Test Rigor**: Unit tests cover YAML parsing, precedence resolution, secret rejection, and CLI flag merging. Integration tests in `test/integration` simulate running `chainctl` commands with configuration files. E2E smoke test scenario is scheduled to validate `chainctl cluster install --config` with dry-run to ensure idempotency.
- **P3 · Operator UX**: CLI updates add a `--config` flag consistent with noun-verb structure, keep `--dry-run`/`--confirm` behavior, surface JSON parity via `--output json`, and document precedence plus error messages for unrecognized keys.
- **P4 · Performance Budgets**: Loader caches parsed YAML per path to avoid repeated IO, limits file size (<256KB warning) and ensures no blocking operations delay startup. Benchmarks will measure load latency to confirm <200ms goal.
- **P5 · Operational Safety**: Loader emits structured logs for discovery steps, ensures secrets are never accepted, integrates telemetry timing, and provides runbook updates explaining fallback hierarchy and remediation when configs fail validation.

## Project Structure

### Documentation (this feature)
```
specs/004-declarative-config-file/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
cmd/chainctl/
├── app/
│   ├── config_flag.go           # NEW: adds --config wiring and validation hooks
│   ├── install.go
│   ├── flags.go
│   └── ...
├── cluster/
│   └── install/
│       └── command.go           # UPDATED: consume resolved configuration

internal/
├── config/
│   ├── locator.go               # NEW: discovery order honoring env/working directory/XDG
│   ├── loader.go                # NEW: YAML parsing, validation, caching
│   └── loader_test.go
├── validation/
│   └── schema.go                # UPDATED: share flag validation helpers

pkg/
├── config/
│   ├── profile.go               # NEW: exported structs for configuration profiles
│   ├── resolver.go              # NEW: merge YAML with CLI overrides
│   └── resolver_test.go
├── state/
└── telemetry/

test/
├── unit/
│   └── config_loader_test.go    # NEW: direct loader coverage
├── integration/
│   └── declarative_config_test.go  # NEW: command-level scenario
└── e2e/
    └── declarative_config_workflow_test.go  # NEW: dry-run install via YAML
```

**Structure Decision**: Extend existing CLI architecture by introducing `pkg/config` as the public interface for declarative configuration while using `internal/config` for environment-specific discovery. Command packages call into `pkg/config` to obtain resolved flag maps, preserving current modular boundaries without spreading filesystem logic through command handlers.

## Phase 0: Outline & Research
1. Inventory unknowns from Technical Context and spec: YAML schema structure, existing flag metadata availability, best practices for precedence resolution, validation strategy for unsupported keys, and caching lifetime.
2. Research tasks to capture in `research.md`:
   - Document how current `cmd/chainctl/app/flags.go` defines flag metadata and determine reuse for schema validation.
   - Explore `internal/config` existing helpers to avoid duplication and decide on new abstractions.
   - Evaluate YAML vs. JSON schema validation approaches (choose YAML with structural validation).
   - Establish caching approach (in-memory per command invocation vs. persistent).
3. Each research entry records Decision, Rationale, Alternatives and resolves remaining NEEDS CLARIFICATION.

## Phase 1: Design & Contracts
1. `data-model.md` will define entities: `ConfigurationProfile`, `CommandSection`, `FlagValue`, `ResolvedInvocation`, including relationships and validation rules (no secret values, type enforcement).
2. `/contracts/` will contain `config-schema.yaml` describing allowable YAML structure plus `validation.adoc` summarizing error responses expected by operators.
3. `quickstart.md` will outline how to create `chainctl.yaml`, specify precedence behavior, and include verification steps with `--dry-run`.
4. Design integration tests:
   - Unit tests for loader/resolver.
   - Integration test verifying CLI merges YAML with `--set flag` and outputs both human-readable and JSON summaries.
   - E2E dry-run ensures safe application while exercising multi-command configurations with reusable profiles.
5. Run `.specify/scripts/bash/update-agent-context.sh codex` to register the feature context for future AI assistance.

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Use `.specify/templates/tasks-template.md` as base.
- Derive tasks from research, data model, and contracts: create implementation tasks for loader/resolver, command wiring, tests, and documentation.
- Tag parallelizable tasks ([P]) such as independent unit tests or documentation updates.

**Ordering Strategy**:
- Start with data/model definitions, then loader/resolver implementation, followed by CLI integration, integration tests, and documentation updates.
- Maintain TDD cadence: write failing tests before implementation, ensure benchmarks/logging tasks after core logic.

**Estimated Output**: 25–30 ordered tasks in `tasks.md`.

## Phase 3+: Future Implementation
**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks)  
**Phase 5**: Validation (run gofmt/gofumpt, go test suites, quickstart, performance checks)

## Complexity Tracking
*No deviations anticipated; keep empty unless constraints change.*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|---------------------------------------|

## Progress Tracking
*This checklist is updated during execution flow*

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
*Based on Constitution v1.2.0 - See `/memory/constitution.md`*
