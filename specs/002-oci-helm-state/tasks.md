# Tasks: CLI Support for OCI Helm Charts and Persistent State

**Input**: Design documents from `/specs/002-oci-helm-state/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/, quickstart.md

## Phase 3.1: Setup
- [X] T001 Ensure `go.mod`/`go.sum` include required schema/testing libs (e.g., gojsonschema); run `go mod tidy` and document dependency diffs.
- [X] T002 Update lint/test automation (`Makefile`, CI scripts) so new `pkg/state`, `pkg/helm/resolver`, and integration/e2e suites are exercised by `make lint test`.
- [X] T003 Scaffold directories/files per plan (`pkg/state/`, `internal/state/`, `cmd/chainctl/app/install.go`, `test/integration/app/`, `test/e2e/`, `test/fixtures/state/`) with package docs and TODO guards.

## Phase 3.2: Tests First (TDD)
- [X] T004 [P] Author unit tests in `pkg/state/state_test.go` covering state file auto-creation, override handling (name vs path), atomic writes, and read-only error propagation.
- [X] T005 [P] Author unit tests in `pkg/helm/resolver_test.go` validating OCI URL parsing, bundle fallback, and mutual-exclusion errors.
- [X] T006 [P] Extend CLI unit tests in `cmd/chainctl/app/install_command_test.go` & `upgrade_command_test.go` to cover new flags (`--state-file`, `--state-file-name`), mutual exclusion, and state-file path messaging.
- [X] T007 [P] Create contract tests in `test/unit/app/cli_contract_test.go` ensuring text/JSON outputs match `contracts/app-upgrade-cli.md` (including state path field) and that invalid override errors align with contract wording.
- [X] T008 [P] Create schema validation tests in `test/unit/state/schema_contract_test.go` verifying generated JSON matches `contracts/state-schema.json` for install/update actions.
- [X] T009 [P] Add envtest integration test `test/integration/app/oci_install_test.go` covering OCI chart install flow and state write success.
- [X] T010 [P] Add envtest integration test `test/integration/app/oci_update_test.go` validating update flow, state overwrite, and timestamp/version changes.
- [X] T011 [P] Add envtest integration test `test/integration/app/bundle_install_test.go` covering local bundle path usage and state capture.
- [X] T012 [P] Add envtest integration test `test/integration/app/state_error_test.go` simulating read-only directory to confirm deployment success + explicit error.
- [X] T013 [P] Add e2e CLI test in `test/e2e/app_install_update_test.go` executing quickstart install/update/error scenarios via compiled binary, including a run with custom state overrides.
- [X] T014 [P] Add resolver benchmark `pkg/helm/resolver_bench_test.go` measuring OCI vs. bundle resolution to enforce P4 budgets.

## Phase 3.3: Core Implementation (after tests fail)
- [X] T015 Implement state persistence library in `pkg/state/state.go` (XDG path resolution hooks, override precedence, atomic JSON writes, permission guards).
- [X] T016 Implement state path helpers in `internal/state/path.go` with XDG fallback, override validation, and 0700 directory creation.
- [X] T017 Implement chart resolver in `pkg/helm/resolver.go` with OCI registry pulls (using Helm SDK) and bundle passthrough.
- [X] T018 Integrate resolver into `pkg/helm/client.go` and related types, ensuring telemetry attributes expose source type/digest.
- [X] T019 Create new `cmd/chainctl/app/install.go` command wiring shared execution pipeline (install/update) and exposing new flags (`--chart`, `--release-name`, `--app-version`, `--namespace`, `--state-file`, `--state-file-name`).
- [X] T020 Refactor `cmd/chainctl/app/upgrade.go` to reuse shared workflow, enforce mutually exclusive flags, and surface clarification-driven errors.
- [X] T021 Wire state persistence into CLI workflow (install/update) recording actions, handling auto-create, honoring overrides, and reporting state file path on success/failure.
- [X] T022 Extend telemetry/logging in `pkg/telemetry` and command layer to emit source/action metadata and structured error logs per P5.
- [X] T023 Update `internal/config`/validation to propagate namespace, release name, and version overrides into profiles safely.

## Phase 3.4: Integration & Validation
- [X] T024 Run `go test ./...` (unit + integration), envtest/kind suites, and ensure new tests pass; capture artifacts for PR.
- [X] T025 Execute e2e script (`make test-e2e` placeholder or dedicated script) against kind cluster documenting outputs per quickstart.
- [X] T026 Execute resolver benchmark `go test -bench Resolver -run Benchmark ./pkg/helm` and record results in `performance/` notes.
- [X] T027 Generate sample state fixtures in `test/fixtures/state/app_success.json` and `app_error.json` for documentation/tests.

## Phase 3.5: Polish
- [X] T028 Update operator docs in `docs/cli/commands.md` and `docs/runbooks/installer.md` with new flags, workflows, and troubleshooting guidance.
- [X] T029 Update Quickstart material in `docs/cli/commands.md` or create `docs/cli/app-install-quickstart.md` mirroring quickstart.md steps; link from README if needed.
- [X] T030 Add telemetry/observability notes to `docs/telemetry/README.md` (or relevant doc) reflecting new metrics/logs.
- [X] T031 Add changelog entry in `CHANGELOG.md` under Unreleased describing OCI chart + state persistence feature.
- [X] T032 Run `gofmt ./... && gofumpt ./... && golangci-lint run` followed by `git status` to verify cleanliness before PR.

## Dependencies
- T001 → (all tasks)
- T002 → T024, T032
- T003 → T004–T027 (scaffolding)
- Tests T004–T014 must complete (and fail) before implementation T015–T023
- T015 & T016 unblock T021; T017 & T018 unblock T019/T020/T022
- T019–T023 must pass before validation tasks T024–T027
- Documentation tasks T028–T031 depend on completed implementation & validation outputs
- T032 is final gate before PR

## Parallel Execution Example
```
# Author test suites in parallel once scaffolding is ready
Task: "T004 [P] Author unit tests in pkg/state/state_test.go"
Task: "T005 [P] Author unit tests in pkg/helm/resolver_test.go"
Task: "T007 [P] Create contract tests in test/unit/app/cli_contract_test.go"
Task: "T009 [P] Add envtest integration test test/integration/app/oci_install_test.go"
```

- Use `task run T004` / `task run T005` style invocations (or equivalent task runner) to coordinate parallel execution.
- Ensure environment variables (`KUBEBUILDER_ASSETS`, registry creds) are configured before running integration/e2e tasks.
