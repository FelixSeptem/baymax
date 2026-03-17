## MODIFIED Requirements

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

### Requirement: Run and Stream SHALL remain semantically equivalent with embedding scoring
For equivalent input and effective config, Run and Stream paths MUST produce semantically equivalent mode selection, reranker usage, fallback behavior, and quality gate outcomes.

#### Scenario: Equivalent Run and Stream with reranker enabled
- **WHEN** equivalent requests run in embedding+reranker mode without failures
- **THEN** Run and Stream produce semantically equivalent final quality gate pass/fail outcomes

#### Scenario: Equivalent Run and Stream fallback path
- **WHEN** equivalent requests encounter adapter/reranker failure under `best_effort`
- **THEN** both Run and Stream fall back to pre-reranker path with semantically equivalent diagnostics

## ADDED Requirements

### Requirement: CA3 reranker SHALL support provider/model-scoped threshold profiles
CA3 reranker execution MUST require explicit provider+model threshold profile for the selected provider/model when reranker is enabled.

Missing provider+model threshold profile MUST be treated as configuration error and MUST fail fast before runtime activation.

#### Scenario: Provider+model profile exists
- **WHEN** reranker is enabled and provider+model threshold profile is configured
- **THEN** CA3 applies the provider+model threshold profile for final gate decision

#### Scenario: Provider+model profile missing
- **WHEN** reranker is enabled and provider+model profile is unavailable
- **THEN** runtime rejects config with fail-fast validation error

### Requirement: CA3 reranker diagnostics SHALL expose provider/model quality signals
CA3 MUST emit additive diagnostics for reranker path selection and quality decision context, including provider/model identity, threshold source, and reranker fallback reason when applicable.

#### Scenario: Reranker success diagnostics
- **WHEN** reranker executes successfully
- **THEN** diagnostics include provider/model identity and threshold source fields

#### Scenario: Reranker fallback diagnostics
- **WHEN** reranker falls back or is bypassed under policy handling
- **THEN** diagnostics include explicit fallback reason and effective quality path marker

## ADDED Requirements

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
