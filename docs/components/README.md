# 组件责任索引

组件页面统一回答 purpose、owned facts/state、输入输出、依赖边界、生命周期、失败/回滚、可观测性、验证和事实源问题。它们是组件导航，不是平行的 API spec；字段约束和终态语义仍以代码、测试和 `openspec/specs` 为准。

建议先读[设计原则](../architecture/design-principles.md)、[运行生命周期](../architecture/runtime-lifecycle.md)和[模块 ownership](../architecture/module-ownership.md)，再进入具体组件。每页的“Verification / source links”给出最小验证入口；发生冲突时回到[文档事实源与迁移矩阵](../documentation-ownership.md)。

| 组件 | 页面 | 主要事实源 |
| --- | --- | --- |
| Runner | [runner.md](runner.md) | `core/runner`, runtime tests |
| Context | [context.md](context.md) | `context/`, context contract gates |
| Memory | [memory owner README](../../memory/README.md) | `memory/`, memory contract gates |
| Provider | [providers.md](providers.md) | `model/<provider>` |
| Tool | [tools.md](tools.md) | `tool/*` |
| Orchestration | [orchestration.md](orchestration.md) | `orchestration/*`, `a2a/*` |
| Runtime | [runtime.md](runtime.md) | `runtime/*` |
| Observability | [observability.md](observability.md) | `observability/*`, diagnostics |
| Adapters | [adapters.md](adapters.md) | `adapter/*`, `integration/*` |
| Native event adapter | [eventadapter owner README](../../host/eventadapter/README.md) | `host/eventadapter`, source event contracts |
| Context handoff/projection | [handoff](../../context/handoff/README.md), [host projection](../../context/hostprojection/README.md) | `context/handoff`, `context/hostprojection` |
| Integration verification | [integration README](../../integration/README.md) | `integration/*`, contract/replay suites |
| MCP | [mcp.md](mcp.md) | `mcp/*` |
| Extensions | [extensions.md](extensions.md) | extension packages and gates |
| Skills | [skills.md](skills.md) | `skill/*`, `.codex/skills` |

## 组件页阅读顺序

1. **Purpose**：确认组件解决的问题和明确非目标。
2. **Owned facts/state**：确认它是事实 owner、投影者还是只读查询者。
3. **Inputs/outputs**：确认输入准入、输出、typed error 和边界。
4. **Dependencies and forbidden dependencies**：确认允许/禁止的依赖方向。
5. **Lifecycle / failure / rollback**：确认状态转移、重试、取消、恢复和回滚 owner。
6. **Observability**：确认 RuntimeRecorder、事件投影和脱敏责任。
7. **Verification / source links**：确认测试、replay、gate 和 canonical source。
