## ADDED Requirements

### Requirement: CA3 embedding adapters SHALL provide multi-provider similarity scoring
CA3 semantic quality pipeline MUST support provider-backed embedding adapters for OpenAI, Gemini, and Anthropic that compute similarity signals for quality evaluation when embedding scorer is enabled.

#### Scenario: Embedding scorer enabled and selected adapter available
- **WHEN** runtime enables CA3 embedding scorer and selected adapter (OpenAI, Gemini, or Anthropic) is available
- **THEN** CA3 computes embedding similarity and includes it in quality evaluation

#### Scenario: Embedding scorer disabled
- **WHEN** runtime does not enable CA3 embedding scorer
- **THEN** CA3 quality evaluation runs without embedding similarity contribution

### Requirement: CA3 hybrid quality scoring SHALL be deterministic and weight-driven
CA3 MUST compute hybrid quality score using deterministic weighted composition of rule score and cosine-based embedding similarity score, with validated bounded weights.

#### Scenario: Valid hybrid weights configured
- **WHEN** runtime provides valid rule and embedding weights and cosine metric setting
- **THEN** CA3 computes hybrid score using configured weights and applies existing quality gate policy

#### Scenario: Invalid weight configuration
- **WHEN** runtime receives invalid hybrid score weights
- **THEN** runtime rejects configuration with fail-fast validation error before applying updates

### Requirement: CA3 embedding adapter failures SHALL preserve policy semantics
Embedding adapter failures MUST preserve existing stage policy semantics for `best_effort` and `fail_fast`.

#### Scenario: Adapter failure under best-effort
- **WHEN** embedding adapter call fails and stage policy is `best_effort`
- **THEN** CA3 falls back to rule-only scoring and records fallback diagnostics

#### Scenario: Adapter failure under fail-fast
- **WHEN** embedding adapter call fails and stage policy is `fail_fast`
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
For equivalent input and effective config, Run and Stream paths MUST produce semantically equivalent embedding scoring mode selection, fallback behavior, and quality gate outcomes.

#### Scenario: Equivalent Run and Stream success path
- **WHEN** equivalent requests run in embedding-enabled mode without adapter failures
- **THEN** Run and Stream produce semantically equivalent quality gate pass/fail outcomes

#### Scenario: Equivalent Run and Stream fallback path
- **WHEN** equivalent requests encounter embedding adapter failure under `best_effort`
- **THEN** both Run and Stream fall back to rule-only scoring with equivalent diagnostics semantics
