# Phase 0 Research: OCI Helm & State Persistence

## Decision: Use Helm SDK OCI client for remote charts
- **Rationale**: Helm v3 SDK already bundled in repo supports `registry` login/pull flows and honors cache/config directories, minimizing new dependencies while aligning with existing Helm interactions in `pkg/helm`.
- **Alternatives Considered**:
  - Shelling out to `helm` binary: rejected to avoid process management and inconsistent error handling.
  - Implementing custom OCI downloader: rejected due to security and maintenance overhead.

## Decision: Store CLI execution state in XDG config directory
- **Rationale**: Using `~/.config/chainctl/state/app.json` (falling back to `$HOME/.chainctl/state/app.json`) meets P5 by keeping operator state private with `0700` permissions and avoids polluting repo directories.
- **Alternatives Considered**:
  - Persisting alongside bundle assets: rejected because bundles may be read-only or shared across users.
  - Environment-dependent temp files: rejected due to volatility and lack of audit trail.

## Decision: Enforce mutual exclusivity between OCI references and local bundles at flag parsing
- **Rationale**: Early validation in CLI command improves UX, aligns with clarification, and simplifies downstream logic by guaranteeing exactly one chart source.
- **Alternatives Considered**:
  - Deferring validation to installer layer: rejected because it couples error handling with deployment side effects.
  - Implicit precedence (OCI over local): rejected per clarification to avoid hidden behavior.

## Decision: Auto-create missing state file before performing Helm action
- **Rationale**: Guarantees state availability for downstream flows, satisfies FR-009, and keeps user workflow frictionless while still surfacing write errors post-action.
- **Alternatives Considered**:
  - Requiring a separate `state init` command: rejected as redundant and error-prone.
  - Lazily writing only after successful action: rejected because concurrent reads expect structured file.

## Decision: Surface state-write failures without rollbacks
- **Rationale**: Clarification dictates preserving deployment while surfacing actionable error; plan includes guidance messaging and follow-up command exit codes.
- **Alternatives Considered**:
  - Rolling back Helm release: rejected due to risk of partial failures and divergence from requested behavior.
  - Ignoring failures silently: rejected as it breaks audit requirements.

## Decision: Support operator overrides for state file name/path
- **Rationale**: Provides flexibility for compliance workflows and aligns with clarification that operators may need alternative storage locations while keeping defaults secure.
- **Alternatives Considered**:
  - Hard-coding single filename/location: rejected to avoid collisions when multiple apps run on same host.
  - Allowing only directory override: rejected because some environments require explicit filename control.
