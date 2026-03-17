## ADDED Requirements

### Requirement: CA3 reranker threshold governance SHALL support enforce and dry-run modes
CA3 semantic quality pipeline MUST support threshold governance mode selection where `enforce` applies threshold decisions to final gate outcomes and `dry_run` evaluates threshold decisions without changing final gate outcomes.

#### Scenario: Enforce mode applies governance threshold decision
- **WHEN** CA3 runs with governance mode `enforce` and rollout matches selected provider:model
- **THEN** threshold governance decision is applied to final quality gate outcome

#### Scenario: Dry-run mode does not alter final gate outcome
- **WHEN** CA3 runs with governance mode `dry_run` and rollout matches selected provider:model
- **THEN** threshold governance decision is evaluated but final quality gate outcome follows pre-governance path

### Requirement: CA3 threshold rollout matching SHALL be deterministic by provider:model
CA3 reranker threshold governance MUST resolve rollout applicability deterministically using provider:model key matching for the selected reranker provider and model.

#### Scenario: Provider:model rollout match hit
- **WHEN** selected reranker provider:model exists in rollout match set
- **THEN** CA3 applies configured governance mode for threshold evaluation

#### Scenario: Provider:model rollout match miss
- **WHEN** selected reranker provider:model does not exist in rollout match set
- **THEN** CA3 bypasses governance enforcement and preserves baseline threshold behavior
