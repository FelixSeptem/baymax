# Implementation verification

日期：2026-10-14。change：`establish-host-execution-plan-bridge-contract`；branch：`feat/host-execution-plan-bridge`。

## Scope and status

本 change 已实现 opt-in `orchestration/planbridge`、原生 validator 接缝、real-owner integration、reference-only replay、双平台 contract gate、离线 fake model/tool example，以及对应的公开维护文档。Run/Stream 继续由单一原生 owner 执行；bridge 不新增 terminal、budget、retry、recovery 或 RuntimeRecorder 写入入口。

对外语义发生变化，维护文档已在同一功能分支同步：`README.md`、`docs/development-roadmap.md`、`docs/components/orchestration.md`、`orchestration/README.md`、`orchestration/planbridge/README.md`、`docs/runtime-module-boundaries.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`examples/host-execution-plan/README.md`。proposal/design/tasks/spec 中均保留 Documentation Impact Assessment、Public Semantic Documentation Commitment 和 Example Impact Assessment。

## Executed checks

| Command / check | Exit | Evidence |
| --- | ---: | --- |
| `go test ./orchestration/planbridge ./tool/diagnosticsreplay -count=1` | 0 | bounded snapshot、limits、canonical golden、offline drift replay |
| `go test ./integration -run TestHostExecutionPlan -count=1` | 0 | real runner/team/workflow/child Run/Stream、pause、observer 与 isolation |
| `go test ./orchestration/... ./integration/... ./tool/diagnosticsreplay -count=1` | 0 | 受影响 orchestration、integration、replay 包回归 |
| `go test ./tool/contributioncheck -count=1` | 0 | 治理与 host-plan gate fixtures；临时 mutation 已恢复 |
| `go test ./... -count=1 -timeout 15m` | 0 | 全包普通测试 |
| `go test -race ./... -count=1 -timeout 20m -p 1` | 0 | 全包 race 测试 |
| `golangci-lint run --config .golangci.yml` | 0 | 0 issues |
| `pwsh -NoProfile -File scripts/check-host-execution-plan-contract.ps1` | 0 | host-plan unit/integration/replay/contributioncheck 与 invalid fixture 负例 |
| `go run ./examples/host-execution-plan` | 0 | `plan-compiled`、`source-owner-preserved`、`run-stream-parity`、`invalid-plan-rejected`、`unsupported-capability-rejected`、`no-side-effects-before-rejection`、`host-commit-required` |
| `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` | 0 | 文档、roadmap、documentation impact 与 contributioncheck docs suite |
| `pwsh -NoProfile -File scripts/check-openspec-example-impact-declaration.ps1` | 0 | 合法 `新增示例` 声明 |
| `pwsh -NoProfile -File scripts/check-openspec-roadmap-status-consistency.ps1` | 0 | roadmap 与 OpenSpec 状态一致 |
| `pwsh -NoProfile -File scripts/check-openspec-documentation-impact.ps1` | 0 | 对外语义文档承诺与实际路径一致 |
| `openspec validate establish-host-execution-plan-bridge-contract --strict` | 0 | OpenSpec 工件有效 |
| `git diff --check` | 0 | 无 whitespace error |

专用 shell contract gate 已提供并接入质量门禁，但本 Windows 环境没有可用 Git Bash：`D:\git\Git\bin\bash.exe` 不存在，WSL relay 也没有 `/bin/bash`。因此不能宣称 shell gate 在本环境取得 exit 0；PowerShell 等价 gate 已取得 exit 0。该环境限制不影响 Go、PowerShell、replay 或 example 证据。

## Quality gate and security scan

完整质量门禁已使用 3000 秒总预算、900 秒步骤预算和串行模式运行。严格模式完成 77 个步骤，唯一失败是既有依赖与 Go 标准库漏洞报告：`govulncheck ./...` exit 3；其余 repo hygiene、docs、OpenSpec、agent-mode、mailbox/context/realtime/native-event/host-plan、Go test、race、A64、contract gate 和 lint 步骤均通过。该失败不是本 change 的编译、测试、lint 或 contract assertion 失败。

仓库质量门禁明确支持 `BAYMAX_SECURITY_SCAN_MODE=warn`：warn 只将 `govulncheck` 发现作为 warning，其他步骤保持严格。本次使用 `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=3000`、`BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=900`、`BAYMAX_QUALITY_GATE_PARALLELISM=1` 和 `BAYMAX_SECURITY_SCAN_MODE=warn` 重跑完整门禁，77 个步骤 exit 0；`govulncheck` 的既有漏洞按 warning 记录，未被写成漏洞已修复。由此 7.2 具备完整验证证据。

## Worktree hygiene and preserved user artifacts

门禁产生的 `.gocache-*` 目录和 `.artifacts/a64/*` 属于用户/门禁产物，未删除且不纳入本 change 提交。此前 `tool/contributioncheck` 中断遗留的 `examples/agent-modes` 临时 sentinel 已恢复，并以定向测试确认通过。

## Delivery status

任务 1.1–2.4、3.1–6.4、7.1、7.2 和 7.3 已按上述代码、测试、文档、gate 与归档脚本证据勾选。任务 7.4（提交、同步 master、合并、推送、合并验证及分支清理）将在本分支交付完成后补记。
