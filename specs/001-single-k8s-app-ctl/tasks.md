# Tasks: Single-k8s-app-ctl CLI

**Input**: Design documents from `/specs/001-single-k8s-app-ctl/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/, quickstart.md

## Execution Flow (main)
```
1. Load plan.md from feature directory (done)
2. Extract tech stack, structure, constitutional obligations
3. Derive tasks honoring P1–P5 gates with TDD-first ordering
4. Map each contract, entity, and scenario to concrete tasks
5. Confirm dependency-safe ordering and parallel opportunities
```

## Phase 3.1: Setup
- [x] T001 Establish Go workspace structure: create `cmd/chainctl`, `pkg/{bootstrap,helm,upgrade,bundle,secrets,tokens,telemetry}`, `internal/{validation,config,kubeclient}`, and `test/{unit,integration,e2e}` with placeholder README.md files.
- [x] T002 Configure tooling: update `go.mod`, add `tools.go` if needed, wire `Makefile` targets for `gofmt`, `gofumpt`, `golangci-lint`, `go test`, `go test -bench`, and add CI jobs.
- [x] T003 [P] Vendor third-party assets: pin Helm SDK, controller-runtime/envtest, OpenTelemetry, Cobra; update `go.sum` via `go mod tidy`.
- [x] T004 Document dev environment expectations in `docs/development/environment.md` (Go 1.22, asdf/goenv, bundle mount paths).

## Phase 3.2: Tests First (TDD)
- [x] T005 [P] Write unit tests for bundle handling in `test/unit/bundle_tarball_test.go` covering checksum failures and manifest parsing.
- [x] T006 [P] Write unit tests for encrypted values in `test/unit/secrets_decrypt_test.go` covering passphrase retry logic and memory wiping.
- [x] T007 [P] Write unit tests for join token lifecycle in `test/unit/tokens_lifecycle_test.go` validating scope, TTL, and revocation.
- [x] T008 [P] Author envtest integration for preflight validation in `test/integration/preflight_validation_test.go` ensuring host/cluster checks gate execution.
- [x] T009 [P] Author envtest integration for system-upgrade-controller plan creation in `test/integration/upgrade_plan_test.go` verifying CRDs and surge/drain settings.
- [x] T010 [P] Add benchmarks in `pkg/bootstrap/bootstrap_bench_test.go` and `pkg/helm/helm_apply_bench_test.go` to measure phase durations.
- [x] T011 [P] Create e2e CLI test `test/e2e/cluster_install_test.go` (kind) matching `chainctl cluster install` contract including air-gapped tarball and dry-run.
- [x] T012 [P] Create e2e CLI test `test/e2e/app_upgrade_test.go` validating Helm diff, rollback handling, and JSON output.
- [x] T013 [P] Create e2e CLI test `test/e2e/cluster_upgrade_test.go` covering system-upgrade-controller orchestration and timeout handling.
- [x] T014 [P] Create e2e CLI test `test/e2e/node_join_test.go` covering pre-shared token join success/failure paths.
- [x] T015 [P] Create CLI unit smoke tests for `chainctl encrypt-values` in `test/unit/secrets_encrypt_command_test.go`.

## Phase 3.3: Core Implementation
- [x] T016 Implement bundle management package `pkg/bundle` for tarball discovery, checksum validation, manifest parsing, and cached extraction; expose interfaces for reuse.
- [x] T017 Implement encrypted values handling in `pkg/secrets` (AES-256-GCM encrypt/decrypt, passphrase prompts, memory wiping) per FR-009.
- [x] T018 Implement installation profile loader + validation in `internal/config/profile_loader.go`, enforcing bootstrap/reuse rules and defaults.
- [x] T019 Implement k3s bootstrap orchestrator in `pkg/bootstrap/bootstrap.go` invoking install script with local-path SC and readiness waits.
- [x] T020 Implement Helm orchestration layer in `pkg/helm/client.go` covering diff, apply, rollback, and OpenTelemetry spans.
- [x] T021 Implement system-upgrade-controller integration in `pkg/upgrade/controller.go` managing CRDs, plan submissions, health gates, surge/drain config.
- [x] T022 Implement token lifecycle management in `pkg/tokens/store.go` (generate, hash, store, invalidate) and integrate with cluster secrets.
- [x] T023 Implement telemetry envelope emission in `pkg/telemetry/emitter.go` producing structured logs/metrics per phase.
- [x] T024 Wire CLI command `cmd/chainctl/cluster/install.go` coordinating preflight, bundle, bootstrap, Helm install, telemetry, and JSON/text output.
- [x] T025 Wire CLI command `cmd/chainctl/app/upgrade.go` utilizing Helm layer, rollback handling, and JSON output parity.
- [x] T026 Wire CLI command `cmd/chainctl/cluster/upgrade.go` orchestrating controller deployment, upgrade progress, and failure handling.
- [x] T027 Wire CLI command `cmd/chainctl/node/token.go` creating scoped tokens with TTL validation.
- [x] T028 Wire CLI command `cmd/chainctl/node/join.go` validating tokens, generating join instructions, monitoring node readiness.
- [x] T029 Wire CLI command `cmd/chainctl/secrets/encrypt.go` for encrypting values files, shared with unit tests.

## Phase 3.4: Integration & Validation
- [x] T030 Integrate host and cluster preflight checks in `internal/validation/preflight.go`, surface actionable remediation, and ensure dry-run mode stops before mutations.
- [x] T031 Implement OpenTelemetry wiring in `cmd/chainctl/main.go` (flags/env), exporting metrics/logs respecting privacy (hashed cluster ID).
- [x] T032 Capture dry-run artifacts: ensure `--dry-run` outputs stored under `artifacts/dry-run/` during CI and attach to PR template.
- [x] T033 Validate performance budgets by running benchmarks and recording baseline JSON in `artifacts/performance/install_baseline.json` and `artifacts/performance/upgrade_baseline.json`.
- [x] T034 Validate memory/goroutine ceilings using `GODEBUG=madvdontneed=1` runs plus `pprof` snapshots; document results in `docs/performance/budgets.md`.
- [x] T035 Implement rollback/runbook updates: refresh `docs/runbooks/installer.md` and add failure triage steps for bundle, passphrase, upgrade failures.
- [x] T036 Ensure observability configuration documented and sample logs added to `docs/telemetry/chainctl_samples.json`.

## Phase 3.5: Polish
- [x] T037 [P] Update CLI reference documentation `docs/cli/commands.md` with new commands, flags, and examples (text + JSON).
- [x] T038 [P] Update `CHANGELOG.md` with feature summary and testing artifacts links.
- [x] T039 [P] Refresh `quickstart.md` if command behavior changes, ensuring dry-run instructions match implementation.
- [x] T040 [P] Final lint/test gate: run `make lint`, `go test ./...`, envtest suites, kind e2e, and benchmarks; capture results in PR description.
- [x] T041 [P] Produce compliance checklist entry verifying P1–P5 adherence and attach to PR template (`docs/compliance/p1-p5-checklist.md`).

## Dependencies
- Phase 3.1 setup tasks precede all others.
- Tests (T005–T015) must be implemented and run before corresponding implementation tasks (T016–T029).
- Bundle/secrets/tokens packages (T016–T023) unblock CLI wiring tasks (T024–T029).
- Integration validation tasks (T030–T036) depend on core CLI functionality.
- Polish tasks (T037–T041) run last after successful validation.

## Parallel Execution Example
```
# After setup, run these in parallel (different files):
Task: "T005 Write unit tests for bundle handling in test/unit/bundle_tarball_test.go"
Task: "T006 Write unit tests for encrypted values in test/unit/secrets_decrypt_test.go"
Task: "T007 Write unit tests for join token lifecycle in test/unit/tokens_lifecycle_test.go"
Task: "T010 Add benchmarks in pkg/bootstrap/bootstrap_bench_test.go and pkg/helm/helm_apply_bench_test.go"
```

## Notes
- Mark tasks complete only with failing-first tests converted to passing via implementation (TDD enforced).
- All tasks must capture artifacts (dry-run outputs, benchmarks, logs) in repo or CI attachments.
- Ensure secrets never written to disk; use in-memory handling per P5.
- Reference plan.md and data-model.md for struct fields and validation rules while implementing.
