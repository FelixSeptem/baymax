## 1. Public Contract and Evidence Model

- [x] 1.1 Add child checkpoint source reference/cursor binding and validation while preserving bounded payload validation; verify positive, malformed, foreign, and cursor mismatch unit tests (`go test ./orchestration/composer -run TestChildContinuationBindsChildHistorySourceAndCursor -count=1`).
- [x] 1.2 Add source-local child capability evidence for fresh-process reopen and per-reason pause cells; verify File owner + history promotes all three while Memory/owner-less wiring remains unsupported (`go test ./orchestration/composer -run TestComposerContinuationCapabilitiesRequireFileOwnerAndHistory -count=1`).
- [x] 1.3 Add child-specific typed expired/gap handling by reusing native history contracts; verify expired/gap errors propagate before claim or execution (`go test ./orchestration/composer -run 'TestComposerResumeChildContinuationRejects(ExpiredSourcePosition|SourceGap)BeforeClaim' -count=1`).

## 2. Fresh-Process Restore and Safety

- [x] 2.1 Add FileChildContinuationStore close/reopen tests with new queue/history/Composer owners for all three pause reasons; verify original parent/child/task/attempt identity is retained (`go test ./orchestration/composer -run TestComposerRestoresChildContinuationWithFreshOwnersForEachPauseReason -count=1`).
- [x] 2.2 Add pending-boundary restore tests proving completed tool/effect refs are never re-executed and source replay remains read-only (`go test ./orchestration/composer -run TestComposerRestoresChildContinuationWithFreshOwnersForEachPauseReason -count=1`).
- [x] 2.3 Add retention-floor, expired cursor, gap, unavailable source, duplicate/conflict claim, and crash-window fail-closed tests; verify the complete composer continuation suite covers these paths (`go test ./orchestration/composer -run 'Test(ComposerResumeChildContinuationRejects|ComposerResumeChildContinuationUsesExistingTaskAndDeduplicates|ComposerResumeChildContinuationFailsClosedWithoutExecutor|ComposerRestoresChildContinuation|PersistChildPauseBindsCurrentNativeHistoryCursor)' -count=1`).

## 3. Capability, Example, Documentation, and Gates

- [x] 3.1 Update child capability manifest promotion to require complete child-specific evidence and preserve deployment scope; verify incomplete cells reject without direct fallback (`go test ./orchestration/planbridge ./orchestration/composer -run 'TestChildContinuationManifestPromotesOnlyChildForm|TestContinuationManifestFailsClosedForIncompleteOrUnavailableSource|TestComposerContinuationCapabilitiesRequireFileOwnerAndHistory' -count=1`).
- [x] 3.2 Update Composer and planbridge public docs and durable continuation example to demonstrate fresh-process child restore, cursor limits, and non-replay; verify package/example tests and expected markers (`go test ./orchestration/composer ./orchestration/planbridge ./examples/durable-continuation-restore -count=1`).
- [x] 3.3 Extend the native continuation contract gate with child fresh-owner/cursor evidence and register the Composer package in its test set; verify the gate passes and the negative replay fixture still fails (`pwsh -File scripts/check-native-continuation-contract.ps1`).
- [x] 3.4 Update roadmap and evidence indexes with exact supported profile and remaining limitations; run roadmap/spec/documentation consistency checks (`pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `openspec validate --all`).

## 4. Verification and Closure

- [x] 4.1 Run affected package tests, example tests, race tests, contract gates, `openspec validate --all`, quality gate, and docs consistency; record any environment-dependent skip.
- [x] 4.2 Confirm every public semantic documentation path above exists in the feature diff and proposal/design/tasks declarations match; archive only after all verification passes.

## Example Impact Assessment

修改示例

## Documentation Impact Assessment

| Area | Outcome | Document | Owner | Evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | — | 不引入新的跨模块架构边界 |
| components | 修改文档 | `orchestration/composer/README.md` | Composer continuation owner | source cursor binding 与恢复前检查 |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config documentation owner | 明确本提案不新增配置键或默认值 |
| contract/API | 修改文档 | `orchestration/planbridge/README.md` | capability manifest owner | pause cell 能力提升条件 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config documentation owner | 明确 continuation 不改变诊断字段 |
| examples | 修改文档 | `examples/durable-continuation-restore/README.md` | continuation example owner | child reopen 与 cursor fail-closed markers |
| CLI/integration | 无需文档变更（附理由） | — | — | 不新增 CLI 或集成入口 |
| best practices | 无需文档变更（附理由） | — | — | 无独立实践语义变化 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | F active 与部署边界 |
| public overview | 修改文档 | `README.md` | repository overview owner | 展示当前 active change 与保守能力状态 |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`orchestration/composer/README.md`, `orchestration/planbridge/README.md`, `examples/durable-continuation-restore/README.md`, `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `README.md`
- 维护 owner：Composer continuation owner; capability manifest owner; continuation example owner; roadmap owner; runtime config documentation owner; repository overview owner
- 验证证据：`go test ./orchestration/composer`; `go test ./orchestration/planbridge`; `go test ./examples/durable-continuation-restore`; `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`; `pwsh -File scripts/check-docs-consistency.ps1`
