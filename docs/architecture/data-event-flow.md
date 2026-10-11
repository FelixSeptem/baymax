# 数据与事件流

```text
input/config -> admission -> context snapshot -> provider request
                                      |
                                tool/MCP attempt
                                      |
                         RuntimeRecorder (single writer)
                                      |
                    timeline + diagnostics + replay fixture
                                      |
                            terminal outcome/checkpoint
```

Canonical facts 只在其 owner 组件产生；投影层可以添加 nullable/default 字段，但不能重写 owner 事实。敏感输入应在 recorder 边界按安全策略脱敏，replay fixture 不携带 credential。

真实 source `types.EventHandler` 回调可显式 fan-out 到 RuntimeRecorder 与 `host/eventadapter`。后者经 `core/types.NativeExecutionProjector` 生成 `native_execution_event.v1` 独立快照，再同步交给获授权的 application sink；宿主将业务 references 与 execution record、AG-UI/SSE 投影关联。RuntimeRecorder 继续是 diagnostics 单写入口，不作为事件总线；observer Status 仅是交付证据，不改变 Run terminal 或授权。

Direct source 只有串行 callback order，没有全局 sequence、durable cursor 或唯一事件 ID；shared TraceID 对应的回调全部保留。Realtime 纯投影仅保留已有 source-owned binding 的序列、overlap/gap 与 cursor outcome。并行 parent/child 通过 source references 关联，各自观察。暂停、resolution 和 terminal 由 source 生成，不能从 RunResult 合成。参见 [接线及失败合同](../../host/eventadapter/README.md)。

Run 与 Stream 必须对同一输入产生语义等价的工具决策、错误分类和终态；事件顺序可因传输方式不同而分段，但不可改变事实含义。

验证入口：[Diagnostics Replay](../diagnostics-replay.md)、[主线契约测试索引](../mainline-contract-test-index.md)。
