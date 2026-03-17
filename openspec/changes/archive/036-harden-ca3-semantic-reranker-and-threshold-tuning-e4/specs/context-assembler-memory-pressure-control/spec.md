## MODIFIED Requirements

### Requirement: CA3 semantic quality gate SHALL support optional embedding similarity component
CA3 semantic quality evaluation MUST support an optional cosine-based embedding similarity component in addition to existing rule-based scoring, and MUST preserve rule-only compatibility when embedding scorer is disabled.

When reranker is enabled, CA3 semantic quality evaluation MUST include a deterministic reranker stage for final gate decision while preserving compatibility of existing thresholds when reranker is disabled.

#### Scenario: Hybrid+reranker mode enabled
- **WHEN** CA3 semantic compaction runs with embedding scorer and reranker enabled
- **THEN** quality evaluation uses rule signal and cosine similarity signal, then applies reranker before final gate decision

#### Scenario: Default hybrid mode without reranker
- **WHEN** embedding scorer is enabled and reranker is disabled by config
- **THEN** CA3 uses base hybrid scoring path and existing threshold semantics

#### Scenario: Rule-only compatibility mode
- **WHEN** CA3 semantic compaction runs with embedding scorer disabled
- **THEN** quality evaluation behaves equivalently to existing rule-only scoring path

### Requirement: CA3 semantic compaction SHALL expose embedding fallback diagnostics
CA3 semantic compaction MUST emit explicit diagnostics for embedding scorer and reranker path selection, including fallback reasons when adapter execution is unavailable, reranker execution fails, or reranker is bypassed.

#### Scenario: Adapter unavailable fallback
- **WHEN** embedding scorer is enabled but configured adapter is unavailable
- **THEN** CA3 records fallback diagnostics and applies policy-driven fallback behavior

#### Scenario: Reranker timeout fallback
- **WHEN** reranker request times out under `best_effort`
- **THEN** CA3 records reranker timeout fallback reason and continues with pre-reranker quality path

#### Scenario: Multi-provider adapter and reranker selection
- **WHEN** runtime config selects OpenAI, Gemini, or Anthropic embedding provider with reranker enabled
- **THEN** CA3 executes selected provider path and preserves equivalent fallback semantics

## ADDED Requirements

### Requirement: CA3 semantic compaction SHALL keep deterministic fallback chain with reranker enabled
CA3 semantic compaction MUST preserve deterministic fallback chain under reranker-enabled quality path:
`hybrid+reranker` -> `hybrid only` -> `rule-only` according to policy and failure reason.

#### Scenario: Best-effort falls back one step
- **WHEN** reranker fails but embedding scorer succeeds and policy is `best_effort`
- **THEN** CA3 falls back to hybrid-only path and continues compaction

#### Scenario: Best-effort falls back to rule-only
- **WHEN** both embedding scorer and reranker paths fail under `best_effort`
- **THEN** CA3 falls back to rule-only path and continues compaction

#### Scenario: Fail-fast aborts without fallback
- **WHEN** reranker or embedding scorer path fails and policy is `fail_fast`
- **THEN** CA3 aborts assembly before model execution and does not enter fallback chain
