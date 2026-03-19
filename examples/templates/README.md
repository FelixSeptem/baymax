# Adapter Templates (A21)

这些模板用于外部集成接入的最小骨架示例，优先级固定为：
1. `MCP adapter`
2. `Model provider adapter`
3. `Tool adapter`

模板定位：
- 用于 onboarding 与迁移对照。
- 保持最小可运行，不等价于生产级框架。
- 生产环境应补齐鉴权、重试策略、可观测、容量治理与安全策略。

运行方式：

```bash
go run ./examples/templates/mcp-adapter-template
go run ./examples/templates/model-adapter-template
go run ./examples/templates/tool-adapter-template
```
