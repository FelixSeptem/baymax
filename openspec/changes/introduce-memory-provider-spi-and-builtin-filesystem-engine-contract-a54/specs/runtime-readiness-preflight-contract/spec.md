## ADDED Requirements

### Requirement: Readiness preflight SHALL evaluate memory backend profile and contract compatibility
Runtime readiness preflight MUST evaluate memory backend dependencies when memory capability is enabled.

Canonical finding codes for this milestone MUST include:
- `memory.mode_invalid`
- `memory.profile_missing`
- `memory.provider_not_supported`
- `memory.spi_unavailable`
- `memory.filesystem_path_invalid`
- `memory.contract_version_mismatch`

#### Scenario: External memory profile is missing
- **WHEN** effective memory config uses `external_spi` and referenced profile cannot be resolved
- **THEN** readiness returns canonical `memory.profile_missing` finding

#### Scenario: Builtin filesystem path is invalid
- **WHEN** effective memory config uses `builtin_filesystem` and configured root path is not usable
- **THEN** readiness returns canonical `memory.filesystem_path_invalid` finding

### Requirement: Memory readiness findings SHALL preserve strict and non-strict mapping semantics
Readiness preflight MUST apply existing strict/non-strict mapping to memory findings:
- non-strict mode MAY classify recoverable memory findings as `degraded`,
- strict mode MUST escalate equivalent blocking-class findings to `blocked`.

#### Scenario: Non-strict mode with recoverable SPI unavailable finding
- **WHEN** preflight detects recoverable external SPI unavailability under non-strict policy
- **THEN** readiness status is `degraded` with canonical memory finding

#### Scenario: Strict mode with equivalent SPI unavailable finding
- **WHEN** the same finding is evaluated under strict policy
- **THEN** readiness status escalates to `blocked`

### Requirement: Readiness preflight SHALL include memory fallback safety findings
Readiness preflight MUST validate whether configured memory fallback policy is executable for the active mode and environment.

Canonical safety findings MUST include:
- `memory.fallback_policy_conflict`
- `memory.fallback_target_unavailable`

#### Scenario: Fallback policy conflicts with active mode
- **WHEN** config sets fallback target inconsistent with active memory mode constraints
- **THEN** readiness returns canonical `memory.fallback_policy_conflict` finding

#### Scenario: Degrade-to-builtin policy but builtin target unavailable
- **WHEN** fallback policy is `degrade_to_builtin` but builtin backend fails readiness checks
- **THEN** readiness returns canonical `memory.fallback_target_unavailable` finding with deterministic status classification
