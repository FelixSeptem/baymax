## 1. Public Contract and Compatibility

- [x] 1.1 Add durable checkpoint/ref/persisted continuation/claim DTOs and typed errors without breaking existing metadata-oriented `RunCheckpointStore` implementers; verify `go test ./core/types` and API compatibility tests.
- [x] 1.2 Define direct pause-reason, source binding, digest, expiry, resource-bound, and capability evidence validation; verify positive, negative, and exact-boundary type tests.
- [x] 1.3 Add capability-manifest cells and stable unsupported reasons for direct/child/team/workflow/mailbox and all three pause reasons; verify manifest contract fixtures and unsupported cells.
- [x] 1.4 Document that planbridge resumed plans remain unsupported despite direct restore; verify pre-activation rejection contract tests.

## 2. Durable Store and Direct Restore

- [x] 2.1 Implement a bounded file-backed durable continuation store with atomic save/load/claim semantics and process-local memory test store; verify independent store handles can reopen records and serialize conflicting claims.
- [x] 2.2 Persist canonical checkpoint identity/digest, source position or continuation reference, run/session/parent/host correlation, pause reason, authority/plan/scope/effect namespace, expiry, predecessor, and attempt state; verify tamper, foreign, expired, oversized, and unavailable tests.
- [x] 2.3 Add Engine injection and new-Engine `ResumeFromCheckpoint` for direct runs only; verify same logical run/session and no provider/tool reexecution before validated claim.
- [x] 2.4 Preserve host verifier ordering outside source locks with generation/state recheck before claim; verify denial leaves checkpoint/key untouched and concurrent callbacks commit at most one resume.
- [x] 2.5 Implement source-owned `approval_required` and `budget_exhausted` pause production/restoration alongside `input_required`; verify approval blocks pending tool dispatch and budget never resets/replenishes on restore.
- [x] 2.6 Define cancellation/timeout/retry/resume precedence and crash-after-claim/start/terminal failure projection; verify source evidence, in-progress/recovery-required, and no automatic retry/new Run.
- [x] 2.7 Verify exact duplicate projection, conflicting duplicate rejection, successor checkpoint isolation, and Run/Stream parity across fresh Engine instances with integration tests.

## 3. Composite Unsupported Boundaries and Replay Evidence

- [x] 3.1 Keep child/fanout, team, workflow, mailbox, and planbridge resumed cells unsupported until each source owner has complete graph restore evidence; verify rejection occurs before enqueue, publish, Run, Stream, member, or step activation.
- [x] 3.2 Add versioned offline checkpoint lifecycle fixtures covering save/load/claim/start/successor/terminal and every crash, corruption, expiry, foreign, conflict, and store-unavailable window; verify replay is deterministic and read-only.
- [x] 3.3 Add fixture drift classifications for invented terminal, missing source binding, duplicate execution, unsupported composite restore, and Run/Stream mismatch; verify shell and PowerShell contract gates agree.

## 4. Documentation-First Example and Public Docs

- [x] 4.1 Complete the offline example documentation baseline with semantic anchor, runtime path, expected markers, and rollback notes before adding example code; verify links and documentation gate.
- [x] 4.2 Implement the example only after the direct store API is delivered and verify reopen/restore, duplicate claim, store failure, and composite rejection markers without live provider dependencies.
- [x] 4.3 Update README, roadmap, integration/best-practices guides, module/config/diagnostic docs, component READMEs, and contract index; verify every declared documentation path changed and governance declarations match.

## 5. Verification and Capability Release

- [x] 5.1 Run focused core/types, core/runner, host and planbridge tests including race coverage; record commands and results in change verification evidence.
- [x] 5.2 Run fixture/replay, shell and PowerShell contract gates for persistence, restore, manifest and documentation; verify no gate treats replay as execution recovery.
- [x] 5.3 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `scripts/check-quality-gate.ps1`, and `scripts/check-docs-consistency.ps1`; record any environment-limited command and reason.
- [x] 5.4 Advertise direct restore cells only after all three direct pause reasons pass cross-process acceptance; verify manifest fixtures reject all unverified cells and preserve typed unavailable/unsupported distinctions.

## Example Impact Assessment

新增示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/runner/README.md`、`core/types/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：runner/types maintainers、host integration maintainers、documentation and examples maintainers
- 验证证据：`openspec validate persist-native-continuations-and-restore-composite-graphs --strict`、OpenSpec documentation/example/roadmap gates、`scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime architecture maintainers | source store and restore ownership boundaries documented |
| components | 修改文档 | `core/runner/README.md`, `core/types/README.md` | runner/types maintainers | public DTO and Engine responsibility documented |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime diagnostics maintainers | typed checkpoint failure and diagnostics documented |
| contract/API | 修改文档 | `docs/guides/integration.md` | host integration maintainers | direct restore and unsupported scope documented |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime diagnostics maintainers | checkpoint facts use canonical writer |
| examples | 新增文档 | `examples/durable-continuation-restore/README.md` | examples maintainers | offline scenario baseline documented |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host integration maintainers | admission and failure behavior documented |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | restore versus retry/replay guidance documented |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | active Spec 3 scope documented |
