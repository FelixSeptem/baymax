## Context

See `proposal.md` for the motivation and `specs/` for the externally observable
contract. Spec 1 already supplies a versioned capability manifest and makes
mailbox lifecycle facts source-owned, but its history/replay dimensions are
explicitly unsupported. `host/eventadapter` projects live `types.EventHandler`
callbacks and deliberately has no retained source sequence or durable cursor.
Current runner, child, team and workflow paths have no persistent source journal
that a new process can interpret without re-entering execution.

The design must retain existing module boundaries: core public types remain
source-neutral and standard-library-only; the mailbox owner retains mailbox
lifecycle ownership; eventadapter is not a store; diagnostics remain written
only through `observability/event.RuntimeRecorder`; and replay cannot mutate any
execution or effect source.

## Goals / Non-Goals

**Goals:**

- Define source-neutral, bounded public history DTOs, reader interface,
  capability facts, opaque cursor validation and typed errors.
- Audit all five source kinds and publish truthful support/unsupported facts.
- Deliver a durable, source-owned mailbox journal plus read-after-cursor replay
  that survives a new reader/consumer process.
- Preserve mailbox lifecycle/attempt/terminal identities and prove cursor replay
  is read-only through integration and offline evidence.
- Update the capability manifest only after actual source behavior and evidence
  exist; preserve explicit unsupported values elsewhere.

**Non-Goals:**

- Retained history for runner, child, team or workflow in this change.
- Execution resume, checkpoint restore, retry/recovery automation, host outbox
  ownership, business commit, a global event order, or a cross-source cursor.
- A new history service, queue, scheduler, subscription worker, eventadapter
  store, diagnostics store, runtime configuration key, or Provider dependency.
- Storing payloads, credentials, raw callback errors, host observation records,
  or arbitrary business data in the journal.

## Decisions

### 1. Publish a source-neutral reader contract; source owners provide behavior

`core/types` will define bounded immutable public values equivalent to
`SourceRef`, `HistoryCapabilities`, `OpaqueCursor`, `NativeEventPage`, and
`NativeEventHistoryReader`. A reader accepts a source reference and optional
cursor, returns a detached page bounded by a caller limit and source maximum,
and exposes only source-owned native event facts. The typed errors have stable
sentinel/category identity so callers can distinguish unsupported, malformed,
foreign, expired, gapped and unavailable states without parsing messages.

`SourceRef` includes only bounded source kind, namespace and source identity;
it does not accept an arbitrary Run ID as a substitute source reference. Cursor
bytes are opaque to callers and encode a versioned mailbox source scope and
position. The reader validates shape/bounds before source lookup, source
namespace/instance before position, then retention state before page assembly.
`limit` is strictly positive and capped; all returned nested values are detached.

Alternative considered: define a generic journal in `host/eventadapter`.
Rejected because a callback projector has neither native lifecycle authority nor
durable source identity. Alternative considered: expose a mailbox-specific
reader only. Rejected because it prevents a truthful common source audit and
would make later owner additions invent a parallel API.

### 2. Implement retained history only inside the existing mailbox owner

The existing `orchestration/mailbox` owner will append a bounded immutable
history entry whenever it commits a source lifecycle or terminal transition. It
will associate each journal with one mailbox source instance, maintain a
monotonic source-local sequence and retain only the public identity/lifecycle
facts already allowed by the mailbox contract. The chosen persistent mailbox
backend owns durable journal state and the in-memory backend has explicitly
declared process-local retention/availability semantics; it must not claim
cross-process retention it cannot provide.

The mailbox owner also implements the history reader directly or returns a
narrow reader bound to itself. Planbridge can query only its public reader and
manifest facts; it never stores history or derives cursor positions. A new
consumer process reopens the same retained mailbox source and uses its confirmed
cursor; it does not reopen execution, admit a message or create a parent Run.

Alternative considered: append journal entries in planbridge or an event
observer. Rejected because not every mailbox lifecycle transition passes those
components, and either would create a second lifecycle owner. Alternative
considered: persist final `Record` snapshots alone. Rejected because snapshots
cannot represent source-local transition order, floors or a cursor position.

### 3. Cursor and retention rules are fail-closed

Each mailbox cursor contains a schema/version, source namespace, source
instance identity and after-sequence position. The serialized value remains
opaque and bounded. A malformed encoded value maps to `ErrCursorMalformed`; a
well-formed cursor from another namespace/instance maps to `ErrCursorForeign`.
After scope validation, a position older than retention maps to
`ErrCursorExpired`; a detected discontinuity within a declared retained range
maps to `ErrHistoryGap`. Both return safe retention-floor evidence where known,
never a silently substituted page.

The journal defines its order per mailbox source only. Child/team/workflow
correlation may appear as source facts but cannot cause merging or sorting across
parallel journals. `Complete` means no further currently retained event exists
after the returned page; it is not a promise that the source will never emit a
later lifecycle event unless the returned source terminal evidence establishes
that semantic separately.

Alternative considered: use timestamp and message ID pagination. Rejected
because ties and retries do not establish a durable source order. Alternative
considered: restart from the retention floor on expiry. Rejected because it
hides loss and causes callers to treat an incomplete replay as a complete one.

