# ca3-semantic-embedding-adapter Specification

## Purpose
TBD - created by archiving change implement-ca3-semantic-embedding-adapter-e3. Update Purpose after archive.
## Requirements
### Requirement: CA3 embedding adapters SHALL provide multi-provider similarity scoring
CA3 semantic quality pipeline MUST support provider-backed embedding adapters for OpenAI, Gemini, and Anthropic that compute similarity signals for quality evaluation when embedding scorer is enabled.

#### Scenario: Embedding scorer enabled and selected adapter available
- **WHEN** runtime enables CA3 embedding scorer and selected adapter (OpenAI, Gemini, or Anthropic) is available
- **THEN** CA3 computes embedding similarity and includes it in quality evaluation

#### Scenario: Embedding scorer disabled
- **WHEN** runtime does not enable CA3 embedding scorer
- **THEN** CA3 quality evaluation runs without embedding similarity contribution

### Requirement: CA3 hybrid quality scoring SHALL be deterministic and weight-driven
CA3 MUST compute base hybrid quality score using deterministic weighted composition of rule score and cosine-based embedding similarity score, with validated bounded weights.

When reranker is enabled, CA3 MUST apply a deterministic reranker adjustment stage after base hybrid score calculation and before final gate decision.

#### Scenario: Valid hybrid and reranker config
- **WHEN** runtime provides valid rule/embedding weights and valid reranker config
- **THEN** CA3 computes base hybrid score and applies reranker stage deterministically before quality gate decision

#### Scenario: Invalid hybrid or reranker configuration
- **WHEN** runtime receives invalid weight values or invalid reranker config
- **THEN** runtime rejects configuration with fail-fast validation error before applying updates

### Requirement: CA3 embedding adapter failures SHALL preserve policy semantics
Embedding adapter and reranker-stage failures MUST preserve existing stage policy semantics for `best_effort` and `fail_fast`.

#### Scenario: Adapter or reranker failure under best-effort
- **WHEN** embedding adapter call or reranker stage fails and stage policy is `best_effort`
- **THEN** CA3 falls back to pre-reranker quality path and records fallback diagnostics

#### Scenario: Adapter or reranker failure under fail-fast
- **WHEN** embedding adapter call or reranker stage fails and stage policy is `fail_fast`
- **THEN** CA3 terminates assembly with normalized error before model execution

### Requirement: CA3 embedding configuration SHALL support independent credentials
CA3 embedding adapter execution MUST support provider-specific independent credentials and MUST allow fallback to shared model-step credentials when independent credentials are not configured.

#### Scenario: Independent credentials configured
- **WHEN** runtime provides provider-specific embedding credentials
- **THEN** embedding adapter uses independent credentials for embedding calls

#### Scenario: Independent credentials not configured
- **WHEN** runtime does not provide provider-specific embedding credentials
- **THEN** embedding adapter uses shared model-step credentials according to credential precedence rules

### Requirement: Run and Stream SHALL remain semantically equivalent with embedding scoring
For equivalent input and effective config, Run and Stream paths MUST produce semantically equivalent mode selection, reranker usage, fallback behavior, and quality gate outcomes.

#### Scenario: Equivalent Run and Stream with reranker enabled
- **WHEN** equivalent requests run in embedding+reranker mode without failures
- **THEN** Run and Stream produce semantically equivalent final quality gate pass/fail outcomes

#### Scenario: Equivalent Run and Stream fallback path
- **WHEN** equivalent requests encounter adapter/reranker failure under `best_effort`
- **THEN** both Run and Stream fall back to pre-reranker path with semantically equivalent diagnostics

### Requirement: CA3 reranker SHALL expose extensible provider-specific implementation interface
CA3 MUST define a stable internal extension interface for provider-specific reranker implementations.

The default runtime path MUST use built-in generic reranker behavior, and users MAY register provider-specific implementations through the extension interface without breaking core contract semantics.

#### Scenario: User registers provider-specific reranker implementation
- **WHEN** runtime loads a valid custom reranker implementation for a provider
- **THEN** CA3 executes the custom implementation while preserving failure-policy semantics

#### Scenario: Custom reranker implementation unavailable
- **WHEN** no custom implementation is registered
- **THEN** CA3 uses built-in reranker behavior with unchanged contract semantics

### Requirement: CA3 reranker provider coverage SHALL include OpenAI, Gemini, and Anthropic usable paths
E4 reranker flow MUST provide usable execution paths for OpenAI, Gemini, and Anthropic provider selections.

#### Scenario: Anthropic reranker selected
- **WHEN** runtime enables reranker with Anthropic provider/model config
- **THEN** CA3 executes usable Anthropic reranker path without relying on unsupported-only fallback behavior

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

