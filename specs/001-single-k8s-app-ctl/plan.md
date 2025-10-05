# Implementation Plan: Single-k8s-app-ctl CLI

**Branch**: `001-single-k8s-app-ctl` | **Date**: 2025-10-05 | **Spec**: `specs/001-single-k8s-app-ctl/spec.md`
**Input**: Feature specification from `/specs/001-single-k8s-app-ctl/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code, or `AGENTS.md` for all other agents).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Deliver a Go 1.22 single binary `chainctl` that can bootstrap or reuse k3s clusters on Ubuntu/RHEL, install and upgrade the Helm-based micro-services stack, orchestrate k3s upgrades via system-upgrade-controller, operate with removable-media air-gapped bundles, and manage node joins through scoped tokens while enforcing encrypted configuration inputs and observability budgets.

## Technical Context
**Language/Version**: Go 1.22 (gofmt + gofumpt enforced)  
**Primary Dependencies**: Cobra CLI, Helm SDK, k3s install scripts, system-upgrade-controller CRDs, OpenTelemetry SDK  
**Storage**: local-path StorageClass (k3s default), filesystem bundle mount under `/opt/chainctl/bundles`  
**Testing**: `go test`, `ginkgo`, controller-runtime envtest, kind-based e2e smoke suites  
**Target Platform**: Linux (Ubuntu 22.04+, RHEL 9+) hosts executing the binary  
**Project Type**: single CLI project with supporting packages  
**Performance Goals**: Install/upgrade completes <10 min on 3-node dev cluster; CLI progress emitted within 5s of action start  
**Constraints**: Offline-capable via mounted tarball, memory footprint <512 MB, idempotent reruns, encrypted values required  
**Scale/Scope**: Single-node to 10-node clusters initially; join flow must handle parallel token issuance and readiness verification

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **P1 · Go Craftsmanship**: Plan adds lint targets (`gofmt`, `gofumpt`, `go vet`, `golangci-lint`) to CI, segments code into `cmd/chainctl`, `pkg/bootstrap`, `pkg/helm`, `pkg/upgrade`, `pkg/secrets`, and `internal/validation` with clear contracts and error wrapping.
- **P2 · Test Rigor**: Define unit tests per package (decrypt, token lifecycle, manifest parsing), envtest suites for k3s API interactions, kind-based e2e for `cluster install`, `app upgrade`, `cluster upgrade`, and `node join` covering dry-run, idempotency, rollback, and passphrase failure paths; coverage gates maintain or increase existing percentage.
- **P3 · Operator UX**: CLI commands follow noun-verb nomenclature, provide `--dry-run`, `--confirm` for mutations, namespace scoping, consistent flag names, JSON/text parity, and ensure `--non-interactive` for automation (passphrase via flag/env) while prompts include defaults/timeouts.
- **P4 · Performance Budgets**: Benchmarks for bundle extraction, Helm apply, and upgrade orchestration added under `pkg/.../_bench_test.go`; instrumentation collects phase durations and enforces concurrency limits/watches to stay under 512 MB and 10-minute install budget.
- **P5 · Operational Safety**: Structured JSON logs per phase, telemetry envelopes (OpenTelemetry) for durations/outcomes, feature flags for risky flows (e.g., k3s surge levels), automatic rollback definitions, and runbook updates in `docs/runbooks/installer.md` triggered alongside feature delivery.

## Project Structure

### Documentation (this feature)
```
specs/001-single-k8s-app-ctl/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── chainctl-cluster-install.md
│   ├── chainctl-app-upgrade.md
│   ├── chainctl-cluster-upgrade.md
│   ├── chainctl-node-join.md
│   ├── chainctl-node-token-create.md
│   └── chainctl-encrypt-values.md
└── tasks.md  # generated during /tasks
```

### Source Code (repository root)
```
cmd/chainctl/
├── main.go
├── root.go
├── cluster/
│   ├── install.go
│   └── upgrade.go
├── app/
│   └── upgrade.go
├── node/
│   ├── join.go
│   └── token.go
└── secrets/
    └── encrypt.go

pkg/
├── bootstrap/        # k3s installation orchestration
├── helm/             # Helm client wrappers & diff
├── upgrade/          # system-upgrade-controller plans and watchers
├── bundle/           # tarball + manifest handling
├── secrets/          # encryption/decryption helpers
├── tokens/           # join token lifecycle management
└── telemetry/        # logging + metrics emitters

internal/
├── validation/       # host & cluster preflight checks
├── config/           # profile loading, defaults, feature flags
└── kubeclient/       # typed clients for k3s & controller-runtime

test/
├── unit/             # package-focused tests
├── integration/      # envtest + kind integration suites
└── e2e/              # CLI-level smoke & regression tests
```

**Structure Decision**: Maintain single Go module with CLI entry under `cmd/chainctl` and supporting domain packages under `pkg/…` to preserve clean interfaces and testability; `internal/` captures helper code not exported. Dedicated `test/` tree houses BDD/e2e suites to avoid polluting production module.

## Phase 0: Outline & Research
1. Resolved runtime uncertainties documented in `research.md`, covering k3s bootstrap approach, bundle mounting, encrypted values handling, token lifecycle, system-upgrade-controller usage, and telemetry strategy.
2. Established manifest format for removable-media tarball (checksum-verified) and encryption command to satisfy offline and security constraints.
3. Confirmed OpenTelemetry metrics + Go benchmarks satisfy performance reporting requirements without external services.

**Output**: `/Users/dobrovolsky/sources/golang/chainctl/specs/001-single-k8s-app-ctl/research.md`

## Phase 1: Design & Contracts
1. Data shapes for installation profiles, bundle manifests, tokens, upgrade plans, and telemetry envelopes defined in `data-model.md` to anchor Go struct design and validation rules.
2. CLI command contracts authored for install, app upgrade, cluster upgrade, node join, token create, and encrypt-values flows, capturing flags, outputs, exit codes, and observability expectations.
3. Quickstart walkthrough documents bootstrap, upgrade, node join, and cluster upgrade scenarios for both online and air-gapped environments.
4. `.specify/scripts/bash/update-agent-context.sh codex` executed to refresh shared agent guidance with new technologies (Go 1.22, Helm SDK, system-upgrade-controller, OpenTelemetry, envtest, kind).

**Outputs**:
- `/Users/dobrovolsky/sources/golang/chainctl/specs/001-single-k8s-app-ctl/data-model.md`
- `/Users/dobrovolsky/sources/golang/chainctl/specs/001-single-k8s-app-ctl/contracts/`
- `/Users/dobrovolsky/sources/golang/chainctl/specs/001-single-k8s-app-ctl/quickstart.md`
- Agent context updated via script

## Phase 2: Task Planning Approach
- `/tasks` will derive tasks aligning with constitutional principles: setup (module scaffolding, lint config), tests-first (unit/envtest/kind, benchmarks, dry-run artifacts), core implementation (bootstrap, bundle, helm, upgrade, tokens, secrets), integration (telemetry, performance validation), and polish (docs/runbooks, JSON parity checks).
- Tasks will reference concrete files in `cmd/chainctl/...`, `pkg/...`, and `test/...`, ensuring [P] markers only where file boundaries allow parallelism.
- Constitution gates embedded: every task maps to P1–P5 compliance (lint/test/perf/logging/runbooks) with explicit dry-run outputs and benchmark artifacts before completion.

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | n/a | n/a |

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

