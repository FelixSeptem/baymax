## ADDED Requirements

### Requirement: CA3 semantic quality gate SHALL support optional embedding similarity component
CA3 semantic quality evaluation MUST support an optional cosine-based embedding similarity component in addition to existing rule-based scoring, and MUST preserve rule-only compatibility when embedding scorer is disabled.

#### Scenario: Hybrid mode enabled
- **WHEN** CA3 semantic compaction runs with embedding scorer enabled
- **THEN** quality evaluation uses both rule signal and cosine similarity signal to compute gate input

#### Scenario: Default hybrid weights and threshold strategy
- **WHEN** embedding scorer is enabled with default configuration
- **THEN** CA3 uses `rule_weight=0.7`, `embedding_weight=0.3`, and reuses existing quality threshold semantics

#### Scenario: Rule-only compatibility mode
- **WHEN** CA3 semantic compaction runs with embedding scorer disabled
- **THEN** quality evaluation behaves equivalently to existing rule-only scoring path

### Requirement: CA3 semantic compaction SHALL expose embedding fallback diagnostics
CA3 semantic compaction MUST emit explicit diagnostics for embedding scorer path selection and fallback reasons when adapter execution is unavailable or fails.

#### Scenario: Adapter unavailable fallback
- **WHEN** embedding scorer is enabled but configured adapter is unavailable
- **THEN** CA3 records fallback diagnostics and applies policy-driven fallback behavior

#### Scenario: Adapter timeout fallback
- **WHEN** embedding scorer request times out under `best_effort`
- **THEN** CA3 records timeout fallback reason and continues with rule-only quality scoring

#### Scenario: Multi-provider adapter selection
- **WHEN** runtime config selects OpenAI, Gemini, or Anthropic embedding provider
- **THEN** CA3 executes the selected provider adapter path and preserves equivalent fallback semantics
