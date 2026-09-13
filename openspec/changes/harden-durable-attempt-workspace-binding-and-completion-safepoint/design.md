## Context

The existing scheduler owns durable task, attempt, lease, retry, async-report, and terminal-commit state. Checkpoint and protocol layers already expose reference-only workspace provenance and integrity validation. Mailbox owns durable completion delivery, while `core/runner` owns bounded runtime-input admission and safe-point application. The missing evidence is the contract seam between these owners: scheduler attempts do not carry an explicit workspace binding, and no replayable fixture proves that a durable completion reaches the next model decision exactly once across duplicate, late, disconnect, and recovery boundaries.

## Goals / Non-Goals

**Goals:**

- Establish a testable, additive contract for task/attempt workspace association and lease-generation isolation.
- Define deterministic validation and replay classification for workspace association, integrity, checkpoint, retry, and recovery drift.
- Prove completion-to-safe-point ownership using existing mailbox, scheduler terminal commit, runtime-input, and terminal-outcome owners.
- Preserve Run/Stream semantic parity and bounded `RuntimeRecorder` diagnostics.

**Non-Goals:**

- No Git/worktree manager, workspace registry, hosted artifact service, or runtime Git/shell execution.
- No new global task/session/coordination state machine, notification queue, or provider-facing completion protocol.
- No automatic merge/push and no persistence of workspace contents or raw reasoning.

## Decisions

### 1. Fixture-first before runtime model expansion

First add bounded scheduler/recovery/mailbox/runtime-input fixtures, replay normalization, and gates. Only a fixture that demonstrates a reproducible stale binding, integrity drift, or lost/duplicated promotion may justify additive runtime fields or interfaces. This keeps the change audit-first and avoids speculative schema expansion.

Alternative: add workspace fields to every scheduler type immediately. Rejected because the current code has no demonstrated producer/consumer contract for those fields and would risk dead data or a second source of truth.

### 2. Reference-only workspace binding

Workspace binding reuses `types.WorkspaceProvenance` identifiers and integrity references. Scheduler and recovery store references and correlations only; host/tool adapters remain responsible for resolving and observing workspace state. Missing, dirty, conflict, checkpoint mismatch, and integrity drift are distinct classifications.

Alternative: let scheduler inspect or mutate workspaces. Rejected by module boundaries and library-first scope.

### 3. Existing owners remain authoritative for completion

Mailbox delivery and scheduler terminal commits remain durable delivery/state owners. The source Runtime remains the only owner that can admit or apply runtime input at a model decision safe point. A completion adapter may translate bounded correlation metadata, but it cannot inject into an active operation or create a parallel pending/terminal FSM.

Alternative: create a dedicated completion queue/state machine. Rejected because it duplicates mailbox and runtime-input semantics and would create ambiguous terminal ownership.

### 4. Strict versus compatible recovery follows existing snapshot policy

Strict recovery fails before mutation on workspace association or integrity violations. Compatible recovery may skip or bound an affected segment only within the existing compatibility window and must record a deterministic downgrade. Snapshot/checkpoint source ownership and idempotency remain unchanged.

### 5. Replay and diagnostics are bounded

Replay consumes offline fixtures with identifiers, hashes, classifications, and bounded payload metadata only. Diagnostics use `RuntimeRecorder`; raw workspace contents, completion bodies, credentials, and reasoning text are excluded.

## Data Flow

```text
task + optional workspace provenance
        │
        ▼
claim → attempt/lease binding → retry/rollover validation
        │                         │
        │                         └─ stale/conflict/drift classification
        ▼
snapshot/recovery integrity reconciliation

mailbox result → scheduler terminal/idempotency
        │
        ▼
source-owned runtime-input admission
        │
        ▼
next safe point → one Run/Stream-equivalent model decision
```

## Risks / Trade-offs

- **[Risk]** Existing legacy tasks omit workspace references. → Keep all new fields optional and preserve legacy behavior; add explicit “binding absent” fixture coverage.
- **[Risk]** Retry policy may differ between workspace reuse and rebinding. → Require an explicit policy outcome in the fixture and reject implicit rollover.
- **[Risk]** Completion can arrive after timeout or disconnect. → Reuse terminal arbiter, scheduler late-report policy, and runtime-input admission classifications; never resurrect a closed Run.
- **[Risk]** Full quality gate may exceed the local Windows budget. → Run focused fixture/replay/gate suites first, record environment or budget blockers, and keep the independent gate deterministic.

## Migration Plan

No persisted-data migration is required for the audit-first slice. Existing snapshots remain readable because new references are additive and nullable. If a later fixture-driven implementation adds fields, legacy snapshots default to “binding absent” and strict validation applies only when a binding is supplied.

## Open Questions

None for the proposal phase. Whether retry reuses a workspace after integrity validation or requires a new binding is intentionally a fixture decision that must be selected and encoded before runtime implementation tasks are approved.

## Example Impact Assessment

无需示例变更（附理由）：本提案首阶段只覆盖 scheduler、snapshot/recovery、mailbox、runtime-input、replay 和 diagnostics 接缝，不修改 `examples/agent-modes`；若后续实现需要示例，必须先更新 `MATRIX.md` 与对应模式 README 的文档基线。
