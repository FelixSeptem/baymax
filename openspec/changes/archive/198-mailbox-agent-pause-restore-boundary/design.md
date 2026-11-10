## Design

### Owner and scope

`orchestration/mailbox` remains the sole owner of message, delivery, lease,
attempt, continuation claim, retry, acknowledgement, and DLQ facts. Handler
continuation is opt-in through `NewContinuationWorker`; `NewWorker` continues to
be lifecycle-only. A configured worker must use a durable `FileStore`, a
registered resumer, and a current revalidator. The supported deployment profile
is `same_host_single_writer_sequential_reopen`; memory stores, multi-writer
claims, cross-machine coordination, and concurrent multi-instance recovery are
unsupported and must be declared as such.

### Checkpoint and resume model

The existing bounded `MailboxContinuation` is the checkpoint envelope. It binds
the original host-plan identity, message/delivery/lease/attempt causation,
consumer, pause reason, pending action, completed effect references, consumed
budget, deadline, expiry, and predecessor/successor lineage. Opaque handler
state remains a reference and is never persisted as payload, credential,
callback error, or business-commit evidence.

`RunOnce` persists a pause while retaining the original in-flight delivery and
does not ack, nack, requeue, publish, or consume a replacement. Resume validates
the deployment and current authority, atomically claims the original record,
heartbeats the same delivery, and invokes the registered resumer once. A
resumer that pauses again creates a successor continuation with incremented
generation and predecessor linkage before the current claim is terminally
projected; it never restarts the handler from the beginning. Exact duplicate
requests return the retained claim/result projection.

### Capability evidence

`ContinuationCapabilities` publishes checkpoint creation, cross-process
restore, handler safe-point, deployment scope, and independent pause cells. The
three canonical pause reasons each require the same fixture reference plus
fresh-process restore, lease/revalidation, terminal, duplicate-claim, and
completed-effect non-replay evidence. Lifecycle-only owners retain the stable
`mailbox_handler_safe_point_unsupported` reason for every handler pause cell.

The manifest may promote only the configured mailbox owner. History replay,
host-plan lifecycle continuation, package presence, or another execution form
cannot promote handler pause cells. Planbridge validates the manifest and
delegates to the existing worker; it never constructs a worker/resumer,
consume loop, queue, scheduler, or fallback direct execution.

### Failure and terminal semantics

Malformed, foreign, stale, expired, unavailable, lease-lost, revalidation-
denied, ambiguous, and deployment-incompatible resumes fail closed before
handler/effect invocation. A resumer error follows the existing handler error
policy and preserves retry/DLQ causation. A persistence failure or crash after
claim but before a committed result is `recovery-required`; it is not converted
to a retry or a fresh handler run. Ack/requeue/nack/DLQ and terminal outcome
remain source lifecycle facts and remain distinct from host business commit.

## Example Impact Assessment

修改示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md`, `examples/host-execution-plan/README.md`
- 维护 owner：orchestration/runtime maintainers
- 验证证据：`pwsh -File scripts/check-mailbox-agent-pause-restore-contract.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `go test ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay -count=1`

## Documentation Impact Assessment

| 文档域 | 变更 | 路径 | Owner | 证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | orchestration/runtime maintainers | owner and deployment boundary |
| components | 修改文档 | `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md` | orchestration maintainers | package contract and delegation boundary |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | states that no runtime keys/defaults changed |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` | orchestration/runtime maintainers | public matrix and test index |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | states that no diagnostics or QueryRuns fields changed |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples maintainers | offline marker and support matrix |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration maintainers | host integration boundary |
| best practices | 修改文档 | `README.md` | runtime maintainers | capability summary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | active status and scope |

## Test strategy

- Table-driven fresh-process coverage for all three pause reasons.
- Explicit counters proving completed handler/effect work is not re-run.
- Negative tests for lease loss, expiry, foreign deployment, revalidation
  denial, persistence/recovery-required state, duplicate resume, and lifecycle-
  only workers.
- Success and failure paths for ack, requeue, nack, retry exhaustion, and DLQ.
- Planbridge pre-publish rejection and Run/Stream parity for the configured
  owner binding.
- Offline fixture/gate scans enforce owner boundaries and stable unsupported
  reasons.
