## Why

The current source history journal and callback projection can report an append failure, but they do not persist a source commit identity or a durable projection position that explains what a consumer must recover after a crash between source append and projection commit. `AppendHealth` is process-local and therefore cannot prove terminal recovery, source/projection atomicity, or a bounded recovery-required state. This change is needed before composite continuation work can safely depend on source facts rather than replaying or rerunning uncertain effects.

## What Changes

- Add a source-owner commit and projection-recovery contract that distinguishes committed source facts, uncommitted/unknown transitions, and consumer projection progress.
- Persist a bounded source commit record containing source identity, source sequence/event identity, terminal evidence, projection cursor, schema/version, digest, and recovery status.
- Provide an owner-scoped recovery/read API that can re-project committed source facts after a crash without invoking Run, Stream, retry, resume, tools, models, acknowledgements, or business effects.
- Make repeated projection of the same source commit idempotent and make conflicting identity/payload or cursor advancement fail closed with typed errors.
- Preserve existing explicit owner journal injection and same-host single-writer boundaries; capability declarations must expose whether the commit/projection transaction is available.
- Keep append health as an observable degraded/unavailable signal, but do not treat it as proof that an effect was prevented or that the external consumer projection committed atomically.
- Add crash-window, restart, overlap, conflict, unavailable-store, terminal-recovery, and Run/Stream parity evidence plus dedicated contract gates.

## Capabilities

### New Capabilities

- `source-history-projection-transaction`: defines source commit records, projection cursor advancement, recovery-required classification, idempotent re-projection, and fail-closed conflict semantics.

### Modified Capabilities

- `native-execution-event-history`: retained source history must expose its commit/projection transaction scope and distinguish committed source facts from unavailable or recovery-required windows.
- `host-runtime-capability-manifest`: source capability entries must report whether durable source commit/projection recovery is available, bounded by the declared deployment scope.

## Impact

Affected areas include `core/history`, `core/types`, runner/child/team/workflow history emitters, source-owner capability reporting, contract fixtures and gates, and the integration/runtime documentation. The change does not enable team or mailbox continuation, does not add a second journal or queue, and does not provide multi-writer or cross-machine coordination. Existing unsupported forms remain typed `unsupported_capability` or `recovery-required`.

## Example Impact Assessment

修改示例

理由：`examples/native-execution-events/README.md` 增加显式 owner ledger 注入、source commit evidence 与 no-rerun recovery boundary 的语义锚点；不改变示例的执行路径、配置键或 expected runtime markers。

## Documentation Impact Assessment

| Area | Outcome | Document | Owner | Evidence |
|---|---|---|---|---|
| architecture | 修改文档 | `docs/runtime-module-boundaries.md`, `core/history/README.md` | Runtime contracts owner | source owner remains the only fact writer; projection recovery is bounded and read-only |
| components | 修改文档 | `core/types/README.md`, `docs/guides/integration.md` | Runtime contracts owner | commit/projection scope, typed recovery-required and same-host ownership are documented |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | Runtime contracts owner | no configuration key is added; the deployment/recovery boundary is documented |
| contract/API | 修改文档 | `openspec/changes/make-source-history-projection-transactional/specs/native-execution-event-history/spec.md`, `openspec/changes/make-source-history-projection-transactional/specs/host-runtime-capability-manifest/spec.md` | Runtime contracts owner | delta requirements and capability fields are validated by contract tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md` | Runtime contracts owner | no new recorder field; degraded/recovery outcome and gate evidence remain additive and typed |
| examples | 修改文档 | `examples/native-execution-events/README.md` | Runtime contracts owner | example documents explicit owner ledger injection, source commit evidence and the no-rerun recovery boundary |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | Runtime contracts owner | explicit owner injection and read-only recovery integration are documented |
| best practices | 无需文档变更（附理由） | — | Runtime contracts owner | this change does not alter normal application construction or operational best practices |
| roadmap | 修改文档 | `README.md`, `docs/development-roadmap.md` | Runtime roadmap owner | active candidate, evidence boundary and successor dependencies stay explicit |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/guides/integration.md`, `docs/mainline-contract-test-index.md`, `core/history/README.md`, `core/types/README.md`, `examples/native-execution-events/README.md`, `openspec/changes/make-source-history-projection-transactional/specs/native-execution-event-history/spec.md`, `openspec/changes/make-source-history-projection-transactional/specs/host-runtime-capability-manifest/spec.md`
- 维护 owner：Runtime contracts owner and source-owner maintainers
- 验证证据：`openspec validate --all`; `pwsh -File scripts/check-openspec-documentation-impact.ps1`; `pwsh -File scripts/check-docs-consistency.ps1`; focused source commit/replay tests and dedicated contract gates
