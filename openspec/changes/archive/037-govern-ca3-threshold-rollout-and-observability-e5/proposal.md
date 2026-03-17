## Why

E4 delivered CA3 reranker and offline threshold tuning, but threshold rollout still lacks explicit governance controls for staged enablement and safe rollback. We need a minimal E5 layer to make threshold changes operationally controllable in production without changing current failure-policy contracts.

## What Changes

- Add CA3 threshold governance config for versioned profiles, provider/model-scoped rollout controls, and deterministic fallback/rollback behavior.
- Add provider:model-only canary controls so rollout can be staged by selected provider/model pairs without introducing run/session-level segmentation.
- Add CA3 threshold evaluation modes:
  - `enforce`: threshold decisions affect gate outcomes.
  - `dry_run`: threshold decisions are evaluated for debugging but do not affect final gate outcomes.
- Extend CA3 diagnostics/event fields for rollout governance signals (profile version, rollout match/hit, threshold drift indicators, fallback reason), while keeping compatibility additive.
- Preserve existing policy semantics (`best_effort` fallback and `fail_fast` termination) and Run/Stream semantic equivalence.
- Add contract tests and benchmark regression gates for governance-enabled paths.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: add CA3 threshold governance/rollout config contracts and additive observability fields.
- `ca3-semantic-embedding-adapter`: extend CA3 reranker threshold decision flow with governance mode (`enforce|dry_run`) and provider:model rollout matching.
- `context-assembler-memory-pressure-control`: define deterministic fallback/rollback chain and Run/Stream equivalence for governance-enabled CA3 semantic compaction.

## Impact

- Affected code:
  - `runtime/config` (governance and rollout config schema + validation)
  - `context/assembler` (threshold governance mode, rollout matching, rollback/fallback decisions)
  - `core/runner`, `runtime/diagnostics`, `observability/event` (additive rollout observability fields)
  - `integration` and `context/assembler` tests (contract + benchmark)
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/context-assembler-phased-plan.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
  - `docs/mainline-contract-test-index.md`
- Compatibility:
  - additive config and diagnostics fields
  - no change to existing default behavior unless governance mode/rollout is enabled
