## Context

See `proposal.md` for motivation. Archive 168 defined the source-neutral
history reader and implemented mailbox history, but runner, isolated child,
team, and workflow remain explicitly unsupported. The implementation must keep
source ownership with each existing runtime owner, keep `host/eventadapter`
real-time-only, and preserve the same-host single-writer storage boundary.

## Goals / Non-Goals

**Goals:**

- Turn the source audit into an independently testable capability declaration
  for all five source kinds.
- Add source-owned journal append/read paths where an owner can provide a real
  retained history, with bounded source-local cursors and fail-closed errors.
- Prove new-process replay, terminal evidence, overlap handling, and no-side-
  effect behavior without using replay as execution recovery.
- Keep manifest, docs, fixtures, and gates aligned with actual source evidence.

**Non-Goals:**

- Composite checkpoint/restore, pause continuation, graph restore, or resumed
  planbridge execution; those belong to the next roadmap candidate.
- Multi-writer CAS, cross-machine coordination, distributed journals, or a
  unified mailbox/runner store.
- A new queue, scheduler, event service, eventadapter journal, or diagnostics
  fact store.

## Decisions

### 1. One public reader, source-owned implementations

Reuse the existing source-neutral `NativeEventHistoryReader` and DTO/error
surface. Each owner implements a narrow reader over its own journal and returns
detached pages. The reader accepts only bounded source references and opaque
cursors; it receives no handler, runner, scheduler, model, tool, or restore
dependency.

Alternative: add a generic history service in planbridge or eventadapter.
Rejected because those layers do not own every source transition and would
create a second event authority.

### 2. Capability audit precedes source support

The manifest contains independent history and after-cursor cells for runner,
child, team, workflow, and mailbox. An owner may report supported only when its
journal, cursor, retention and restart evidence exist. Otherwise it reports
`ErrHistoryUnsupported` or `ErrSourceUnavailable` with a stable reason.

Alternative: infer support from a form or public DTO. Rejected because it would
reintroduce the exact false-positive contract this change is intended to remove.

### 3. Source journal identity and ordering

Each source journal uses a stable event identity namespace and a source-local
monotonic sequence whose scope is declared in the capability. Event records are
bounded lifecycle/terminal facts and preserve native run, parent, member/step,
attempt, causation and host-correlation references where the owner already
defines them. Payloads, credentials, callback errors and business commits are
excluded. Parallel sources are never merged into a synthetic global order.

### 4. Retention and cursor validation are fail-closed

An opaque cursor binds schema/version, source namespace/instance and an
after-sequence position. The reader validates shape, source scope, retention
floor and sequence continuity in that order. It returns typed malformed,
foreign, expired, gap, or unavailable outcomes and never substitutes latest or
floor data after an error. Pages are produced from one owner-consistent read
snapshot so pruning cannot produce a partial page.

### 5. Replay is structurally read-only

Replay reads immutable journal snapshots and returns detached DTOs. It cannot
claim a lease, renew or acknowledge delivery, enqueue work, retry, DLQ, invoke a
handler, write diagnostics, call a model/tool, or start/resume a run. Fixture
replay uses pure parsers and does not construct source owners. Consumer-side
overlap deduplication is keyed by source event identity; same identity with a
different payload is a conflict.

### 6. Storage scope is explicit

The default implementation supports a single writer on one host and a later
process reopening persisted state where that backend already guarantees it.
Memory-only sources remain process-local. No implementation or manifest value
will imply multi-writer CAS, cross-host fencing, distributed cursor
transactions, or multi-instance resume claims. A later distributed-store
proposal must add those guarantees and evidence independently.

## Risks / Trade-offs

- **Journal leaks unbounded or sensitive data** → append only bounded public
  lifecycle facts, validate sizes, redact payload/callback errors, and test
  negative cases.
- **Pruning races a reader** → read from an owner-consistent snapshot and return
  explicit expiry/gap rather than partial or latest data.
- **Source capability drifts from implementation** → require fixture evidence
  references and manifest contract gates per source/form.
- **Replay accidentally mutates execution** → use read-only interfaces and
  counting fakes proving no publish, claim, ack, retry, Run, Stream, tool,
  model, handler, or diagnostics call.
- **Different owners invent incompatible cursors** → keep cursor opaque and
  require namespace/version/scope declarations plus source-specific validation.

## Migration Plan

1. Add or update source declarations and unsupported reasons without changing
   existing live callback or execution behavior.
2. Implement source-owned journal append/read for each owner that can provide
   the declared guarantee; keep unsupported owners explicit.
3. Add restart, cursor, retention, conflict, terminal, and no-side-effect tests
   before changing any manifest support cell.
4. Add fixtures, shell/PowerShell gates, integration documentation and the
   offline example; verify all declared documentation paths are modified.
5. Roll back by withdrawing the affected history capability cell or exposing
   source unavailable. Do not delete source history, rewrite cursors, retry
   execution, or alter business effects.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`
- 维护 owner：roadmap maintainers；实施阶段由 runtime-core、source-owner、mailbox、host-integration、observability、documentation 与 examples maintainers 同步其余公开文档
- 验证证据：OpenSpec strict validation、documentation/example-impact 与 roadmap consistency gates、source-history contract/replay gates、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| components | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| configuration | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| contract/API | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| diagnostics | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| examples | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| CLI/integration | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| best practices | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |

## Example Impact Assessment

修改示例。`examples/native-execution-events/README.md` 与离线示例将展示 source matrix、确认 cursor 后重启读取、typed cursor errors 和 replay 无副作用边界。
