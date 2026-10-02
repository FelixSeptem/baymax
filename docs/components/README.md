# 组件责任索引

组件页面统一回答 purpose、owned facts/state、输入输出、依赖边界、生命周期、失败/回滚、可观测性、验证和事实源问题。

| 组件 | 页面 | 主要事实源 |
| --- | --- | --- |
| Runner | [runner.md](runner.md) | `core/runner`, runtime tests |
| Context | [context.md](context.md) | `context/`, context contract gates |
| Provider | [providers.md](providers.md) | `model/<provider>` |
| Tool | [tools.md](tools.md) | `tool/*` |
| Orchestration | [orchestration.md](orchestration.md) | `orchestration/*`, `a2a/*` |
| Runtime | [runtime.md](runtime.md) | `runtime/*` |
| Observability | [observability.md](observability.md) | `observability/*`, diagnostics |
| Adapters | [adapters.md](adapters.md) | `adapter/*`, `integration/*` |
| MCP | [mcp.md](mcp.md) | `mcp/*` |
| Extensions | [extensions.md](extensions.md) | extension packages and gates |
| Skills | [skills.md](skills.md) | `skill/*`, `.codex/skills` |
