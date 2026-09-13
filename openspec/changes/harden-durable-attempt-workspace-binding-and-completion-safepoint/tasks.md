## Example Impact Assessment

无需示例变更（附理由）：首阶段只验证 scheduler、snapshot/recovery、mailbox、runtime-input、replay 与 diagnostics 接缝，不修改 `examples/agent-modes`；若实现中发现必须调整示例，先更新 `MATRIX.md` 与对应模式 README 后再进入示例代码任务。

## 1. Baseline Audit and Gap Fixtures

- [x] 1.1 Record the current scheduler task/attempt/lease, checkpoint workspace provenance, snapshot/recovery, mailbox result, and runtime-input safe-point ownership matrix in the proposal/design evidence; verify every claimed owner is linked to source code and an existing test.
- [x] 1.2 Define a bounded `durable_attempt_workspace_binding.v1` fixture schema for absent binding, valid binding, lease rollover, retry reuse/rebind, stale commit, missing/dirty/conflict/drift, checkpoint association, and recovery reconciliation; verify malformed, oversized, and unknown-version inputs fail deterministically.
- [x] 1.3 Define a bounded `completion_safe_point_ownership.v1` fixture schema for correlation, duplicate, late timeout, disconnect, recovery, not-applied, and Run/Stream parity; verify raw completion bodies and reasoning content are excluded.
- [x] 1.4 Add red tests that demonstrate the current task/attempt workspace gap and completion-to-safe-point ownership gap before any runtime field or API change; verify each test fails for the intended missing contract rather than unrelated setup.

## 2. Task-Attempt Workspace Binding

- [ ] 2.1 Select and document explicit retry semantics—reuse only after integrity validation or rebind with rollover correlation—based on gap fixture evidence; verify no implicit workspace choice remains in fixture expectations.
- [ ] 2.2 Add the minimal optional reference-only workspace binding to scheduler task/attempt/commit surfaces proven necessary by the fixtures; verify legacy records and snapshots without the fields retain existing behavior.
- [ ] 2.3 Enforce task/attempt/lease/workspace correlation on claim, heartbeat, retry, rollover, and terminal commit; verify stale attempt and mismatched workspace results cannot mutate the current task.
- [ ] 2.4 Extend scheduler snapshot and composer recovery normalization with binding references and pre-mutation integrity reconciliation; verify strict fail-fast and compatible bounded downgrade behavior for missing, dirty, conflict, checkpoint mismatch, and integrity drift.
- [ ] 2.5 Add memory/file backend and snapshot round-trip tests for positive, negative, boundary, retry, lease-expiry, manual-retry, recovery, and idempotency cases; verify backend parity.

## 3. Completion Safe-Point Ownership

- [ ] 3.1 Define the minimal bounded adapter from mailbox/scheduler terminal completion correlation to the existing source-owned runtime-input envelope; verify it does not create a new queue, terminal arbiter, cursor, or Run state machine.
- [ ] 3.2 Apply accepted completion input only at the existing next-decision or idle/terminal boundary; verify no active provider, tool, or HITL operation is mutated mid-operation.
- [ ] 3.3 Reconcile duplicate completion, duplicate promotion, late timeout, terminal conflict, disconnect, backpressure, and not-applied outcomes through existing scheduler and runtime-input classifications; verify the authoritative terminal outcome remains immutable.
- [ ] 3.4 Persist and restore only bounded pending/applied completion references needed for reconciliation; verify repeated recovery applies at most one logical completion and never resurrects a closed Run.
- [ ] 3.5 Add equivalent Run and Stream integration tests for completion admission, safe-point application, duplicate, late, disconnect, terminal race, and recovery; verify normalized outcomes differ only by permitted event ordering.

## 4. Replay, Diagnostics, and Gates

- [ ] 4.1 Implement offline, side-effect-free replay for both fixture namespaces and classify schema, binding, association, integrity, stale-attempt, completion-correlation, duplicate, late, disconnect, recovery, and Run/Stream parity drift; verify replay never invokes providers, tools, Git, or workspace mutation.
- [ ] 4.2 Add `RuntimeRecorder` diagnostics tests for bounded additive workspace/completion identifiers and reason codes; verify raw workspace contents, completion bodies, reasoning, credentials, and unbounded payloads are absent.
- [ ] 4.3 Add shell and PowerShell contract gates with equivalent fixture, replay, source-owner, architecture-boundary, idempotency, and parity checks; verify removing required evidence makes each gate fail.
- [ ] 4.4 Register the independent gate in `scripts/check-quality-gate.*` and update `docs/mainline-contract-test-index.md`; verify shell/PowerShell gate parity and docs consistency.

## 5. Documentation and Scope Review

- [ ] 5.1 Update `README.md`, `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, and relevant scheduler/snapshot/replay READMEs with ownership, compatibility, privacy, rollback, and no-new-config decisions; verify status consistency.
- [ ] 5.2 Review implementation against non-goals and architecture constraints; verify no Git/worktree manager, runtime Git/shell execution, hosted workspace/artifact service, automatic merge/push, raw-content diagnostics, or parallel task/session/coordination FSM was introduced.
- [ ] 5.3 Re-evaluate Example Impact Assessment after fixture-driven implementation; if examples remain untouched, document the reason, otherwise complete the mandatory `MATRIX.md` and mode README baseline before example code.

## 6. Integrated Verification and Handoff

- [ ] 6.1 Run focused scheduler, mailbox, composer recovery, snapshot, runner runtime-input, diagnostics replay, and Run/Stream integration suites; record positive, negative, boundary, idempotency, and backend-parity evidence.
- [ ] 6.2 Run `openspec validate --all`, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; record exact results and any environment-blocked command.
- [ ] 6.3 Archive with `scripts/openspec-archive-seq.ps1`, update roadmap/README/archive index status, merge the proposal branch to the latest `master`, verify the merged result, push, and delete the merged local branch/worktree.
