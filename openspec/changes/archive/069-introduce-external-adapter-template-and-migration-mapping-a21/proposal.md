## Why

随着 A20 进入实施，lib-first 主链路能力已具备示例与门禁，但“外部适配接入样板 + 迁移映射”仍缺统一交付，导致新集成方需要跨目录自行拼装实践。A21 目标是补齐可复制的适配模板与迁移指引，降低接入摩擦并减少语义漂移。

## What Changes

- 新增外部适配样板能力，提供文档主导 + 最小代码模板（非生产级插件实现）。
- 样板覆盖优先级固定：`MCP adapter > Model provider adapter > Tool adapter`。
- 新增迁移映射文档，按“能力域 + 典型代码片段”双维度组织。
- 迁移策略统一表达为：`additive + nullable + default + fail-fast` 边界语义。
- 增加“常见错误与替代写法”章节，覆盖适配接入高频误区。
- 将模板与映射索引纳入 docs consistency 与 contributioncheck 可追溯路径。

## Capabilities

### New Capabilities
- `external-adapter-template-and-migration-mapping`: 定义外部适配样板交付、迁移映射结构、边界语义与可追溯校验要求。

### Modified Capabilities
- `api-reference-coverage`: 扩展 API 参考覆盖要求，新增外部适配样板入口与迁移映射导航。
- `go-quality-gate`: 扩展质量门禁要求，新增适配样板/迁移映射文档一致性与索引追踪校验。

## Impact

- 文档：
  - `README.md`
  - `docs/` 下新增外部适配样板与迁移映射文档
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 模板：
  - `examples/templates/*` 或 `docs/templates/*` 最小代码骨架（MCP/Model/Tool）
- 质量与索引：
  - `tool/contributioncheck/*`
  - `scripts/check-docs-consistency.*`
