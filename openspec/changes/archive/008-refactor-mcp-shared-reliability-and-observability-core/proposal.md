## Why

当前 `mcp/http` 与 `mcp/stdio` 在重试、超时、事件发射和诊断写入上存在重复实现，导致行为漂移和维护成本升高。现在进入 R3 前的稳定化窗口，适合先收敛为包内共享核心，再推进后续高级示例与多 provider 扩展。

## What Changes

- 新增 MCP 共享可靠性与可观测性核心（强封装，仅 `mcp` 包内复用），统一重试/backoff/超时包装、事件模板和诊断映射。
- 重构 `mcp/http` 与 `mcp/stdio`：保留 transport 差异逻辑，移除可共享重复逻辑并接入内部核心。
- 新增跨 transport 契约测试，确保相同失败/重试场景下的错误分类、诊断字段和事件语义一致。
- 增加重复代码收敛度量与验收门槛，要求本次重构后 `mcp/http + mcp/stdio` 重复逻辑片段相对下降（按基线计算）。
- 更新 README 与 docs，说明 shared core 职责边界与 transport-specific 部分。

## Capabilities

### New Capabilities
- `mcp-shared-reliability-core`: 在 `mcp` 域内提供内部共享的可靠性与可观测性执行核心，统一重试、事件与诊断语义。

### Modified Capabilities
- `mcp-runtime-reliability-profiles`: profile 在 HTTP/STDIO 两种 transport 上的执行语义改为通过共享核心统一实现。
- `runtime-module-boundaries`: 补充 `mcp/internal/*` 强封装边界与依赖方向约束。

## Impact

- 影响目录：`mcp/http`、`mcp/stdio`、新增 `mcp/internal/*`（或等价内部路径）、相关测试目录。
- 测试影响：新增跨 transport 契约测试与重复逻辑统计脚本/检查步骤。
- API 影响：对外 API 保持兼容；仅内部实现重构。
- 文档影响：`README.md`、`docs/mcp-runtime-profiles.md`、`docs/runtime-module-boundaries.md`。