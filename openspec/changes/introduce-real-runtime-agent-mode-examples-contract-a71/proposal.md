## Why

当前 `examples/agent-modes` 已具备统一目录与可执行入口，但多数模式仍是同构模板实现，模式语义差异和主干链路覆盖深度不足。需要独立于 a62 的新提案，把示例从“可运行骨架”升级为“可迁移、可回归、可审计”的真实示例。 

## What Changes

- 新增独立提案 `a71`，对 `examples/agent-modes` 全量 28 模式执行“真实逻辑替换”，不与 a62 任务混用。
- 为每个模式的 `minimal` 与 `production-ish` 实现真实运行时路径，禁止复用同构占位模板（仅 runner + 本地打分工具的泛化实现）。
- 按模式语义补齐主干依赖接入：`core/runner`、`orchestration/*`、`runtime/config`、`context/*`、`memory`、`mcp/*`、`model/*`、`tool/local`（按模式选择，不强制每个模式覆盖全部域）。
- 新增真实示例阻断门禁：
  - `check-agent-mode-real-runtime-semantic-contract.sh/.ps1`
  - `check-agent-mode-readme-runtime-sync-contract.sh/.ps1`
- 增强 smoke 与回放验证：默认执行 `minimal + production-ish` 双变体，输出必须包含模式语义相关验证证据（非统一模板字段）。
- 为每个示例 README 增加强制章节并与代码同 PR 同步：`Run`、`Prerequisites`、`Real Runtime Path`、`Expected Output/Verification`、`Failure/Rollback Notes`。
- 建立“模式 -> 真实能力点 -> 契约 -> 门禁 -> 回放夹具”的对照矩阵，并纳入 CI 阻断。

## Example Impact Assessment

- 修改示例

## Capabilities

### New Capabilities
- `real-runtime-agent-mode-examples-contract`: 定义 agent-modes 全量示例的真实逻辑替换标准、模式语义覆盖要求、README 同步要求与验收口径。

### Modified Capabilities
- `go-quality-gate`: 增加真实示例语义门禁与 README 同步门禁，要求 shell/PowerShell parity 且失败即阻断。

## Impact

- 示例代码：`examples/agent-modes/*/{minimal,production-ish}/main.go`
- 示例文档：`examples/agent-modes/*/{minimal,production-ish}/README.md`、`examples/agent-modes/MATRIX.md`、`examples/agent-modes/PLAYBOOK.md`
- 质量门禁：`scripts/check-agent-mode-real-runtime-semantic-contract.*`、`scripts/check-agent-mode-readme-runtime-sync-contract.*`、`scripts/check-agent-mode-examples-smoke.*`、`scripts/check-quality-gate.*`
- 测试与回放：`integration/*`（新增示例语义回归）、`tool/diagnosticsreplay`（新增/扩展回放夹具）
- 约束边界：保持 lib-first，不引入平台化控制面，不新增并行同域提案拆分。
