# 配置与诊断

## 配置优先级

所有配置遵循 `env > file > default`。解析后形成不可变 snapshot；非法值和非法热更新必须 fail-fast + 原子回滚，不得部分应用。

完整键、默认值、校验和迁移映射见 [runtime-config-diagnostics.md](../runtime-config-diagnostics.md)。

## 诊断原则

- 写入只经过 `observability/event.RuntimeRecorder`。
- QueryRuns 和诊断新增字段采用 additive + nullable + default。
- credential、完整 prompt、敏感 header 和原始 secret 不进入 fixture 或导出 bundle。
- 诊断字段变更必须补 parser compatibility、replay 和 gate 覆盖。

## 排查顺序

1. 查看配置来源和 effective snapshot。
2. 查 Run/Stream terminal outcome 与 primary reason。
3. 查 tool/MCP/provider attempt/completion 事件。
4. 使用 bounded replay fixture 重现，不直接重放生产 secret。

验证：`go test ./runtime/config ./runtime/diagnostics -count=1` 和 [diagnostics replay](../diagnostics-replay.md)。
