## ADDED Requirements

### Requirement: Replay Tooling SHALL Support A69 Context Compression Fixture Contract
Diagnostics replay tooling MUST support versioned A69 fixture contract for context compression production hardening (recommended fixture namespace: `context_compression_production.v1`).

A69 fixture coverage MUST include at minimum:
- semantic compaction quality outcome and fallback chain,
- rule-based tool-result eligibility behavior,
- lifecycle tier transition and swap-back ranking behavior,
- cold-store retention/quota/cleanup/compact actions,
- crash/restart/replay recovery idempotency behavior.

#### Scenario: Replay validates canonical A69 fixture
- **WHEN** replay tooling processes valid A69 fixture input and normalized output matches canonical expectation
- **THEN** replay validation succeeds deterministically

#### Scenario: Replay receives malformed A69 fixture schema
- **WHEN** replay tooling receives malformed or unsupported A69 fixture payload
- **THEN** replay validation fails fast with deterministic schema validation reason

### Requirement: Replay Drift Classification SHALL Include A69 Governance Drift Taxonomy
Replay tooling MUST classify A69 semantic drift using canonical classes:
- `context_compaction_quality_drift`
- `context_rule_eligibility_drift`
- `context_tier_transition_drift`
- `context_swapback_ranking_drift`
- `context_cold_store_governance_drift`
- `context_recovery_idempotency_drift`

#### Scenario: Replay detects swap-back ranking drift
- **WHEN** replay output swap-back ranking differs from canonical relevance/recency expectation
- **THEN** replay validation fails with deterministic `context_swapback_ranking_drift` classification

#### Scenario: Replay detects recovery idempotency drift
- **WHEN** replay output shows duplicated restore side effects under equivalent recovery fixture input
- **THEN** replay validation fails with deterministic `context_recovery_idempotency_drift` classification

### Requirement: A69 Fixture Support SHALL Preserve Mixed-Fixture Backward Compatibility
Adding A69 fixture support MUST NOT break historical fixture parsing or validation.

#### Scenario: Mixed fixture suite includes historical fixtures and A69 fixtures
- **WHEN** replay gate executes archived fixtures together with A69 fixtures
- **THEN** all fixture generations are parsed and validated deterministically without parser regression
