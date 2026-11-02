## Context

See proposal.md. Native event observers are synchronous projections; native
source journals own retained execution facts. An external adapter needs its
own explicit delivery checkpoint without weakening either owner.

## Goals / Non-Goals

Deliver polling and callback offers through one bounded sequential confirmation
path. No source execution control, implicit worker, global store, diagnostics
writer, provider dependency, configuration key or business-effect ledger.

## Decisions

1. Add `host/sourceadapter`, depending only on the standard library. Use a
   local SourceIdentity with bounded kind/namespace/ID rather than widening
   native history's closed source-kind vocabulary. Metadata binds identity
   namespace and cursor version; every cursor carries the same binding.
2. Source.Poll returns one bounded Page; Adapter.RunOnce polls and Adapter.Offer
   admits explicit callback pages through the same delivery code. Source,
   consumer, store and metadata are constructor-injected. No automatic loop.
   Default operation timeout is five seconds; cooperative dependencies must
   honor context. Negative timeout and invalid page limits fail admission.
3. Facts have bounded payload (64 KiB), external ID/reference (256 bytes), a
   sha256 payload digest and next cursor (4096 bytes). Validate and detach a
   whole page before any effect. A canonical receipt fingerprint binds all
   fact fields, not only the payload digest. Bound pages/receipt windows to 200.
4. State contains immutable metadata, revision, confirmed cursor, one pending
   receipt and up to 200 recent confirmed receipts. Save pending before consume
   without advancing cursor; this detects changed facts after crashes/retry.
   Confirm sent/gone with a second state CAS, clear pending, and preserve prior
   confirmed prefix on all other outcomes. Host effect CAS remains required
   for crash-after-effect and identity evidence older than the bounded window.
   No unbounded dedup map or business outcome storage.
5. Memory and file stores expose Load/CompareAndSwap with detached snapshots.
   Memory supports concurrent calls within the one injected store instance.
   File storage is same-host single-writer sequential reopen, uses restrictive
   temp-file plus rename and bounded strict JSON decoding. It does not claim
   multi-instance or distributed CAS. Save failures must not mutate in-memory
   state; uncertain external store commits require reload/reconciliation.
6. Adapter operations use a fail-fast serial admission flag and short mutexes
   only for health/shutdown; no lock is held across source/consumer calls.
   Dispose/Abort cancel a lifecycle context and are idempotent. A store commit
   already admitted can finish during shutdown and must be reloaded; never
   report cancellation as proof of rollback. Health uses finite classes,
   saturating counters and bounded references; raw dependency errors stay out.
7. Offline replay uses pure admission/transition validation and reference-only
   fixtures, no executable source/consumer dependencies. Dedicated shell and
   PowerShell gates exercise positive and intentional drift cases.

Alternatives: extending Observer would change callback semantics; using a
history SourceRef would mislabel external sources; a background loop adds hidden
scheduling; committing only cursor loses pending conflict evidence after crash;
an unbounded receipt ledger duplicates host ownership.

## Risks / Trade-offs

- Uncooperative callbacks cannot be killed -> require cooperative context,
  expose the limit, and never detach effect goroutines.
- Effect succeeds before state commit -> keep pending receipt and prior cursor;
  retry only with host CAS and immutable identity.
- Limited receipt evidence -> publish 200-entry window, never claim global
  dedup or catch arbitrary historical identity reuse.
- Multiple file writers -> unsupported; host supplies a coordinated state store
  before using a broader deployment scope.
- Source/consumer supplies false facts -> host retains source trust and
  permanent-gone authority; hashes are integrity references, not authorization.

## Migration Plan

Additive opt-in integration. Start with explicit empty source-bound store and
host scheduling. Rollback stops polling/offers and preserves state and host
effect records for reconciliation; never erase pending evidence to retry.

## Example Impact Assessment

新增示例

新增离线 `examples/source-adapter-lifecycle`，展示 sent/gone/retryable、
cursor-after-commit、abort 和 restart；不连接真实外部 source。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/source-adapter-lifecycle.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`examples/source-adapter-lifecycle/README.md`
- 维护 owner：`host/sourceadapter` maintainers
- 验证证据：`pwsh -File scripts/check-source-adapter-lifecycle-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | boundary tests and dedicated gate |
| components | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | source-adapter lifecycle suite |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | host integration maintainers | constructor bounds and timeout; no env/file/hot-update keys |
| contract/API | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | bounds and restart tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | host integration maintainers | finite API health; no RuntimeRecorder write or QueryRuns field |
| examples | 新增文档 | `examples/source-adapter-lifecycle/README.md` | examples maintainers | offline example marker suite |
| CLI/integration | 修改文档 | `README.md` | documentation maintainers | library-first integration entry |
| best practices | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | cursor-after-commit and CAS notes |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status consistency |
