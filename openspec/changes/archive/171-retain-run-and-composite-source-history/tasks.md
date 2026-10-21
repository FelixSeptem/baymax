## 1. Baseline, source audit and documentation

- [x] 1.1 Mark `retain-run-and-composite-source-history` as the active roadmap candidate and verify the slug matches `openspec list --json` with `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`.
- [x] 1.2 Audit runner, isolated child, team, workflow, and mailbox owners and record each source's identity namespace, sequence scope, retention/floor, cursor scope/version, terminal recovery, storage availability, and unsupported reason; verify the matrix has no inferred cells.
- [x] 1.3 Update every path named in the Public Semantic Documentation Commitment and Example Impact Assessment before implementation completion; verify `pwsh -File scripts/check-openspec-documentation-impact.ps1` and `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` pass.
- [x] 1.4 Document the same-host single-writer/sequential-reopen boundary and explicitly mark multi-writer, cross-machine, distributed-journal, and multi-instance claim behavior unsupported; verify integration docs and manifest use the same wording.

## 2. Source-neutral history contract

- [x] 2.1 Extend the source-neutral history DTO/reader validation for bounded source references, detached pages, opaque cursor scope/version, retention floor, and complete semantics; verify positive, zero, oversized, and caller-mutation cases.
- [x] 2.2 Preserve typed `ErrHistoryUnsupported`, `ErrCursorMalformed`, `ErrCursorForeign`, `ErrCursorExpired`, `ErrHistoryGap`, and `ErrSourceUnavailable` behavior; verify errors are distinguishable without message parsing and happen before source activation.
- [x] 2.3 Define source event identity and sequence validation rules for runner, child, team, workflow, and mailbox declarations; verify local ordering is accepted and synthetic cross-source ordering is rejected.

## 3. Source-owner journal implementations

- [x] 3.1 Add bounded source-journal append facts to each owner that can provide retained history, preserving native run/parent/member/step/attempt/causation/terminal references while excluding payloads, credentials, callback errors, and business-commit assertions; verify size/redaction negatives.
- [x] 3.2 Implement source-local sequence, retention floor/window, and availability reporting for each supported owner; verify MemoryStore/process-local and persisted same-host reopen behavior are not conflated.
- [x] 3.3 Implement source-owned bounded `Capabilities` and `ReadAfter` readers and detached page conversion; verify a newly constructed reader can resume after a confirmed cursor without Run, Stream, retry, resume, or restore.
- [x] 3.4 Add snapshot-consistent retention/pruning reads and overlap conflict detection; verify identical source identities deduplicate and conflicting payloads fail closed.
- [x] 3.5 Keep unsupported or unavailable sources fail-closed; verify no source falls back to latest, retention floor, mailbox history, callback IDs, or host observation sequences.

## 4. Manifest and adapter boundaries

- [x] 4.1 Update the versioned capability manifest with independent history and after-cursor cells, evidence references, retention limits, and unsupported/unavailable reasons for every source/form; verify package/DTO presence cannot upgrade a cell.
- [x] 4.2 Preserve `host/eventadapter` as real-time projection-only and reject callback EventID, TraceID, and observation sequence as journal identity/cursor; verify adapter requests do not create history.
- [x] 4.3 Verify planbridge reads source capability facts without creating a journal, scheduler, queue, or execution; verify history availability does not alter Run/Stream or business-commit semantics.

## 5. Replay, restart and no-side-effect evidence

- [x] 5.1 Add restart integration for every supported source, including crash-before-consumer-projection and retained terminal evidence; verify the same source facts are readable from a later process.
- [x] 5.2 Add negative tests for malformed, foreign, expired, gapped, unsupported, unavailable, boundary-limit, overlap, conflict, and caller-mutation cases; verify errors are typed and fail before effects.
- [x] 5.3 Add counting-fake tests proving history reads never publish, claim, renew, ack, nack, requeue, retry, DLQ, invoke handlers, call model/tool, write diagnostics, or start Run/Stream.
- [x] 5.4 Add source-local ordering tests for independent child/team sources; verify no global parallel order or cursor-as-checkpoint interpretation is exposed.

## 6. Fixtures, gates, examples and docs

- [x] 6.1 Add versioned reference-only history fixtures for capability matrix, cursor restart, terminal recovery, malformed/foreign/expired/gapped/unsupported/unavailable, overlap, conflict, and unsafe latest fallback; verify fixture replay constructs no source owner or runtime action.
- [x] 6.2 Add equivalent shell and PowerShell source-history contract gates and wire them into quality gates; verify both detect identity, order, retention, cursor, and no-side-effect drift.
- [x] 6.3 Update `examples/native-execution-events/README.md` before changing example code with semantic anchors, source ownership, expected markers, unsupported paths, and rollback notes; verify the offline example demonstrates restart replay without a provider or business database.
- [x] 6.4 Synchronize README, integration, best-practices, module-boundary, runtime-config/diagnostics, contract-test-index, package README, and roadmap documentation; verify docs consistency and documentation-impact gates.

## 7. Final verification

- [x] 7.1 Run targeted source-owner, core/types, planbridge, eventadapter, integration, and diagnostics-replay tests; verify source-specific history and unsupported paths.
- [x] 7.2 Run `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml`; record commands, results, risks, rollback, and environmental limitations in `verification.md`.
- [x] 7.3 Run `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, source-history shell/PowerShell gates, `openspec validate retain-run-and-composite-source-history --strict`, roadmap/documentation/example-impact gates, and `git diff --check`; verify all contract commitments before archive.

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

修改示例。`examples/native-execution-events/README.md` 与离线示例展示 source matrix、确认 cursor 后重启读取、typed cursor errors 和 replay 无副作用边界。
