<!--
Sync Impact Report
Version change: N/A → 1.0.0
Modified principles: none
Added sections: Core Principles, Operational Guardrails, Delivery Workflow, Governance
Removed sections: none
Templates requiring updates:
  ✅ .specify/templates/plan-template.md
  ✅ .specify/templates/tasks-template.md
Follow-up TODOs: none
-->
# chainctl Constitution

## Core Principles

### P1. Go Craftsmanship as Default
- All Go code MUST be gofmt and gofumpt formatted, pass `go vet`, and clear the shared golangci-lint profile before review.
- Package boundaries MUST reflect user-facing commands or internal services; shared utilities belong under `pkg/internal` with documented contracts.
- Every exported function MUST declare clear error semantics using wrapped `errors` to preserve context.
- Refactors that lower maintainability index or raise cyclomatic complexity beyond 15 MUST include a remediation plan.
Rationale: Maintainable, reviewable Go code keeps the CLI reliable during rapid installer iterations.

### P2. Test Rigor for Cluster Confidence
- Unit, integration, and end-to-end tests MUST be defined for each feature before implementation; unit tests fail first, then integration, then e2e.
- Integration tests MUST exercise Kubernetes interactions with envtest or kind; they MAY mock cloud providers but MUST cover kubectl-equivalent flows.
- End-to-end smoke tests MUST validate that installations succeed, rerun idempotently, and rollback safely on failure.
- No merge is allowed if test coverage for touched packages drops or any new logic lacks deterministic tests.
Rationale: High-fidelity tests are the only safe way to ship cluster-changing installers.

### P3. Consistent Operator UX
- The CLI MUST expose nouns then verbs (`chainctl cluster install`), consistent flag names, and long-form options mirroring kube tooling.
- Every destructive command MUST support `--dry-run`, `--confirm`, and namespace scoping before applying changes.
- Output MUST default to human-readable progress with `--output json` parity; errors report actionable remediation steps.
- Interactive prompts MUST provide defaults, timeout-safe behavior, and a `--non-interactive` switch for automation.
Rationale: Consistency reduces operator error across clusters and scripts.

### P4. Performance Budgets for Install Paths
- Installer flows MUST complete within 10 minutes on a 3-node dev cluster and surface progress within 5 seconds of action start.
- API calls to Kubernetes MUST be batched or cached to avoid more than 50 sequential calls per phase; exponential backoff governs retries.
- Memory usage of the CLI process MUST stay below 512 MB, and concurrent goroutines MUST be bounded and instrumented.
- Performance regressions MUST ship with benchmarks (`go test -bench`) proving adherence to the agreed budgets.
Rationale: Predictable performance makes large-cluster rollouts feasible and scriptable.

### P5. Operational Safety & Observability
- Every command touching cluster state MUST emit structured logs (JSON) at info/error and support `--log-level` overrides.
- Telemetry hooks MUST capture duration, success/failure, and cluster identifiers without leaking secrets.
- Rollback paths and feature flags MUST exist for all new installation behaviors before exposure in stable channels.
- Operational runbooks MUST be updated alongside code when UX or performance behaviors change.
Rationale: Safe observability allows operators to diagnose and recover from failed installs quickly.

## Operational Guardrails
- Supported Go toolchain: 1.22+, managed via `asdf` or `goenv`; CI MUST pin exact versions.
- Kubernetes compatibility target: upstream 1.28–1.30; features relying on alpha APIs MUST be gated behind feature flags.
- Third-party binaries (helm, kubectl) MUST be version-locked and verified via checksums before use.
- Secrets and kubeconfigs MUST never be written to disk outside designated cache directories with 0700 permissions.

## Delivery Workflow
- All work begins with a constitution check recorded in `plan.md`, mapping features to impacted principles and mitigations.
- Pull requests MUST include: green unit/integration/e2e suites, lint report, `--dry-run` UX recordings/log samples, and benchmark deltas.
- Release candidates MUST pass a full install/uninstall matrix across at least two Kubernetes distributions (kind + managed offering).
- Incidents or regressions MUST trigger a postmortem that reviews principle adherence and updates this constitution if gaps appear.

## Governance
- Amendments require consensus from the maintainers group and a recorded decision in `docs/governance/decisions.md` with impact analysis.
- Semantic versioning applies to this constitution; MAJOR for principle removals or incompatible rewrites, MINOR for new principles/sections, PATCH for clarifications.
- The maintainers chair schedules quarterly compliance reviews; findings feed backlog items within one sprint.
- Emergency deviations MUST be documented in the affected PR and resolved before the next release branch is cut.

**Version**: 1.0.0 | **Ratified**: 2025-10-05 | **Last Amended**: 2025-10-05
