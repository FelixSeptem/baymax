## 1. Owner-Bound Workflow Restore

- [x] 1.1 Add an explicit workflow continuation resume request/owner binding and validate required checkpoint, graph, authority, and deployment facts before activation; verify focused validation tests reject missing and foreign bindings.
- [x] 1.2 Route workflow Run and Stream resume through the configured workflow source owner/store without calling Run/Stream; verify fresh-engine restore, duplicate request projection, conflict rejection, and Run/Stream terminal parity.
- [x] 1.3 Add bridge-level integration tests proving workflow resume never dispatches completed steps again and preserves native checkpoint/terminal projection.

## 2. Capability Evidence and Unsupported Boundaries

- [x] 2.1 Audit manifest generation against configured source owner/store evidence per form and pause reason; verify absent, mismatched, memory-only, and unsupported deployment evidence remains unsupported/unavailable.
- [x] 2.2 Add regression tests that mailbox handler continuation and resumed planbridge execution fail before side effects, and that child/team evidence cannot promote workflow or sibling-form capability.
- [x] 2.3 Update planbridge capability contract documentation and fixtures to match tested owner-bound restore paths; verify fixture/gate assertions reflect the same support matrix.

## 3. Public Contract and Gates

- [x] 3.1 Update the planbridge README, integration guide, host execution plan example README, roadmap, and root README with the implemented binding/support boundary; verify every declared path is changed in the branch diff.
- [x] 3.2 Run focused planbridge/workflow tests, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `openspec validate --all`; record any environment-blocked command without marking the task complete. Full quality gate completed successfully with `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=7200`; all 88 steps passed. Focused tests, documentation checks, and OpenSpec validation also passed.

## Documentation Impact Assessment

| Area | Outcome | Maintenance docs | Owner | Verification evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | integration docs | documented owner-first restore boundary |
| components | 修改文档 | `orchestration/planbridge/README.md` | planbridge | public binding documentation |
| configuration | 修改文档 | `orchestration/planbridge/README.md` | planbridge | owner/store binding configuration |
| contract/API | 修改文档 | `orchestration/planbridge/README.md` | planbridge | focused bridge tests |
| diagnostics | 修改文档 | `docs/guides/integration.md` | integration docs | native outcome/error boundary |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples owner | workflow restore guidance |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration docs | integration guide gate |
| best practices | 无需文档变更（附理由） | — | runtime docs | no new operational advice |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime architecture | roadmap status gate |

## Example Impact Assessment

- 修改示例

更新 host execution plan README 的 workflow continuation owner/store 配置与 unsupported 边界说明，不新增在线依赖。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`orchestration/planbridge/README.md`、`docs/guides/integration.md`、`examples/host-execution-plan/README.md`、`docs/development-roadmap.md`、`README.md`
- 维护 owner：planbridge、integration docs、examples owner、runtime architecture
- 验证证据：`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`、`go test ./tool/contributioncheck -run TestReleaseStatusParityDocsConsistency -count=1`
