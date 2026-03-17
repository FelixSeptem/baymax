## Why

E3 delivered CA3 semantic embedding adapters and hybrid quality scoring, but score behavior still depends on static defaults and lacks a dedicated reranker/tuning workflow for stable operations. We should add a narrow E4 hardening layer so teams can tune quality thresholds with reproducible evidence while preserving existing fail-fast and best-effort semantics.

## What Changes

- Add CA3 semantic reranker stage on top of existing rule+embedding quality scoring, controlled by runtime config and default-off rollout.
- Add threshold tuning toolkit for offline evaluation and recommendation output (input corpus -> score report -> suggested threshold range).
- Add provider/model-scoped quality diagnostics and trend fields to support tuning and incident triage.
- Keep deterministic fallback semantics unchanged:
  - `best_effort`: reranker/scoring failure falls back to pre-reranker quality path.
  - `fail_fast`: reranker/scoring failure terminates assembly.
- Add contract tests and benchmark scenarios for reranker enabled/disabled behavior and Run/Stream semantic equivalence.

## Capabilities

### New Capabilities
- `ca3-semantic-threshold-tuning-toolkit`: Offline toolkit for evaluating CA3 semantic quality score distributions and recommending threshold ranges.

### Modified Capabilities
- `ca3-semantic-embedding-adapter`: Extend from adapter-backed hybrid scoring to reranker-enhanced quality decision path and provider/model observability.
- `context-assembler-memory-pressure-control`: Extend CA3 semantic compaction quality gate with reranker-aware fallback semantics and deterministic mode selection.
- `runtime-config-and-diagnostics-api`: Extend config and diagnostics contracts for reranker controls, tuning outputs, and provider/model-scoped quality signals.

## Impact

- Affected code:
  - `context/assembler` (reranker integration, fallback branch, quality pipeline composition)
  - `runtime/config` (reranker/tuning config fields + fail-fast validation)
  - `runtime/diagnostics`, `observability/event`, `core/runner` (new quality observability fields)
  - `integration` (benchmark and contract tests)
  - `scripts/` or `cmd/` utilities (offline threshold tuning toolkit)
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/context-assembler-phased-plan.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
  - `docs/mainline-contract-test-index.md`
- Compatibility:
  - Default behavior remains current E3 path when reranker is not enabled.
  - New config/diagnostics/tool outputs are additive and backward compatible.
