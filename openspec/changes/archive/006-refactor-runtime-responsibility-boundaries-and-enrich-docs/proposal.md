## Why

当前配置与诊断 API 核心能力实现落在 `mcp/runtime` 路径下，职责边界与命名语义不一致，后续扩展到 runner/tool/skill/observability 时会持续引入耦合成本。现在进行职责重构并补齐文档体系，可以在保持现有语义一致的前提下降低未来迭代复杂度。

## What Changes

- 将通用配置与管理能力从 MCP 单体 runtime 包抽离到全局 runtime 配置模块（路径待设计阶段定案）。
- 将诊断 API 能力抽离为全局 runtime 诊断接口层，并保留 MCP 诊断字段语义模型。
- 删除 `mcp/runtime` 包，改为按功能命名的 MCP 子包：`mcp/profile`、`mcp/retry`、`mcp/diag`。
- 统一依赖注入方式：`mcp/http`、`mcp/stdio`、`core/runner`、`tool/local`、`skill/loader`、`observability/*` 使用同一运行时配置快照接口。
- 统一诊断读取接口：MCP/runner/tool/skill/observability 通过同一诊断 API 获取最近运行摘要、调用摘要和配置快照信息。
- 明确 API 兼容策略：提供迁移期兼容别名/转发层，避免一次性破坏调用方。
- 完善文档：架构职责边界图、模块 owner 表、配置字段索引、迁移指南、FAQ、示例引用矩阵。
- 建立文档质量门禁：README 与 docs 关键章节对齐检查、术语与字段命名一致性检查。

## Capabilities

### New Capabilities
- `runtime-module-boundaries`: 统一定义 runtime 核心模块职责边界、依赖方向与扩展约束。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 从 MCP 子域实现升级为全局 runtime 能力，覆盖配置与诊断 API 的多子系统接入语义及兼容迁移策略。

## Impact

- 代码目录：`mcp/profile`、`mcp/retry`、`mcp/diag`、`core/runner`、`tool/local`、`skill/loader`、`observability/*`、新增全局 runtime config/diagnostics 包。
- 文档目录：`README.md`、`docs/*`（架构、配置、迁移、示例导航）。
- API 影响：可能出现包路径调整；通过兼容层与迁移指引控制升级成本。
- 质量保障：新增职责边界测试/静态约束检查与文档一致性检查。
