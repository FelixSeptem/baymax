## ADDED Requirements

### Requirement: Readiness preflight SHALL evaluate sandbox adapter profile-pack availability and compatibility
Runtime readiness preflight MUST evaluate sandbox adapter profile-pack dependencies when sandbox mode is enabled.

Canonical finding codes for this milestone MUST include:
- `sandbox.adapter.profile_missing`
- `sandbox.adapter.backend_not_supported`
- `sandbox.adapter.host_mismatch`
- `sandbox.adapter.session_mode_unsupported`

#### Scenario: Required sandbox adapter profile is missing
- **WHEN** effective sandbox configuration references non-existent adapter profile
- **THEN** readiness preflight returns canonical `sandbox.adapter.profile_missing` finding

#### Scenario: Host backend support is unavailable
- **WHEN** referenced adapter profile backend is unsupported on current host/runtime
- **THEN** readiness preflight returns canonical `sandbox.adapter.backend_not_supported` finding

### Requirement: Sandbox adapter profile findings SHALL preserve strict/non-strict mapping semantics
Readiness preflight MUST map sandbox adapter profile findings using existing strict/non-strict rules:
- non-strict mode MAY classify recoverable adapter-profile issues as `degraded`,
- strict mode MUST escalate equivalent blocking-class findings to `blocked`.

#### Scenario: Non-strict mode with non-required adapter profile issue
- **WHEN** preflight evaluates recoverable sandbox adapter profile issue under non-strict policy
- **THEN** readiness status is `degraded` with canonical sandbox adapter finding

#### Scenario: Strict mode with equivalent adapter profile issue
- **WHEN** same issue is evaluated under strict policy
- **THEN** readiness status escalates to `blocked`
