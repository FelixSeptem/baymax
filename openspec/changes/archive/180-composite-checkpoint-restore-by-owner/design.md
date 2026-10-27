## Context

See proposal.md for the motivation. Owner restore implementations exist for child, team, and workflow, but planbridge's bindings and dispatch paths are asymmetric. Workflow continuation restore is implemented by `workflow.Engine.ResumeContinuation`; the bridge currently constructs an engine for Run/Stream but has no resume request binding. Capability manifests are detached contracts and must remain tied to the configured owner/store evidence and deployment scope.

## Goals / Non-Goals

**Goals:**

- Expose explicit owner-bound workflow continuation restore through planbridge.
- Keep bridge admission form-specific and fail closed when evidence or owner configuration is missing.
- Keep Run and Stream on the same owner restore path and preserve the native source result.
- Correct public capability guidance to distinguish owner APIs from bridge support.

**Non-Goals:**

- Implement new checkpoint formats or source stores for child/team/workflow.
- Add mailbox handler execution safe points, resumed planbridge graph restore, or a second mailbox/scheduler owner.
- Expand support to multi-writer, cross-machine, or multi-instance deployments without store evidence.
- Claim every pause reason is supported merely because an owner has a resume method.

## Decisions

1. Add a workflow resume request to `WorkflowBinding` and route it to the existing workflow continuation owner/store. Reuse its `ResumeContinuation` validation, atomic claim, idempotency, and terminal projection rather than introducing bridge state.
2. Preserve the public Run/Stream distinction only for event delivery. Both resume paths call the same source-owner restore operation, passing the optional stream callback for native events.
3. Keep capability decisions manifest-driven. Bridge configuration must provide the owner and compatible manifest/store scope; absent evidence stays unsupported or unavailable. Owner API existence is not sufficient evidence for each pause reason.
4. Keep mailbox handler continuation and resumed-plan trigger unsupported until those forms have execution safe points and owner-native graph/message restore evidence.

## Risks / Trade-offs

- [A workflow owner may be configured with a checkpoint store but not the matching continuation store] -> Require explicit continuation owner/store wiring and reject missing restore configuration before dispatch.
- [Manifest evidence can drift from runtime binding] -> Validate manifest compatibility and required capability cells against the actual binding before claim; contract tests cover mismatch cases.
- [Run/Stream callbacks can diverge] -> Share one restore dispatch branch and test equivalent requests against both APIs.

## Migration Plan

Additive API change: existing non-resumed workflow plan bindings continue to Run/Stream unchanged. Hosts opting into restore provide the workflow resume request and source owner store. Rollback consists of removing the resume binding and using the native workflow API explicitly; existing checkpoints and completed effects remain untouched.

## Documentation Impact Assessment

| Area | Outcome | Maintenance docs | Owner | Verification evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | integration docs | owner delegation and deployment scope guidance |
| components | 修改文档 | `orchestration/planbridge/README.md` | planbridge | implementation boundary text |
| configuration | 修改文档 | `orchestration/planbridge/README.md` | planbridge | owner/store binding configuration |
| contract/API | 修改文档 | `orchestration/planbridge/README.md` | planbridge | workflow binding contract tests |
| diagnostics | 修改文档 | `docs/guides/integration.md` | integration docs | native error/outcome boundary |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples owner | workflow restore opt-in guidance |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration docs | integration guide verification |
| best practices | 无需文档变更（附理由） | — | runtime docs | no change to recovery guidance |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime architecture | roadmap consistency gate |

## Example Impact Assessment

- 修改示例

host execution plan README 增加 workflow owner restore 的配置与拒绝边界说明，保持离线、无 provider 依赖。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`orchestration/planbridge/README.md`、`docs/guides/integration.md`、`examples/host-execution-plan/README.md`、`docs/development-roadmap.md`、`README.md`
- 维护 owner：planbridge、integration docs、examples owner、runtime architecture
- 验证证据：`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`、`go test ./tool/contributioncheck -run TestReleaseStatusParityDocsConsistency -count=1`