### 4. Read paths are structurally isolated from execution paths

The history reader receives no handler, action, scheduler or restore dependency.
Its code path uses read-only journal access and detached snapshot conversion;
tests use counting fakes to prove calls never reach publish, claim, ack, nack,
requeue, retry, DLQ, Run, Stream, model, tool, host callback or
`RuntimeRecorder`. The event adapter gains only a projection boundary that
preserves history facts if a caller supplies them; it cannot open a journal or
turn a callback reference into a cursor.

Offline tooling reads JSON fixtures into pure DTOs and verifies capability,
cursor and terminal evidence. It must not construct a mailbox runtime. This
maintains the distinction between validating a transcript and interacting with
the owner represented by that transcript.

Alternative considered: allow `ReadAfter` to reclaim expired leases before
returning a page. Rejected because an observation operation would mutate source
lifecycle and could create retry/DLQ evidence that was absent at call start.

### 5. Manifest support follows source evidence, never the type surface

The manifest reader gains a source-history capability projection or evidence
reference that distinguishes form eligibility from each concrete source's live
availability. `mailbox_async` reports supported durable history/after-cursor
replay only for compatible retained mailbox sources. Direct, subagent, team and
workflow entries retain `unsupported` plus the precise reason until their source
owners add their own journal evidence. A currently unavailable retained mailbox
source returns `ErrSourceUnavailable`, not a direct-run fallback and not
`ErrHistoryUnsupported`.

The contract-test evidence reference points to the history fixture/gate but does
not itself grant read authority. Capability lookup has no side effects and does
not create a history store.

## Risks / Trade-offs

- **A history entry leaks payload or callback error data** → reuse bounded
  mailbox public facts, validate before append, deep-copy permitted values and
  assert redaction/limit negatives in source and replay tests.
- **Retention pruning races a reader** → produce pages from one owner-consistent
  journal snapshot; if the requested position cannot be represented, return the
  applicable expiry/gap error and floor rather than a partial substitute.
- **In-memory mailbox is represented as durable** → require each backend/source
  capability declaration to state durable availability and fail unavailable or
  unsupported before read.
- **A cursor is replayed against another source** → bind namespace and source
  instance identity in the opaque value; test cross-mailbox and cross-form
  foreign cursor failures.
- **Terminal evidence is mistaken for business commit** → retain separate
  lifecycle/terminal facts only and document that host business commit remains
  external.
- **Future sources copy mailbox ordering claims** → make source-kind capability
  audit and manifest tests mandatory; later sources require a new delta and
  source-specific evidence.

## Migration Plan

1. Add source-neutral public history DTOs/error taxonomy and audit all source
   kinds as unsupported by default; no existing execution API changes behavior.
2. Extend mailbox owner/state persistence with bounded journal append and
   source-local retention metadata, preserving existing envelopes and lifecycle
   APIs.
3. Add mailbox reader/cursor validation with no-side-effect tests, then update
   the manifest only for a retained compatible mailbox source.
4. Add reference-only fixtures, restart integration evidence, dual-platform
   gates, maintained docs and the offline example before declaring support.
5. Rollback disables construction/exposure of the reader or marks the mailbox
   source unavailable; it never deletes mailbox lifecycle records, rewrites
   cursors, replays execution, or claims rollback of external effects.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/types/README.md`、`orchestration/mailbox/README.md`、`host/eventadapter/README.md`、`orchestration/planbridge/README.md`、`examples/native-execution-events/README.md`
- 维护 owner：runtime-core、mailbox、host-integration、documentation 与 examples
  maintainers
- 验证证据：OpenSpec strict validation、documentation impact/example impact/
  roadmap consistency gates、history/cursor contract gates、`go test ./...`、
  `go test -race ./...` 与 `golangci-lint run --config .golangci.yml`

## Example Impact Assessment

修改示例

- 影响路径：`examples/native-execution-events/README.md` 与其离线示例代码。
- 理由：示例必须区分实时 callback projection 与 mailbox source journal，展示
  cursor restart replay、typed unsupported/gap/expiry 结果，以及 replay 不触发
  执行或业务写入。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime-core maintainers | module ownership and read-only boundary reviewed; docs consistency gate |
| components | 修改文档 | `core/types/README.md`, `orchestration/mailbox/README.md`, `host/eventadapter/README.md`, `orchestration/planbridge/README.md` | package owners | package history/cursor contracts and unsupported boundaries; package and history gates |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime configuration maintainers | explicit no-new-key/diagnostic limits documented; docs consistency gate |
| contract/API | 修改文档 | `README.md`, `docs/guides/integration.md` | host-integration maintainers | support matrix, API bounds and typed errors; integration and history gates |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md` | observability maintainers | reference-only replay and RuntimeRecorder boundary; replay contract gate |
| examples | 修改文档 | `examples/native-execution-events/README.md` | examples maintainers | offline restart replay markers and rollback; example smoke/history gate |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host-integration maintainers | actual reader call sequence and source audit; integration/history tests |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | confirmed cursor, expiry/gap and no-latest guidance; docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | active change slug matches `openspec list --json`; roadmap consistency gate |
