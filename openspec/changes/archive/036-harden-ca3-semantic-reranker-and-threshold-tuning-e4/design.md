## Context

E3 delivered CA3 embedding adapters, hybrid rule+cosine scoring, and additive diagnostics. The current gap is operational hardening: teams still rely on static threshold defaults and do not have a repeatable offline tuning workflow to calibrate score cut lines by provider/model.

E4 focuses on a narrow hardening layer:
- add an optional reranker stage after hybrid scoring,
- add an offline threshold tuning toolkit with reproducible outputs,
- extend diagnostics with provider/model-scoped quality trend fields.

This change MUST preserve existing policy contracts:
- `best_effort` continues with deterministic fallback when reranker path fails,
- `fail_fast` terminates assembly on reranker/scoring failure,
- Run and Stream semantics remain equivalent for equal input and config.

## Goals / Non-Goals

**Goals:**
- Add optional CA3 reranker decision stage that can refine gate decisions without changing default behavior when disabled.
- Provide an offline threshold tuning toolkit that produces evidence artifacts and threshold recommendations.
- Extend runtime config and diagnostics contracts for reranker controls and provider/model quality observability.
- Keep E3 compatibility and preserve existing failure-policy semantics.

**Non-Goals:**
- No automatic online threshold self-learning in E4.
- No new vector index lifecycle or retrieval orchestrator changes.
- No change to CA2 retriever routing logic.
- No mandatory enablement in production defaults.

## Decisions

### Decision 1: Keep reranker as an optional stage behind explicit config
- Choice: reranker is disabled by default and activated via CA3 config.
- Rationale: minimize behavior drift and allow staged rollout by environment.
- Alternative considered: enable reranker by default in non-prod.
- Rejected because: creates hidden expectation gaps between environments and increases triage complexity.

### Decision 2: Enforce provider/model threshold profiles when reranker is enabled
- Choice: provider/model threshold profiles are mandatory when reranker is enabled; missing profile is a fail-fast config error.
- Rationale: score distributions differ by embedding model and provider implementation.
- Alternative considered: optional provider/model profile with global fallback.
- Rejected because: optional fallback weakens governance and causes unpredictable cross-provider quality drift.

### Decision 3: Ship offline tuning toolkit in minimal markdown output mode
- Choice: toolkit reads labeled/fixture corpus and emits operator-facing markdown report (`md`) as required output; runtime config is not auto-mutated.
- Rationale: keeps operations reviewable and audit-friendly.
- Alternative considered: direct hot-reload patch from toolkit output.
- Rejected because: weak change control and higher operational risk.

### Decision 4: Preserve fail-fast and best-effort semantics exactly
- Choice:
  - `best_effort`: reranker failure falls back to pre-reranker path and records fallback diagnostics.
  - `fail_fast`: reranker/scoring failure aborts assembly before model execution.
- Rationale: existing operational contract is already depended on by callers.
- Alternative considered: always fallback.
- Rejected because: violates strict mode expectations.

### Decision 5: Diagnostics remain additive and backward compatible
- Choice: add reranker/tuning fields without changing existing field meaning.
- Rationale: avoid downstream parser breaks and migration churn.
- Alternative considered: replacing existing quality diagnostics schema.
- Rejected because: unnecessary migration cost for E4 scope.

### Decision 6: Provide extensible provider-specific reranker interface
- Choice: E4 defines a stable reranker extension interface so users can implement provider-specific reranker logic when needed.
- Rationale: keeps core path generic while allowing advanced provider-specific optimization without forcing it into core runtime.
- Alternative considered: hard-code provider-specific reranker logic in core.
- Rejected because: increases maintenance burden and slows future provider onboarding.

### Decision 7: Anthropic reranker path MUST be usable in E4
- Choice: E4 must include a usable Anthropic reranker path rather than diagnostics-only unsupported behavior.
- Rationale: maintain provider parity with explicit delivery requirement.
- Alternative considered: keep Anthropic as fallback-only in E4.
- Rejected because: does not satisfy operational parity goals.

### Decision 8: Use corpus-size baseline as recommendation guidance, not hard gate
- Choice: toolkit publishes corpus-size guidance in report and confidence notes, but does not enforce fixed sample thresholds as fail/accept hard gate.
- Rationale: keep rollout flexible across teams with different data maturity while preserving transparency.
- Alternative considered: strict acceptance gate with fixed sample thresholds.
- Rejected because: blocks adoption in smaller domains and increases operational friction.

## Risks / Trade-offs

- [Risk] Reranker latency may increase CA3 processing time.
  - Mitigation: bounded timeout, default-off rollout, and benchmark regression gates.
- [Risk] Poor corpus quality can produce misleading threshold recommendations.
  - Mitigation: enforce minimum corpus quality checks and publish confidence notes in toolkit report.
- [Risk] Mandatory provider/model thresholds increase onboarding complexity.
  - Mitigation: ship documented starter profile templates per provider/model and strict validation messages.
- [Risk] Run/Stream behavior drift under fallback paths.
  - Mitigation: contract tests for equivalent reranker success/fallback semantics.

## Migration Plan

1. Add runtime config schema for reranker controls and mandatory provider/model threshold profiles with startup/hot-reload validation.
2. Integrate reranker stage into CA3 quality pipeline with strict policy-aware fallback handling.
3. Implement offline threshold tuning toolkit command and report formats.
4. Extend diagnostics/event mapping with provider/model-scoped reranker and threshold-hit fields.
5. Add contract tests and benchmark baselines for reranker enabled/disabled and fallback behavior.
6. Update docs and roadmap with E4 scope, rollout, and tuning workflow.

## Open Questions

- None for current E4 scope.
