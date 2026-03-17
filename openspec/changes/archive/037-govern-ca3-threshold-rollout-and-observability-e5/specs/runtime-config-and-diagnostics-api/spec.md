## ADDED Requirements

### Requirement: Runtime config SHALL expose CA3 threshold governance rollout controls
Runtime config MUST expose CA3 threshold governance controls with deterministic precedence `env > file > default`, including governance mode (`enforce|dry_run`), profile version identifier, and provider:model-scoped rollout match settings.

#### Scenario: Startup with valid CA3 governance config
- **WHEN** runtime starts with valid CA3 governance mode and provider:model rollout settings
- **THEN** effective config includes resolved governance fields and CA3 can evaluate rollout matching deterministically

#### Scenario: Invalid CA3 governance mode value
- **WHEN** runtime loads CA3 governance config with unsupported mode value
- **THEN** startup or hot reload fails fast with a validation error

### Requirement: Runtime diagnostics SHALL expose additive CA3 threshold governance fields
Runtime diagnostics MUST expose additive CA3 threshold governance observability fields sufficient for rollout triage, including profile version, rollout-match hit, threshold-source, threshold-hit, and fallback reason, without changing existing field semantics.

#### Scenario: Governance-enabled CA3 enforcement run
- **WHEN** CA3 executes with governance mode `enforce` and rollout match hits selected provider:model
- **THEN** diagnostics include additive governance fields for profile version, rollout hit, and threshold evaluation outcome

#### Scenario: Governance fallback path in best-effort mode
- **WHEN** governance evaluation fails under `best_effort`
- **THEN** diagnostics include governance fallback reason while preserving existing reranker/compaction fields
