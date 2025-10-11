# Tasks: Declarative YAML Configuration for CLI Flags

**Input**: Design documents from `/specs/004-declarative-config-file/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/, quickstart.md

## Phase 3.1: Setup
- [x] T001 Update module deps for YAML tooling: add/verify `gopkg.in/yaml.v3` in `go.mod`; run `go mod tidy`
- [x] T002 Ensure lint/format pipelines cover new packages: update `Makefile` or CI to include `pkg/config` and `internal/config`
- [x] T003 Scaffold configuration packages and files: create `pkg/config` and `internal/config` directories with placeholder doc.go files

## Phase 3.2: Tests First (TDD)
- [x] T004 Write unit tests for YAML discovery precedence in `internal/config/locator_test.go`
- [x] T005 Write unit tests for YAML parsing and secret rejection in `internal/config/loader_test.go`
- [x] T006 [P] Author resolver unit tests for merge/override logic in `pkg/config/resolver_test.go`
- [x] T007 Create configuration profile struct tests in `pkg/config/profile_test.go`
- [x] T008 [P] Add CLI integration test covering `--config` precedence, reusable profiles, multi-command YAML files, and summary output (table + `--output json`) in `test/integration/declarative_config_test.go`
- [x] T009 [P] Add CLI e2e dry-run scenario using YAML defaults in `test/e2e/declarative_config_workflow_test.go`
- [x] T010 Define benchmark skeleton measuring loader performance in `pkg/config/resolver_bench_test.go`

## Phase 3.3: Core Implementation
- [x] T011 Implement configuration discovery order in `internal/config/locator.go`
- [x] T012 Implement YAML parsing, validation (schema, unknown keys), and secret guard in `internal/config/loader.go`
- [x] T013 Build exported configuration profile types in `pkg/config/profile.go`
- [x] T014 Implement precedence resolver merging defaults, command sections, and runtime overrides in `pkg/config/resolver.go`
- [x] T015 Wire `--config` flag and resolved flag application into `cmd/chainctl/app/config_flag.go` (new file) and update command entrypoints (`cmd/chainctl/app/install.go`, `cmd/chainctl/app/upgrade.go`, `cmd/chainctl/cluster/install/command.go`)
- [x] T016 Emit structured logs/telemetry for configuration discovery and resolution in `pkg/telemetry` or relevant command logging hooks
- [x] T017 Update validation helpers to surface actionable errors for unknown flags/commands in `internal/validation/schema.go`

## Phase 3.4: Integration & Validation
- [x] T018 Run `go test ./...` ensuring new unit/integration/e2e tests fail before implementation, then pass after implementation
- [x] T019 Execute resolver benchmark and record results to confirm <200ms load target
- [x] T020 Capture CLI summary output screenshots or logs demonstrating YAML summary and override messaging; store under `docs/examples/config/`
- [x] T021 Add or update operator runbook section in `docs/runbooks/declarative-config.md`

## Phase 3.5: Polish
- [x] T022 [P] Update CLI documentation (`docs/cli/commands.md`, `README.md`) with `--config` usage and discovery order
- [x] T023 [P] Provide sample YAML configuration under `examples/config/chainctl.yaml`
- [x] T024 [P] Add changelog entry in `CHANGELOG.md` describing declarative config support
- [x] T025 Final verification: run `gofmt`, `gofumpt`, `golangci-lint`, and full `go test ./...`; attach artifacts to PR

## Parallel Execution Guidance
- Initial test authoring can run concurrently: T006, T008, T009 target different packages.
- After core implementation, documentation tasks T022–T024 may proceed in parallel once CLI behavior is frozen.

## Dependencies
- Setup (T001–T003) precedes all other work.
- Tests (T004–T010) must exist and fail before implementing T011–T017.
- Implementation T011–T017 must complete before validation tasks T018–T021.
- Documentation polish T022–T024 depends on final CLI UX (post T015/T020).
- Final verification T025 runs last.
