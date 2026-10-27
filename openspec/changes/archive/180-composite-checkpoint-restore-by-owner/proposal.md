## Why

Composite owners already expose source-owned restore operations, but `host_execution_plan.v1` does not consistently route plans through them: child and team have explicit resume bindings while workflow has no resume request/owner path. This leaves a public bridge unable to continue an evidenced workflow checkpoint and makes its capability/documentation boundary easy to misread. The change closes only the bridge gap backed by owner evidence; it does not claim mailbox handler continuation or resumed-plan support.

## What Changes

- Add an explicit workflow continuation resume binding to planbridge and delegate restore to the configured workflow owner/store, preserving native result, checkpoint lineage, and Run/Stream event behavior.
- Verify continuation admission against form-specific source-owner capability evidence and deployment scope; never infer support from a package, DTO, or another form.
- Align child/team/workflow/mailbox/resumed-plan capability and public documentation with actual owner restore paths, retaining typed unsupported/unavailable behavior where evidence is absent.
- Add bridge-level tests for successful fresh-owner workflow restore, duplicate/conflicting resume, missing/incompatible owner, deployment scope, and unsupported mailbox/resumed-plan paths.

## Capabilities

### New Capabilities

None. This change extends the existing composite continuation contract.

### Modified Capabilities

- `composite-continuation-restore`: require planbridge restore to delegate to the matching source owner and require manifest support to be evidence- and deployment-scoped per form.
- `host-execution-plan-bridge`: define explicit owner-bound workflow resume behavior and fail-closed unsupported boundaries.

## Impact

Affected code: `orchestration/planbridge/{types.go,compile.go,execute.go,capability.go}` and focused contract/integration tests. Public documentation: `orchestration/planbridge/README.md` and `docs/development-roadmap.md`. No new dependency, queue, scheduler, mailbox owner, or host-side continuation state machine is introduced.

## Documentation Impact Assessment

| Area | Outcome | Maintenance docs | Owner | Verification evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | integration docs | workflow owner-bound restore and unsupported boundary paragraphs |
| components | 修改文档 | `orchestration/planbridge/README.md` | planbridge | binding and delegation contract section |
| configuration | 修改文档 | `orchestration/planbridge/README.md` | planbridge | owner/store binding configuration is documented |
| contract/API | 修改文档 | `orchestration/planbridge/README.md` | planbridge | WorkflowBinding owner/store/resume API tests |
| diagnostics | 修改文档 | `docs/guides/integration.md` | integration docs | native outcome and error boundary is documented |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples owner | workflow restore opt-in and unsupported fallback are documented |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration docs | host integration usage and fail-closed behavior |
| best practices | 无需文档变更（附理由） | — | runtime docs | existing owner-first recovery guidance remains valid |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime architecture | active OpenSpec status gate |

## Example Impact Assessment

- 修改示例

更新 host execution plan 离线 README，明确 workflow continuation 必须提供匹配的 owner/store/manifest binding，并保留 mailbox handler continuation 与 resumed-trigger plan 的 unsupported 边界；不引入 live provider 依赖。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`orchestration/planbridge/README.md`、`docs/guides/integration.md`、`examples/host-execution-plan/README.md`、`docs/development-roadmap.md`、`README.md`
- 维护 owner：planbridge、integration docs、examples owner、runtime architecture
- 验证证据：`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`、`go test ./tool/contributioncheck -run TestReleaseStatusParityDocsConsistency -count=1`
