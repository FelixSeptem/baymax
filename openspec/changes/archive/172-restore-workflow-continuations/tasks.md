## 1. Baseline and Source Ownership

- [x] 1.1 Audit workflow graph, checkpoint, scheduler/A2A and existing direct continuation owners; record source boundaries and typed unsupported cells in workflow and capability tests.
- [x] 1.2 Freeze bounded graph continuation schema, canonical digest inputs, identity/causation fields and same-host single-writer limits; verify exact-boundary and oversize tests.

## 2. Workflow Graph Continuation Material

- [x] 2.1 Add detached workflow checkpoint/reference and persisted graph-state DTOs covering graph identity/version/digest, parent/run/session/causation, dependencies, pending/completed steps, budgets, pause boundary, effect references, terminal arbitration and predecessor/successor; verify serialization and digest tests.
- [x] 2.2 Persist expanded subgraph and A2A step state without changing deterministic planning; verify alias-path, dependency and graph-digest drift tests.
- [x] 2.3 Add source-owned save/load/commit transitions and typed corrupt/foreign/expired/unavailable errors; verify reopen and crash-window tests on the configured store.

## 3. Atomic Restore and Lifecycle Races

- [x] 3.1 Implement workflow restore admission and atomic claim with full normalized resume identity, binding/revision/budget validation and source generation recheck; verify concurrent duplicate and conflicting claim tests.
- [x] 3.2 Restore the same native graph after new Engine/process reopen and skip terminal steps/effects; verify no parent Run, enqueue, history replay or duplicate effect is created.
- [x] 3.3 Define cancel/timeout/retry/resume precedence and crash-after-claim/start/terminal projections; verify in-progress/recovery-required and terminal-evidence tests without automatic retry.

## 4. Pause Reason Producers

- [x] 4.1 Implement workflow `input_required` safe-point persistence and restore; verify pending input, dependency state, source evidence and Run/Stream parity.
- [x] 4.2 Implement workflow `approval_required` before effect dispatch with deny/timeout terminal evidence; verify pending action is not dispatched and duplicate approval cannot create a second effect.
- [x] 4.3 Implement workflow `budget_exhausted` persistence with consumed resources and original deadline; verify current budget revalidation, no replenishment/reset and deterministic resume.

## 5. Capability Manifest and Unsupported Matrix

- [x] 5.1 Add independent workflow checkpoint/restore/pause cells with contract/API version, build revision, evidence, retention/expiry and storage limitation; verify manifest fixture coverage.
- [x] 5.2 Keep child/fanout, team parallel/vote, mailbox host plan and planbridge resumed cells explicitly `unsupported_capability`; verify rejection before enqueue/publish/Run/Stream/member/step activation.
- [x] 5.3 Verify planner cannot infer support from package presence, DTOs, workflow history, callback IDs or mailbox capability; verify typed unavailable versus unsupported distinctions.

## 6. Replay, Integration, and Gates

- [x] 6.1 Add versioned reference-only workflow continuation fixtures for graph drift, duplicate/conflict, foreign/stale/expired/corrupt, crash windows, cancel/timeout/retry races, terminal evidence and unsupported forms; verify replay is deterministic and side-effect free.
- [x] 6.2 Add Run/Stream integration parity and new-process restore tests for all supported workflow pause cells; verify source identity, graph state and terminal outcome equivalence.
- [x] 6.3 Add dedicated shell and PowerShell workflow continuation contract gates and wire them into quality/replay paths; verify both paths reject intentional fixture drift.

## 7. Documentation and Example

- [x] 7.1 Update integration, module-boundary, runtime-diagnostics and contract-index docs with workflow-first scope, source ownership, storage limits and unsupported matrix; verify declared paths are changed and links resolve.
- [x] 7.2 Update the workflow package README and durable continuation example markers/rollback notes without adding a parallel execution semantic; verify the offline example remains provider-free.
- [x] 7.3 Update roadmap active status and archive references only through OpenSpec status evidence; verify roadmap consistency and documentation impact gates.

## 8. Verification and Release

- [x] 8.1 Run focused workflow, checkpoint, capability and replay tests including positive, negative, boundary and race cases; record evidence in the change verification record.
- [x] 8.2 Run `openspec validate restore-workflow-continuations --strict`, `git diff --check`, documentation/example/roadmap gates and package contract gates; verify no gate treats replay as restore.
- [x] 8.3 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; record any environment-limited command and reason.

## Example Impact Assessment

修改示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/workflow/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：workflow/runtime、diagnostics/replay、documentation maintainers
- 验证证据：`openspec validate restore-workflow-continuations --strict`；`pwsh -File scripts/check-openspec-documentation-impact.ps1`；`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | workflow/runtime | source-owner boundary review |
| components | 修改文档 | `orchestration/workflow/README.md` | workflow/runtime | package contract review |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | workflow/runtime | configuration boundary review |
| contract/API | 修改文档 | `docs/guides/integration.md` | workflow/runtime | OpenSpec strict validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay | replay gate mapping review |
| examples | 修改文档 | `examples/durable-continuation-restore/README.md` | documentation maintainers | expected-marker review |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | workflow/runtime | admission boundary review |
| best practices | 修改文档 | `docs/guides/integration.md` | documentation maintainers | restore safety review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status gate |
