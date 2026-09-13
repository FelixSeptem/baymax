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

## Current Ownership Evidence Matrix (Task 1.1)

The following matrix records only ownership that is present in source and exercised by an existing test. A `gap` entry is intentional evidence for the audit-first scope; it is not a claim that the capability already exists.

| Concern | Current owner and source evidence | Existing test evidence | Audit finding |
| --- | --- | --- | --- |
| Scheduler task identity and lifecycle | `orchestration/scheduler/types.go:97` (`Task`), `:164` (`TaskRecord`), and `orchestration/scheduler/scheduler.go:193` (`Claim`) | `orchestration/scheduler/store_test.go:23` (`TestMemoryAndFileStoreParity`), `:1129` (`TestSchedulerLifecycleTimelineCorrelation`) | Scheduler owns task state and lifecycle correlation. No workspace reference is present on `Task` or `TaskRecord` (gap). |
| Attempt and lease generation | `orchestration/scheduler/types.go:152` (`Attempt`), `orchestration/scheduler/store_memory.go:53` and `store_file.go:128` (`Claim`), `scheduler.go:375` (`CommitTerminal`) | `orchestration/scheduler/store_test.go:308` (lease expiry/takeover), `:405` (`TestFileStoreCrashRecoveryAndTakeover`), `:853` (`TestSchedulerRestoreNoRewindRejectsTerminalTaskWithRunningAttempt`) | Attempt ID, lease token, expiry, and stale terminal commit checks are scheduler-owned. Workspace-to-attempt binding is not represented (gap). |
| Scheduler snapshot and restore | `orchestration/scheduler/types.go:441` (`StoreSnapshot`), `scheduler.go:433`/`:444` (`Snapshot`/`Restore`) | `orchestration/scheduler/store_test.go:56` (`TestStoreSnapshotRestoreRoundTrip`), `:853` (restore boundary rejection) | Scheduler snapshot/restore is authoritative for task, queue, terminal commit, and control-op state; no workspace binding is snapshotted (gap). |
| Checkpoint workspace provenance and integrity | `core/types/protocol.go:549` (`CheckpointRef`), `:589` (`WorkspaceProvenance`), `:599` (`Validate`), `:733` (`ValidateWorkspaceIntegrity`) | `core/types/protocol_test.go:59` (`TestCheckpointHistoryAndWorkspaceProvenanceValidation`); `orchestration/snapshot/protocol.go:33` and `protocol_test.go:27` (`TestProtocolCheckpointRefWithProvenanceProjectsRecoveryContext`) | Checkpoint/protocol layer owns reference-only workspace provenance and integrity validation; it does not own scheduler attempts or workspace mutation. |
| Snapshot/checkpoint import and recovery reconciliation | `orchestration/snapshot/contract.go:192` (`Importer.Import`), checkpoint association checks at `:245`-`:248`; `orchestration/composer/recovery_runtime.go` (`CaptureRecoverySnapshot`/`Recover`) | `orchestration/snapshot/contract_test.go:61` (`TestImportIdempotencyNoInflation`), `:95`/`:129` (strict/compatible restore), `orchestration/snapshot/session_history_restore_test.go:11` (`TestImporterValidatesHistoryAndCheckpointBeforeRestore`), `orchestration/composer/unified_snapshot_runtime_test.go` | Snapshot importer and composer recovery own pre-restore validation, idempotency, and scheduler restore sequencing. Workspace-to-attempt reconciliation is not yet implemented (gap). |
| Mailbox result delivery and correlation | `orchestration/mailbox/bridge.go:11` (`Correlation`), `:18` (`AsyncReport`), `:54`/`:70`/`:95` (result envelope and publish); `orchestration/mailbox/types.go:51` (`Envelope`) | `orchestration/mailbox/mailbox_test.go:38` (`TestMailboxDuplicatePublishConvergesByIdempotencyKey`), `:178` (`TestMailboxSnapshotRestoreMemoryFileParity`), `orchestration/invoke/mailbox_bridge_test.go:12` and `:79` (sync/async result publication) | Mailbox owns durable result delivery, correlation, idempotency, retry, and snapshot behavior. It does not own Run state or safe-point application. |
| Runtime-input admission and safe-point application | `core/runner/runtime_input.go:72` (`Engine.AdmitRuntimeInput`), `:121` (`ActiveRunControl.DrainRuntimeInputSafePoint`), `:149` (`popFollowUpAtBoundary`) | `core/runner/runtime_input_test.go:57` (`TestActiveRunRuntimeInputSteeringDrainsOnlyAtSafePoint`), `:110` (cancel settlement), `:132` (unknown session/closed run rejection); `core/runner/runtime_input_parity_test.go:50` (`TestRuntimeInputRunStreamSafePointAndPromotionParity`) | `core/runner` source-owned control is authoritative for admission and safe-point application, with Run/Stream parity coverage. No mailbox-completion adapter to this safe point exists yet (gap). |

## Example Impact Assessment

无需示例变更（附理由）：首阶段只验证 scheduler、recovery、mailbox 和 runtime-input 接缝，不修改 `examples/agent-modes`；若后续需要示例变化，必须先完成 `MATRIX.md` 与对应 README 文档基线。
