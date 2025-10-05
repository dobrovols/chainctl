# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → If not found: ERROR "No implementation plan found"
   → Extract: Go modules, packages, structure, constitutional obligations
2. Load optional design documents:
   → data-model.md: Extract entities → installer/service tasks
   → contracts/: Each file → contract/e2e test task
   → research.md: Extract decisions → setup/benchmark tasks
3. Generate tasks by category:
   → Setup: module scaffolding, tooling, linting
   → Tests: unit (pkg/internal), integration (envtest/kind), e2e CLI flows
   → Core: installers, kube clients, telemetry, CLI commands
   → Integration: performance benchmarks, logging, rollout/rollback
   → Polish: docs, runbooks, UX assets
4. Apply task rules:
   → Different files/packages = mark [P] for parallel
   → Same file/package = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness:
   → All contracts have tests?
   → All principles represented?
   → Benchmarks + dry-run artifacts captured?
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- CLI entrypoints live in `cmd/chainctl/...`
- Shared logic lives in `pkg/...` or `internal/...`
- Tests live in `test/unit`, `test/integration`, and `test/e2e`
- Benchmarks live alongside the package under test (e.g., `pkg/installer/installer_bench_test.go`)

## Phase 3.1: Setup
- [ ] T001 Ensure go.mod/go.sum reflect new dependencies; run `go mod tidy`
- [ ] T002 Configure golangci-lint and gofumpt pipelines; update `Makefile` or CI if needed
- [ ] T003 [P] Scaffold package and directory structure per plan (e.g., `pkg/installer`, `cmd/chainctl/install`)

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**
- [ ] T004 [P] Unit tests in `test/unit/<feature>_test.go` covering pure functions and error paths
- [ ] T005 [P] Integration tests in `test/integration/<feature>_test.go` using envtest/kind to exercise kube interactions
- [ ] T006 [P] CLI e2e test in `test/e2e/<command>_test.go` validating `--dry-run`, `--confirm`, and idempotent reruns
- [ ] T007 Benchmarks in `<package>_bench_test.go` capturing performance budgets and success metrics

## Phase 3.3: Core Implementation (ONLY after tests are failing)
- [ ] T008 [P] Implement installer workflow in `pkg/installer/<feature>.go` with structured logging hooks
- [ ] T009 [P] Extend kubeclient adapters in `internal/kubeclient/<feature>.go`
- [ ] T010 Wire CLI command in `cmd/chainctl/<command>/command.go` with UX-parity flags and output modes
- [ ] T011 Implement rollback/feature flag logic in `pkg/installer/<feature>_rollback.go`
- [ ] T012 Ensure telemetry emission in `pkg/telemetry/<feature>.go`

## Phase 3.4: Integration & Validation
- [ ] T013 Validate performance via `go test -bench` and capture baseline artifacts
- [ ] T014 Record `--dry-run` output and update operator docs/log samples
- [ ] T015 Verify memory and goroutine budgets using `go test -run Benchmark -bench . -benchtime=1x`
- [ ] T016 Update runbooks in `docs/runbooks/<feature>.md`

## Phase 3.5: Polish
- [ ] T017 [P] Update `docs/cli/commands.md` with syntax, examples, and json output contract
- [ ] T018 [P] Add changelog entry in `CHANGELOG.md`
- [ ] T019 Finalize configuration examples in `examples/<feature>/values.yaml`
- [ ] T020 Run `make lint test` (or equivalent) and attach artifacts to PR description

## Dependencies
- Tests (T004-T007) before implementation (T008-T012)
- T008 blocks T011 and telemetry integration
- Performance validation (T013-T015) requires implementation complete
- Documentation tasks (T014-T019) depend on finalized UX/metrics

## Parallel Example
```
# Launch independent test authoring together:
Task: "Unit tests in test/unit/<feature>_test.go"
Task: "Integration tests in test/integration/<feature>_test.go"
Task: "CLI e2e test in test/e2e/<command>_test.go"
Task: "Benchmarks in pkg/installer/<feature>_bench_test.go"
```

## Notes
- [P] tasks = different files/packages, no shared state
- Verify tests fail before implementing
- Attach lint/test/benchmark artifacts to PR
- Ensure every task references relevant constitutional principles (P1–P5)

## Task Generation Rules
*Applied during main() execution*

1. **From Contracts & CLI specs**:
   - Each contract file → e2e test + CLI task
   - Each CLI flag/command → UX validation task (dry-run, json output)
   
2. **From Data Model**:
   - Each entity → installer/package update + validation tests
   - Relationships → kubeclient or dependency wiring tasks
   
3. **From User Stories**:
   - Each operator scenario → e2e test + doc sample
   - Failure stories → rollback coverage tasks

4. **Ordering**:
   - Setup → Tests → Installer packages → CLI wiring → Validation → Docs
   - Dependencies block parallel execution

## Validation Checklist
*GATE: Checked by main() before returning*

- [ ] All principles P1–P5 have explicit coverage in tasks
- [ ] Tests precede implementation work
- [ ] Performance/benchmark tasks exist for installer changes
- [ ] CLI UX tasks cover dry-run/confirm/json flows
- [ ] Documentation/runbook updates are present
- [ ] No [P] task shares a file or package with another [P] task
