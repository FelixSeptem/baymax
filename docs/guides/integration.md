# 集成指南

## 选择路径

- 单轮模型调用：从 `examples/01-chat-minimal` 开始。
- 工具闭环：参考 `examples/02-tool-loop-basic`。
- MCP：参考 `examples/03-mcp-mixed-call` 和 [MCP profiles](../mcp-runtime-profiles.md)。
- 多 agent：参考 `examples/07-multi-agent-async-channel`、`08`、`09`。
- 外部 adapter：先读 [template index](../external-adapter-template-index.md) 和 [migration mapping](../adapter-migration-mapping.md)。

## 最小集成步骤

1. 创建宿主配置并显式选择 Provider、model、tool 和 recorder。
2. 先运行非网络 fixture，再接入真实 endpoint；不要依赖自动探测。
3. 为 Run 和 Stream 使用相同的 policy、tool registry 和终态断言。
4. 将事件写入 `RuntimeRecorder`，按 [配置与诊断](configuration-and-diagnostics.md) 查询和脱敏。
5. 运行对应 package test、replay 和 conformance gate。

## 边界与回滚

宿主拥有 credential、网络、持久化和全局路由。接入失败时禁用新增 adapter/profile，恢复上一个显式配置；不要通过降级隐藏 contract 不匹配。
