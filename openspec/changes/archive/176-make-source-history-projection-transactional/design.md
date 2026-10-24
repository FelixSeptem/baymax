## Context

See `proposal.md` for the motivation. Today `history.Journal.Append` persists a
bounded event snapshot and owners expose only process-local `AppendHealth`.
That is sufficient to make a failed callback append visible, but it cannot
describe the durable recovery state after an interruption between a source
transition, journal append, and external consumer projection. The existing
history reader must remain side-effect free and existing continuation owners
must retain their own checkpoint authority.

## Goals / Non-Goals

**Goals:**
- Persist a bounded, owner-scoped source commit ledger containing stable source
  facts and a recovery classification.
- Bind projection acknowledgements/cursors to source commit identity and make
  duplicate application deterministic and conflicting evidence fail closed.
- Keep recovery reads reference-only and usable after sequential process reopen.
- Surface the recovery contract in source capability/manifest evidence without
  upgrading forms that have no owner store.

**Non-Goals:**
- Atomic commits to a Danny/application business database.
- A second journal, queue, scheduler, execution loop, or automatic retry.
- Team/mailbox continuation, resumed planbridge execution, multi-writer CAS, or
  cross-machine source replication.
- Replacing owner-specific checkpoint stores or using history as a checkpoint.

## Decisions

1. **Introduce a source-owned projection ledger next to `history.Journal`.**
   A ledger record is keyed by the immutable `NativeHistoryIdentity.EventID`
   and includes a canonical digest of the payload-omitted source fact, terminal
   evidence, and a state of committed or recovery-required. The store is an
   explicit option owned by the source; no engine creates an implicit durable
   store. This separates history fact persistence from external projection
   execution while giving a restartable owner ledger.

2. **Use one owner operation to append a history fact and record its commit
   evidence.** A `Commit` operation first normalizes/persists the source fact
   through the owner journal, then persists its ledger record under the same
   same-host writer boundary. If ledger persistence fails after journal append,
   the owner records a durable `recovery_required` marker where possible and
   returns typed unavailability. It never reports a successful projection
   transaction or reruns the effect.

3. **Model consumer projection acknowledgement as idempotent source evidence.**
   `AcknowledgeProjection` accepts source identity, event ID/digest and a
   bounded projection reference. Identical acknowledgement returns the original
   result; a changed digest/ref or a non-monotonic cursor returns conflict. The
   API stores only opaque bounded references, never business payloads or host
   credentials. External consumers remain responsible for their own atomic
   writes/outbox use.

4. **Recovery is a bounded read.** `ReadCommittedAfter` returns committed
   records and a source cursor; it cannot call execution owners or mutate the
   source. A recovery-required record is observable separately and blocks
   cursor acknowledgement past the uncertain fact. This preserves source-local
   order and avoids fabricating a global child/team order.

5. **Manifest evidence is opt-in and source-specific.** Extend
   `HistoryCapabilities` and the planbridge manifest with a recovery field only
   for a reader/store that explicitly reports it. Existing journals continue to
   advertise retained history only, not transactional projection recovery.

## Risks / Trade-offs

- [Two local files cannot atomically cover a consumer database] → document the
  boundary, retain idempotent source evidence, and require consumer-side dedup.
- [Crash after journal append but before ledger write] → return/retain
  recovery-required; do not infer execution outcome or advance projection.
- [Ledger retention can make old acknowledgement unavailable] → bind the
  retention floor to source cursor and return explicit expiry/gap errors.
- [Existing callback emitters are asynchronous] → keep their current health
  behavior; add the ledger only through explicit source-owner transaction APIs.

## Migration Plan

1. Add types, in-memory and file-backed ledger, and negative crash-window tests.
2. Integrate the ledger with explicit history owners and source capability
   projection; preserve unsupported results when the ledger is absent.
3. Add fixtures and dedicated shell/PowerShell gates, then update public docs
   and roadmap status.
4. Roll back by disabling the explicit ledger option/capability. Existing
   journals continue as retained-history-only and no source reruns work.

## Example Impact Assessment

修改示例

理由：`examples/native-execution-events/README.md` 增加 source-owned ledger/recovery 的语义锚点；不增加用户可运行的常规 form、配置键或 example marker，重启、冲突和 crash-window fixtures 仍是主验证方式。

## Documentation Impact Assessment

| Area | Outcome | Document | Owner | Evidence |
|---|---|---|---|---|
| architecture | 修改文档 | `docs/runtime-module-boundaries.md`, `core/history/README.md` | Runtime contracts owner | source owner/consumer boundary and no second journal |
| components | 修改文档 | `core/types/README.md`, `docs/guides/integration.md` | Runtime contracts owner | commit, acknowledgement, recovery-required and local deployment scope |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | Runtime contracts owner | no configuration key is added; the deployment/recovery boundary is documented |
| contract/API | 修改文档 | `openspec/changes/make-source-history-projection-transactional/specs/native-execution-event-history/spec.md`, `openspec/changes/make-source-history-projection-transactional/specs/host-runtime-capability-manifest/spec.md` | Runtime contracts owner | source commit recovery manifests and contract tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md` | Runtime contracts owner | no RuntimeRecorder schema addition; gate evidence remains explicit |
| examples | 修改文档 | `examples/native-execution-events/README.md` | Runtime contracts owner | example documents explicit owner ledger injection and the no-rerun recovery boundary |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | Runtime contracts owner | explicit owner injection and read-only recovery integration are documented |
| best practices | 无需文档变更（附理由） | — | Runtime contracts owner | this change does not alter normal application construction or operational best practices |
| roadmap | 修改文档 | `README.md`, `docs/development-roadmap.md` | Runtime roadmap owner | active candidate and successor dependencies |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/guides/integration.md`, `docs/mainline-contract-test-index.md`, `core/history/README.md`, `core/types/README.md`, `examples/native-execution-events/README.md`, `openspec/changes/make-source-history-projection-transactional/specs/native-execution-event-history/spec.md`, `openspec/changes/make-source-history-projection-transactional/specs/host-runtime-capability-manifest/spec.md`
- 维护 owner：Runtime contracts owner and source-owner maintainers
- 验证证据：`openspec validate --all`; `pwsh -File scripts/check-openspec-documentation-impact.ps1`; `pwsh -File scripts/check-docs-consistency.ps1`; focused source commit/replay tests and dedicated contract gates
