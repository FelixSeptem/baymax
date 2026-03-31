## ADDED Requirements

### Requirement: Readiness preflight SHALL evaluate observability export and diagnostics bundle dependencies
Runtime readiness preflight MUST evaluate observability export sink and diagnostics bundle storage dependencies when corresponding features are enabled.

Canonical finding codes for this milestone MUST include:
- `observability.export.profile_invalid`
- `observability.export.sink_unavailable`
- `observability.export.auth_invalid`
- `diagnostics.bundle.output_unavailable`
- `diagnostics.bundle.policy_invalid`

Findings MUST use canonical structured schema and remain machine-assertable.

#### Scenario: Preflight detects unavailable export sink
- **WHEN** export profile is enabled and sink probe fails
- **THEN** readiness preflight returns canonical `observability.export.sink_unavailable` finding

#### Scenario: Preflight detects unwritable bundle output path
- **WHEN** bundle generation is enabled and configured output path is unavailable
- **THEN** readiness preflight returns canonical `diagnostics.bundle.output_unavailable` finding

### Requirement: Observability readiness findings SHALL preserve strict non-strict mapping semantics
Readiness classification for observability and bundle findings MUST follow existing strict/non-strict policy semantics:
- `strict=false` may classify recoverable observability findings as `degraded`,
- `strict=true` MUST escalate equivalent blocking-class findings to `blocked`.

#### Scenario: Non-strict policy with export sink unavailable
- **WHEN** preflight receives `observability.export.sink_unavailable` and `runtime.readiness.strict=false`
- **THEN** readiness status is `degraded` with canonical finding preserved

#### Scenario: Strict policy with same export finding
- **WHEN** equivalent preflight input is evaluated with `runtime.readiness.strict=true`
- **THEN** readiness status escalates to `blocked` with deterministic canonical finding
