## Context

The source-history and continuation owners already persist bounded state and
can reopen a durable snapshot in a new process. Their public contracts do not
yet distinguish that sequential local guarantee from multi-writer CAS,
cross-machine coordination, or concurrent resume claims. The implementation
must preserve the existing owner boundaries and keep mailbox history,
scheduler state, runner checkpoints, and composite graph state separate.

## Goals / Non-Goals

**Goals:**

- Make deployment scope, writer ownership, fencing, and transaction scope
  explicit in source-owned capability evidence.
- Admit only same-host single-writer sequential reopen with the evidence the
  current stores actually provide.
- Reject stale, foreign, duplicate, and unsupported writers before side
  effects, with stable typed classifications and replay fixtures.
- Keep capability, history, checkpoint, composite, and planbridge projections
  consistent and source-specific.

**Non-Goals:**

- Building a distributed database, remote event journal, queue, worker,
  coordinator, or hosted control plane.
- Adding multi-writer CAS, writer fencing across machines, distributed
  transactions, or multi-instance resume support.
- Replacing source-owned stores with one unified persistence layer.
- Changing model/tool execution, business authorization, or terminal outcome
  arbitration.

## Decisions

### 1. Use a shared vocabulary, not a shared store

Add bounded manifest fields for `deployment_profile`, `writer_scope`,
`fencing_scope`, and `transaction_scope`. The canonical supported profile is
`same_host_single_writer_sequential_reopen`; unsupported profiles remain
explicit cells with reasons. Each owner projects these fields from its own
store and keeps its own identity namespace.

**Alternative rejected:** adding a central deployment registry or shared
store. That would create a new owner and make a capability declaration look
like coordination evidence.

### 2. Treat ownership as an admission precondition

Persisted snapshots and checkpoint refs carry an owner/fence reference. Store
operations validate it before mutation or claim. A second writer, stale fence,
foreign source, or missing transaction capability returns a typed failure. The
plan bridge and replay tooling consume the same declaration but never create a
fence or claim on behalf of an owner.

**Alternative rejected:** optimistic last-writer-wins. It would allow lost
cursor advancement and duplicate resume while making the failure invisible to
hosts.

### 3. Separate local restart evidence from distributed evidence

Contract fixtures cover four distinct results: local sequential reopen,
duplicate writer rejection, stale/foreign owner rejection, and unsupported
cross-machine or multi-instance request. Successful local reopen is never
promoted to a distributed capability. Replay remains read-only and reports the
same distinction.

**Alternative rejected:** infer scope from `FileStore` atomic rename or a
successful `Load`; those facts do not establish reader/writer coordination.

### 4. Keep unsupported behavior fail closed

Admission checks deployment scope before publish/enqueue/claim/restore. On
store failure, the caller gets `unavailable` or `unsupported_capability`; no
fresh parent, re-enqueue, retry, latest fallback, or history replay is used as
recovery. Existing same-host behavior remains unchanged.

## Risks / Trade-offs

- [Risk] Consumers may have depended on implicit multi-instance behavior.
  -> Make the new fields additive, default existing stores to their truthful
  local profile, and provide stable diagnostics explaining the limitation.
- [Risk] Owner projections drift from each other.
  -> Use one versioned fixture vocabulary and require independent source cells
  in the capability and history/continuation contract gates.
- [Risk] A fence reference could leak authority or payload data.
  -> Keep it opaque, bounded, source-owned, and exclude business authority,
  credentials, and application payloads.
- [Risk] Existing tests may assert only successful reopen.
  -> Add negative duplicate/stale/foreign/unsupported cases and Run/Stream
  parity checks before task completion.

## Migration Plan

1. Add additive types and projections with the local sequential profile as the
   only supported default.
2. Add negative fixtures and shell/PowerShell gates before enabling any new
   admission checks.
3. Update package and integration documentation and verify status parity.
4. Roll back by removing the new manifest cells and gates; no persisted format
   migration is required because ownership metadata is bounded and additive.

## Example Impact Assessment

无需示例变更（附理由）：the runtime examples do not request concurrent or
distributed persistence. The change is demonstrated by capability fixtures and
offline replay/contract gates; existing example markers and execution paths
remain unchanged.

## Public Semantic Documentation Commitment (draft)

对外语义变化：是。 This design changes the public interpretation of durable
reopen, writer ownership, unsupported deployment scope, and admission errors.
The implementation must update `docs/guides/integration.md`,
`docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`,
`docs/mainline-contract-test-index.md`, the affected package READMEs, and the
five delta specs in this change. Owners are the source packages named in the
proposal. Verification uses the dedicated shell/PowerShell gates,
`go test ./...`, `go test -race ./...`, `golangci-lint`, OpenSpec validation,
and documentation consistency checks.

## Documentation Impact Assessment

| Area | Outcome | Maintained path | Owner | Evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | source owners | deployment scope gate |
| components | 修改文档 | `core/types/README.md` | core/types | Go contract tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics owner | docs consistency gate |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md` | planbridge owner | OpenSpec validation |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics owner | replay fixture |
| examples | 修改文档 | `docs/guides/integration.md` | integration owner | integration guide |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration owner | contract gate |
| best practices | 修改文档 | `docs/guides/integration.md` | source owners | fail-closed tests |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | roadmap status gate |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `docs/development-roadmap.md`, `core/types/README.md`, `core/runner/README.md`, `orchestration/README.md`, `orchestration/planbridge/README.md`, `orchestration/mailbox/README.md`, `orchestration/scheduler/README.md`, `orchestration/composer/README.md`, `orchestration/workflow/README.md`
- 维护 owner：core/types, core/checkpoint, core/history, orchestration source owners, planbridge
- 验证证据：deployment scope Go tests, replay fixture, shell/PowerShell contract gates, docs consistency, OpenSpec validation
