# Tasks: Logging Transparency for Operational Steps

**Input**: Design documents from `/specs/003-logging/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/logging-schema.json, quickstart.md

## Task List
- [x] T001 Audit logging feature dependencies (UUID generation, JSON schema validation) and run `go mod tidy` to ensure go.mod/go.sum are current (repository root)
- [x] T002 Scaffold new logging helpers: create `internal/cli/logging/` package, `pkg/telemetry/logger.go`, and `cmd/chainctl/cluster/logging.go` with minimal placeholders (no logic yet)
- [x] T003 [P] Create redaction unit tests in `internal/cli/logging/sanitizer_test.go` covering secret masking, allowlist behavior, and token edge cases (P1, P2, P5)
- [x] T004 [P] Add structured logger unit tests in `pkg/telemetry/logger_test.go` validating required fields, severity transitions, init failure handling, and metadata serialization (P1, P2, P5)
- [x] T005 [P] Author bootstrap runner unit tests in `pkg/bootstrap/runner_test.go` ensuring command capture, stderr buffering, and error propagation (P1, P2, P5)
- [x] T006 [P] Add JSON contract test in `test/unit/logging_schema_contract_test.go` to validate emitted sample entries against `contracts/logging-schema.json` (P2, P5)
- [x] T007 Draft integration test `test/integration/cluster_logging_success_test.go` verifying dry-run install logs start/end workflow entries and sanitized Helm command (P2, P3, P5)
- [x] T008 Draft integration test `test/integration/cluster_logging_failure_test.go` ensuring failing Helm command logs sanitized stderr excerpts and severity=error (P2, P5)
- [x] T009 Draft integration test `test/integration/cluster_bootstrap_logging_test.go` to confirm bootstrap failures emit structured command logs with correlation IDs (P2, P5)
- [x] T010 Draft integration test `test/integration/cluster_logging_disabled_test.go` that simulates logger initialization failure/disabled mode and asserts the CLI aborts before mutating cluster state with actionable error messaging (P2, P3, P5)
- [x] T011 Add benchmark suite in `pkg/telemetry/logger_bench_test.go` measuring per-entry overhead under 5% target (P2, P4)
- [x] T012 Implement sanitization helpers in `internal/cli/logging/sanitizer.go` with configurable patterns and test-driven behavior (P1, P5)
- [x] T013 Implement structured logger in `pkg/telemetry/logger.go` producing JSON lines with required fields, correlation propagation, and explicit error when initialization fails (P1, P5)
- [x] T014 Extend `pkg/telemetry/emitter.go` to share workflow IDs, invoke logger hooks, and surface initialization failures to callers so workflows abort cleanly (P1, P5)
- [x] T015 Implement command runner wrapper in `pkg/bootstrap/runner.go` to capture sanitized command strings, exit codes, and stderr excerpts (P1, P5)
- [x] T016 Integrate runner wrapper in `pkg/bootstrap/bootstrap.go` so bootstrap flow emits structured command logs, propagates workflow context, and respects fail-fast behavior when logging is unavailable (P3, P5)
- [x] T017 Introduce Helm executor logging adapter in `pkg/helm/executor.go` and update `pkg/helm/client.go` to emit sanitized Helm commands with fail-fast handling when logger cannot start (P1, P5)
- [x] T018 Add workflow logging helper in `cmd/chainctl/cluster/logging.go` to wrap high-level steps (bootstrap, helm) with start/end entries and propagate initialization failures (P3, P5)
- [x] T019 Wire install command (`cmd/chainctl/cluster/install.go`) to emit workflow start/end logs, pass correlation IDs, abort when logging setup fails, and log external command outcomes (P3, P5)
- [x] T020 Wire upgrade command (`cmd/chainctl/cluster/upgrade.go`) with the same logging helpers, fail-fast semantics, and failure capture (P3, P5)
- [x] T021 Update `cmd/chainctl/app/action.go` to log Helm resolve/install invocations, redact inputs per sanitizer rules, and honour logging initialization errors (P3, P5)
- [x] T022 Ensure integration tests in `cmd/chainctl/cluster` account for new logging side effects (including disabled mode) and update fixtures as needed (P2, P3)
- [x] T023 Run `gofmt`, `gofumpt`, `golangci-lint`, and full `go test ./...` to confirm code quality and failing tests now pass (P1, P2)
- [x] T024 Validate logger benchmark results meet <5% overhead; document findings in `pkg/telemetry/logger_bench_test.go` comments or artifacts (P4)
- [x] T025 Update operator documentation: refresh `specs/003-logging/quickstart.md`, add log field reference to `docs/runbooks/logging.md`, and capture sanitized stderr guidance plus fail-fast messaging (P3, P5)
- [x] T026 Publish sample JSON log bundle under `docs/examples/logging/` and reference ingestion steps for ELK integration (P3, P5)
- [x] T027 Prepare release notes: update `CHANGELOG.md` and `docs/governance/decisions.md` with logging feature summary and constitution reference (P3, P5)

## Dependencies
- T001 before T002 (module hygiene precedes scaffolding)
- T002 before tests touching new packages (T003–T010)
- Tests/benchmarks (T003–T011) must exist before implementation tasks (T012–T022)
- T012 feeds T013–T017; sanitizer logic required before wiring logs
- T018 depends on T013–T017 (logging infrastructure ready)
- T019 & T020 depend on T018 and prior package integrations
- T021 depends on T017 & T019
- T022 depends on T019–T021
- T023 depends on completion of implementation tasks (T012–T022)
- T024 depends on T013 & T014 (logger/emitter behavior finalized)
- T025–T027 follow successful validation (T023, T024)

## Parallel Execution Examples
```
# After scaffolding (T002) is complete, author tests in parallel:
/task P T003
/task P T004
/task P T005
/task P T006
/task P T010

# Once logging infrastructure (T012–T017) is merged, wire CLI flows concurrently:
/task P T019
/task P T020
/task P T021
```

## Validation Checklist
- Unit, integration, and benchmark tests authored before implementation (P2, P4)
- Logging sanitization prevents sensitive data exposure and fail-fast behavior is covered by tests (P5)
- Workflow start/end logs preserve operator UX expectations including `--dry-run` support (P3)
- Performance overhead measured and within target (P4)
- Documentation and governance updates completed (P3, P5)
- All tasks respect Go craftsmanship standards (P1)
