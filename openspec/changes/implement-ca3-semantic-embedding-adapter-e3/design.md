## Context

CA3 semantic compaction currently has a rule-based quality gate plus template controls and an embedding SPI placeholder, but no provider adapter is bound in runtime. As a result, quality decisions rely on rule heuristics only and may be less robust for semantically similar but lexically different inputs.

This change targets the next incremental milestone: wire provider-backed embedding adapters (OpenAI, Gemini, Anthropic) into CA3 quality scoring while preserving existing v1 operational safety semantics (`best_effort` fallback and `fail_fast` termination), Run/Stream equivalence, and additive diagnostics.

## Goals / Non-Goals

**Goals:**
- Deliver production-usable embedding adapter paths for OpenAI, Gemini, and Anthropic in CA3 semantic quality scoring.
- Introduce hybrid quality scoring (rule score + embedding similarity component) with deterministic weighting and threshold checks.
- Preserve existing policy behavior:
  - `best_effort`: adapter failures fall back to rule-only score path.
  - `fail_fast`: adapter failures terminate assembly.
- Expose runtime config and diagnostics fields needed for operations, tuning, and incident triage.
- Keep default behavior backward compatible (embedding scorer disabled by default).
- Use cosine similarity as the E3 baseline metric.
- Support independent embedding credentials in runtime config (separate from model-step credentials).
- Set default hybrid weights to `rule=0.7` and `embedding=0.3`, and reuse the existing quality threshold.

**Non-Goals:**
- Not introducing vector database lifecycle management or retrieval index orchestration.
- Not changing CA2 retriever protocol or agentic routing behavior.
- Not replacing existing rule-score logic; embedding is additive and optional.
- Not introducing non-cosine similarity metrics in E3.

## Decisions

### Decision 1: Ship three provider adapters in E3 with explicit runtime selection
- Choice: implement OpenAI, Gemini, and Anthropic embedding adapters in E3, selected by runtime config.
- Rationale: this aligns with project multi-provider baseline and avoids a second churn cycle for adapter expansion.
- Alternative considered: single-provider first.
- Rejected because: would immediately require follow-up proposal for parity and delay cross-provider validation.

### Decision 2: Hybrid score formula with bounded, validated weights
- Choice: compute `quality_score = rule_weight * rule_score + embedding_weight * similarity_score`, with weight validation and deterministic fallback behavior.
- Rationale: preserves existing rule-score interpretability and adds semantic signal incrementally.
- Alternative considered: fully replace rule-score with embedding similarity.
- Rejected because: lowers explainability and increases sensitivity to provider behavior.

### Decision 3: Cosine-only similarity metric for E3
- Choice: E3 only supports cosine similarity for embedding contribution.
- Rationale: keeps metric behavior predictable across providers and simplifies cross-provider contract tests.
- Alternative considered: supporting multiple metrics immediately.
- Rejected because: adds tuning and compatibility complexity without immediate validation value.
### Decision 4: Adapter failures follow existing stage policy semantics
- Choice:
  - `best_effort`: if embedding call fails/timeouts/unavailable, continue with rule-only scoring and record fallback reason.
  - `fail_fast`: return error before model step proceeds.
- Rationale: keeps operator expectations aligned with existing CA3 failure-policy contract.
- Alternative considered: always fallback regardless of policy.
- Rejected because: violates strict-failure semantics already established in runtime.

### Decision 5: Embedding credentials can be configured independently from model-step path
- Choice: runtime config supports dedicated embedding credentials and endpoints per provider, with fallback to shared credentials when not explicitly configured.
- Rationale: allows safer key scoping and provider-specific operations in production.
- Alternative considered: strictly reusing model-step credential chain only.
- Rejected because: reduces operational flexibility and blocks least-privilege setups.

### Decision 6: Diagnostics expand as additive fields only
- Choice: add embedding-related quality diagnostics without changing existing field meaning.
- Rationale: protects compatibility for existing diagnostics consumers.
- Alternative considered: replacing existing quality fields with a new schema.
- Rejected because: unnecessary migration cost and risk.

### Decision 7: Default weights and threshold strategy are fixed for first rollout
- Choice: default `rule_weight=0.7`, `embedding_weight=0.3`, and hybrid scoring reuses existing quality threshold in E3.
- Rationale: conservative rollout bias that preserves existing gate sensitivity while adding semantic signal.
- Alternative considered: separate hybrid threshold from day one.
- Rejected because: introduces extra tuning dimension before baseline data is collected.

## Risks / Trade-offs

- [Risk] Provider embedding latency increases CA3 pressure-handling overhead
  - Mitigation: configurable timeout, fallback path, and benchmark gate with relative regression threshold.
- [Risk] Cross-provider embedding behavior drift may cause score distribution variance
  - Mitigation: provider-specific contract fixtures and baseline snapshots with shared cosine metric.
- [Risk] Weight or threshold misconfiguration can degrade quality decisions
  - Mitigation: fail-fast config validation and conservative defaults with scorer disabled by default.
- [Risk] Embedding dimension/model mismatch causes runtime errors
  - Mitigation: startup/hot-reload validation and explicit diagnostics reason codes.
- [Risk] Run/Stream behavioral drift under adapter failures
  - Mitigation: contract tests for equivalent inputs across both paths.

## Migration Plan

1. Add runtime config schema for embedding scorer (enabled flag, provider/model selector, optional independent credentials, timeout, weights).
2. Implement OpenAI, Gemini, and Anthropic embedding adapters and internal scorer wiring.
3. Integrate hybrid quality evaluator into CA3 semantic path with policy-aware fallback handling.
4. Extend diagnostics/event mapping with embedding contribution and fallback metadata.
5. Add/extend contract tests for pass/fail/fallback cases on Run and Stream.
6. Add benchmark baselines for semantic path with embedding enabled/disabled and document tuning guidance.

## Open Questions

- Default model identifiers for OpenAI, Gemini, and Anthropic embedding paths need final naming confirmation in runtime config examples.
- Credential fallback precedence (independent embedding credential vs shared model credential) needs explicit docs wording to avoid operator confusion.
