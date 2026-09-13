## Why

Evaluation currently compares corpus and experiment outcomes, while compaction, handoff, snapshot restore, and recovery expose related state through separate reference-first contracts. There is no bounded comparator that can prove the source-owned continuity facts remain equivalent across those boundaries, so regressions can be reported only as opaque digest changes or discovered after recovery. This change adds an offline, deterministic continuity comparison contract and replay gate while keeping transcript and artifact bodies outside the evaluation surface.

## What Changes

- Add a versioned `eval_continuity_comparison.v1` contract for bounded baseline/candidate projections.
- Normalize and compare fixed continuity axes: identity, objective, task, attempt/lease, workspace binding, pending correlated request, checkpoint, and artifact references.
- Emit deterministic, axis-specific drift classifications for schema, identity, ownership, missing/duplicate references, privacy violations, Run/Stream parity, and recovery idempotency.
- Add adapters/tests proving continuity projections can be derived from existing handoff and unified snapshot/checkpoint contracts without copying source-owned bodies or changing ownership.
- Extend diagnostics replay with a continuity fixture format and fail-fast evaluator; retain existing fixture compatibility.
- Add contract/gate coverage and update the mainline contract index and development roadmap.
- Do not add runtime configuration, persistence services, provider/tool/Git calls, scheduler FSMs, or transcript/reasoning storage.

## Capabilities

### New Capabilities

- `evaluation-continuity-comparison`: Bounded, reference-only baseline/candidate continuity projections and deterministic drift comparison across compaction, handoff, snapshot, and recovery boundaries.

### Modified Capabilities

- `diagnostics-replay-tooling`: Replay tooling validates the new versioned continuity fixtures and reports canonical continuity drift classes while preserving historical fixtures.
- `evaluation-corpus-badcase-and-experiment-contract`: Evaluation comparison gains a continuity comparison surface that remains additive, deterministic, and reference-first.

## Impact

- Affected code: `runtime/evalcontract`, `context/handoff` projection adapters, `orchestration/snapshot` projection adapters, and `tool/diagnosticsreplay` fixture/replay code.
- Affected docs: `docs/development-roadmap.md` and `docs/mainline-contract-test-index.md`.
- No new external dependencies or hosted services; no change to provider, tool, workspace, or transcript ownership.

## Example Impact Assessment

无需示例变更（附理由）：本提案只增加离线 audit/contract/replay 能力，不改变 `examples/agent-modes` 的运行路径或用户可见行为。
