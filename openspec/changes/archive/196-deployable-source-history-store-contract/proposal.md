## Why

Baymax already has source-owned native history, projection ledgers, and fenced CAS primitives, but callers must assemble them independently and can accidentally omit an owner binding or advertise a stronger deployment profile than the backing store proves. The next roadmap item makes the durable source-store boundary deployable and explicit while preserving the existing same-host and non-atomic projection limits.

## What Changes

- Add an explicit owner-store factory that binds source identity, native journal, optional projection ledger, coordinator, writer identity, lease policy, and deployment request as one validated deployment profile.
- Make capability publication fail closed when the owner store is missing, unavailable, fenced by another writer, or requested beyond its declared scope; retain same-host/single-writer semantics for file-backed stores.
- Strengthen multi-writer source append semantics with stable source event identity, idempotent duplicate append, fenced CAS conflict handling, and atomic sequence/retention-floor publication.
- Provide bounded read-after-cursor and reconciliation APIs through the owner binding, retaining typed malformed, foreign, expired, gap, conflict, unavailable, and recovery-required outcomes.
- Add fresh-process, concurrent-writer, crash-before-projection, duplicate, and unavailable-store contract evidence plus equivalent shell/PowerShell gates.
- Update public documentation and the host execution-plan example to state the deployable profile, explicit factory requirement, and non-goals around cross-machine coordination and cross-project transactions.

## Capabilities

### New Capabilities

None. This change extends existing source-history and deployment contracts; it does not create a parallel history model.

### Modified Capabilities

- `durable-source-store-coordination`: require an explicit owner-store deployment binding and expose the validated writer/fencing/transaction profile before source activation.
- `native-execution-event-history`: add owner-bound source history construction as the canonical deployable path, stable idempotent append identity, and capability evidence tied to the selected store profile while preserving existing explicitly injected journal compatibility.
- `source-history-projection-transaction`: require owner-bound journal/ledger reconciliation and explicit non-atomic cross-project projection semantics.
- `source-history-failure-semantics`: require unavailable/recovery-required outcomes when the bound store or source append commit cannot be confirmed.

## Impact

- `core/history` and `core/sourcecoordination` gain the public owner-store binding and deployment validation seam.
- Runner, child, team, workflow, mailbox, and planbridge capability evidence can consume one validated source-owner profile without changing execution ownership.
- New contract fixtures/gates cover same-host multi-instance CAS where implemented, and explicitly reject unsupported cross-machine or unconfigured profiles.
- Public semantic documentation changes in `README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `core/types/README.md`, and `examples/host-execution-plan/README.md`.

## Example Impact Assessment

修改示例

理由：host execution plan 示例需要展示显式 owner-store factory/deployment profile，并保留未配置、跨机器和 projection 非原子边界的拒绝标记。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/types/README.md`、`examples/host-execution-plan/README.md`
- 维护 owner：runtime contracts owner、source history owner、examples/documentation maintainer
- 验证证据：`go test ./core/history ./core/sourcecoordination ./core/types`、`pwsh -File scripts/check-source-history-store-contract.ps1`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | owner-store deployment binding tests |
| components | 修改文档 | `core/types/README.md` | source history owner | capability and reconciliation tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | fail-closed profile checks |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | typed error and factory tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | source-history gate evidence |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples/documentation maintainer | deployment profile matrix |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | owner-store typed admission/reconciliation evidence |
| best practices | 修改文档 | `README.md` | documentation maintainer | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap status gate |
