## Why

当前配置与诊断核心能力实现落在 `mcp/runtime` 路径下，职责边界与命名语义不一致，后续扩展到 runner/tool/observability 时会持续引入耦合成本。现在进行职责重构并补齐文档体系，可以在保持现有语义一致的前提下降低未来迭代复杂度。

## What Changes

- 将通用配置与管理能力从 `mcp/runtime` 抽离到全局 runtime 配置模块（路径待设计阶段定案），MCP 子模块仅保留 MCP 语义逻辑。
- 统一依赖注入方式：`mcp/http`、`mcp/stdio`、`core/runner`、`tool/local`、`observability/*` 使用同一运行时配置快照接口。
- 明确 API 兼容策略：提供迁移期兼容别名/转发层，避免一次性破坏调用方。
- 完善文档：架构职责边界图、模块 owner 表、配置字段索引、迁移指南、FAQ、示例引用矩阵。
- 建立文档质量门禁：README 与 docs 关键章节对齐检查、术语与字段命名一致性检查。

## Capabilities

### New Capabilities
- `runtime-module-boundaries`: 统一定义 runtime 核心模块职责边界、依赖方向与扩展约束。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 从 MCP 子域实现升级为全局 runtime 能力，覆盖多子系统接入语义与兼容迁移策略。

## Impact

- 代码目录：`mcp/runtime`、`core/runner`、`tool/local`、`observability/*`、新增全局 runtime config 包。
- 文档目录：`README.md`、`docs/*`（架构、配置、迁移、示例导航）。
- API 影响：可能出现包路径调整；通过兼容层与迁移指引控制升级成本。
- 质量保障：新增职责边界测试/静态约束检查与文档一致性检查。
