## Why

Checkpoint and protocol layers already carry workspace provenance and integrity references, while scheduler task/attempt records carry lease and retry identity without a durable workspace binding. In parallel, mailbox completion and source-owned runtime-input safe points exist as separate contracts, but their ownership and replay behavior at the completion-to-next-decision seam is not yet proven. This is now actionable because the previous provider/host conformance work is archived and the roadmap names this seam as the next P1 risk.

## What Changes

- Establish an additive, nullable, defaultable contract for associating a scheduler task/attempt and lease generation with bounded workspace provenance references.
- Define deterministic validation for workspace association, integrity drift, stale attempt/workspace commits, retry and lease rollover, and snapshot/recovery reconciliation.
- Define the ownership boundary for routing a durable mailbox completion into an existing source-owned runtime-input safe point, without introducing a second coordination or task state machine.
- Add bounded positive, negative, boundary, replay, and Run/Stream parity fixtures for missing, dirty, conflict, drift, duplicate, late, disconnected, and recovered completion cases.
- Add diagnostics/replay classifications and shell/PowerShell gate coverage through existing `RuntimeRecorder` and replay owners.
- Preserve current scheduler, mailbox, checkpoint, snapshot, terminal outcome, and runtime-input source ownership; introduce no workspace store or hosted service.

## Capabilities

### New Capabilities

- `durable-task-attempt-workspace-binding`: Defines observable task/attempt/lease binding to reference-only workspace provenance, rollover rules, integrity validation, and recovery reconciliation.
- `completion-safe-point-ownership`: Defines how mailbox-backed completion results correlate to an existing runtime-input safe point and next model decision with idempotent, late, disconnect, recovery, and Run/Stream semantics.

### Modified Capabilities

No existing capability requirement is replaced in this audit-first phase. The new capabilities constrain additive integration points owned by the existing scheduler, snapshot/recovery, mailbox, runtime-input, terminal-outcome, and diagnostics-replay contracts.

## Impact

- Affected owners: `orchestration/scheduler`, `orchestration/mailbox`, `orchestration/composer`, `orchestration/snapshot`, `core/runner`, `core/types`, `observability/event`, and `tool/diagnosticsreplay`.
- Expected first implementation slice: audit fixtures, replay classifications, contract/gate tests, and documentation; runtime fields or API changes are permitted only if a fixture demonstrates a real drift.
- Non-goals: Git/worktree lifecycle management, runtime Git/shell execution, hosted workspace/artifact persistence, automatic merge/push, a second task/session/coordination FSM, or copying external daemon/thread/worktree-index patterns.
- Rollback: remove the new fixture/replay/gate namespace and any additive optional projections; existing scheduler/mailbox/snapshot/runtime-input behavior remains usable without migration.

## Example Impact Assessment

无需示例变更（附理由）：首阶段只验证 scheduler、recovery、mailbox 和 runtime-input 接缝，不修改 `examples/agent-modes`；若后续需要示例变化，必须先完成 `MATRIX.md` 与对应 README 文档基线。
