## Why

Baymax now exposes source history and durable continuation contracts, but their
deployment guarantees are easy to overread: a reopenable `FileStore` is not a
multi-writer store, and a new process reading a persisted snapshot is not proof
of cross-machine coordination or duplicate-resume fencing. Danny's recovery
contract needs a signed, source-owned statement of those limits before a host
can claim general durable recovery. This change makes the supported boundary
explicit and fail-closed without pretending to deliver a distributed store.

## What Changes

- Add a versioned deployment/concurrency scope to source-history,
  continuation, checkpoint, and mailbox capability evidence.
- Declare the default supported profile as same-host, single-writer,
  sequential reopen of persisted state; distinguish process restart from
  concurrent or distributed recovery.
- Add source-owned writer ownership/fencing facts and typed unavailable or
  unsupported outcomes when a second writer, foreign writer, stale writer, or
  unsupported deployment scope is presented.
- Make resume claims, cursor advancement, and source snapshot writes advertise
  whether they are atomic only within one writer or transactionally coordinated
  across writers; never infer the latter from a file rename or reopen.
- Add fixtures, replay evidence, integration documentation, and shell/
  PowerShell contract gates for local reopen, duplicate-writer rejection,
  stale-owner rejection, and unsupported cross-machine/multi-instance paths.
- Keep mailbox lifecycle history and runner/composite continuation stores as
  separate source-owned stores; do not introduce a unified database, queue,
  remote journal, hosted control plane, or best-effort retry fallback.

## Example Impact Assessment

无需示例变更（附理由）：本 change changes capability/deployment claims and
failure evidence only. Existing examples continue to use the already-supported
same-host single-writer profile and no runtime execution path or semantic
markers change. The integration guide and capability fixtures are updated
instead.

### Public Semantic Documentation Commitment

对外语义变化：是。公开 capability, error, and deployment-scope semantics
change. The same change updates:

- `docs/guides/integration.md` (deployment and ownership boundary),
- `docs/runtime-module-boundaries.md` (source-store ownership),
- `docs/runtime-config-diagnostics.md` (unsupported scope diagnostics),
- `docs/mainline-contract-test-index.md` (new contract/replay gates), and
- the affected package READMEs and capability specs.

The implementation owner is the relevant source owner (`core/checkpoint`,
`core/history`, `orchestration/scheduler`, `orchestration/mailbox`, and
`orchestration/planbridge`); verification is provided by the dedicated
PowerShell and shell gates plus `go test -race ./...` and documentation
consistency checks.

## Capabilities

### New Capabilities

- None. The change tightens deployment claims for existing source-owned
  stores and capabilities rather than introducing a new runtime capability.

### Modified Capabilities

- `host-runtime-capability-manifest`: add deployment scope, writer ownership,
  and unsupported-scope evidence to capability cells.
- `native-execution-event-history`: define source-journal persistence and
  cursor advancement guarantees for same-host single-writer operation.
- `durable-native-continuation-restore`: define checkpoint persistence,
  resume-claim ownership, and cross-process versus concurrent restore scope.
- `composite-continuation-restore`: bind composite graph restore claims to the
  same store ownership and deployment boundary.
- `host-execution-plan-bridge`: reject mailbox/plan continuation admission when
  its owner cannot satisfy the declared deployment scope.

## Impact

- Public capability and error DTOs in `core/types`, `core/history`, and
  `core/checkpoint`.
- Existing scheduler, mailbox, composer, workflow/team, and planbridge store
  adapters and their capability projections.
- Contract fixtures, diagnostics/replay tooling, integration guidance, package
  READMEs, and quality-gate wiring.
- No new external dependency, scheduler, queue, mailbox owner, remote store,
  or distributed coordination service.

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
