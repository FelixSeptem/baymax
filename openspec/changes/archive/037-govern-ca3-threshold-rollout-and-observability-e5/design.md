## Context

E4 introduced CA3 reranker and offline threshold tuning, but production rollout still depends on manual all-or-nothing config flips. Teams need a controlled path to stage threshold enforcement and roll back safely by provider/model while preserving existing CA3 contracts (`best_effort` and `fail_fast`) and Run/Stream semantic equivalence.

Current state:
- Threshold profiles exist and are mandatory when reranker is enabled.
- No explicit governance mode for evaluating threshold decisions without affecting final gating.
- No provider:model-scoped rollout control to stage threshold enforcement.
- Existing diagnostics are strong for reranker execution, but threshold-governance signals are not structured as a rollout layer.

Constraints:
- Keep behavior additive and backward-compatible.
- Do not introduce run/session-level rollout segmentation in E5.
- `dry_run` is debugging-only and not part of external diagnostics API contract.
- Include both contract tests and benchmark regression gates.

## Goals / Non-Goals

**Goals:**
- Add CA3 threshold governance controls that support deterministic provider:model rollout and rollback.
- Add governance mode `enforce|dry_run` so operators can validate threshold decisions before enforcing.
- Keep failure-policy semantics unchanged under governance-enabled paths.
- Add additive observability fields for threshold governance outcomes.
- Add contract tests and benchmark checks for governance-enabled and disabled paths.

**Non-Goals:**
- No online auto-learning or adaptive threshold training.
- No default threshold template distribution in this milestone.
- No run/session/user-level canary routing.
- No approval workflow fields (e.g., operator/reason) in config contracts.

## Decisions

### Decision 1: Provider:model is the only rollout granularity in E5
- Choice: rollout matching keys are limited to `provider:model`.
- Rationale: aligns with existing threshold profile indexing and keeps rollout deterministic and auditable.
- Alternative considered: add session/run/user segmentation.
- Rejected because: increases routing complexity and creates non-deterministic troubleshooting paths.

### Decision 2: Introduce explicit governance mode (`enforce|dry_run`)
- Choice:
  - `enforce`: threshold decisions affect final quality gate.
  - `dry_run`: threshold decisions are evaluated and recorded for debugging but do not alter final gate outcome.
- Rationale: enables low-risk rollout validation before production enforcement.
- Alternative considered: single enforce-only mode with manual shadow tooling.
- Rejected because: shadow tooling fragments runtime truth and complicates parity checks.

### Decision 3: Keep rollback deterministic and local to threshold governance layer
- Choice: if governance config/profile lookup/rollout matching fails:
  - `best_effort` falls back to pre-governance threshold path and records fallback signal.
  - `fail_fast` terminates assembly before model execution.
- Rationale: preserves existing policy contract and limits behavioral surprises.
- Alternative considered: always fallback regardless of policy.
- Rejected because: violates strict-mode guarantees.

### Decision 4: Make observability additive, but keep dry-run output internal-debug oriented
- Choice: add governance fields for profile version, rollout hit, threshold-source, threshold-hit, drift/fallback reason; do not require dry-run signal in external diagnostics API contract.
- Rationale: respects current diagnostics compatibility and user-requested boundary.
- Alternative considered: expose all dry-run internals in external diagnostics API.
- Rejected because: expands external contract scope without clear operator need.

### Decision 5: Dual quality gate for E5 completion
- Choice: E5 done criteria require both contract-test coverage and benchmark regression checks.
- Rationale: governance changes affect both semantics and latency; both must be guarded.
- Alternative considered: tests only.
- Rejected because: misses rollout-path latency regressions.

## Risks / Trade-offs

- [Risk] Config surface area increases and can be misconfigured.
  - Mitigation: fail-fast validation with precise error messages and deterministic precedence.
- [Risk] `dry_run` can be misunderstood as enforcement-ready.
  - Mitigation: explicit mode field and docs with non-enforcement semantics.
- [Risk] Provider:model rollout may not cover all real-world segmentation needs.
  - Mitigation: keep extension points open for future finer-grained rollout change.
- [Risk] Additional observability fields can create ingestion drift.
  - Mitigation: additive-only schema evolution and contract tests for backward compatibility.

## Migration Plan

1. Extend runtime config schema with threshold governance controls and provider:model rollout fields.
2. Add validation and deterministic resolution order (`env > file > default`).
3. Integrate governance mode and rollout matching into CA3 reranker threshold decision flow.
4. Add additive diagnostics/event/store mappings for governance signals.
5. Add contract tests for enforce/dry_run behavior and Run/Stream equivalence.
6. Add benchmark cases for governance enabled vs disabled latency comparison.
7. Update docs and acceptance/test index.

## Open Questions

- None for E5 scope.
