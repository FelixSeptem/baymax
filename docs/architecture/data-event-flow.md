# 数据与事件流

本页区分四类东西：source owner 产生的事实、面向消费者的事件投影、diagnostics/replay 证据，以及宿主拥有的业务 effect。它们可以相关，但不能互相冒充。

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

| 层 | 产生者 | 可以做什么 | 明确不能做什么 |
| --- | --- | --- | --- |
| source fact | runner/workflow/scheduler/mailbox/tool 等 owner | 推进 lifecycle、产生 terminal、保留 source history | 让另一个 adapter 猜测或重写事实 |
| event projection | `types.EventHandler`、native adapter | 将已产生事实映射为 bounded event/reference | 生成 durable cursor、恢复执行或提交 effect |
| diagnostics/replay | `RuntimeRecorder`、fixture/replay owner | 单写、查询、离线重建和验证 evidence | 调用模型、工具、retry、resume 或 ack |
| host effect | 宿主 authorization/effect/commit owner | 业务授权、外部写入、幂等/CAS 和最终确认 | 把 Baymax ack/digest/terminal 当作业务 commit |

真实 source `types.EventHandler` 回调可显式 fan-out 到 RuntimeRecorder 与 `host/eventadapter`。后者经 `core/types.NativeExecutionProjector` 生成 `native_execution_event.v1` 独立快照，再同步交给获授权的 application sink；宿主将业务 references 与 execution record、AG-UI/SSE 投影关联。RuntimeRecorder 继续是 diagnostics 单写入口，不作为事件总线；observer Status 仅是交付证据，不改变 Run terminal 或授权。

Direct source 只有串行 callback order，没有全局 sequence、durable cursor 或唯一事件 ID；shared TraceID 对应的回调全部保留。Realtime 纯投影仅保留已有 source-owned binding 的序列、overlap/gap 与 cursor outcome。并行 parent/child 通过 source references 关联，各自观察。暂停、resolution 和 terminal 由 source 生成，不能从 RunResult 合成。参见 [接线及失败合同](../../host/eventadapter/README.md)。

Run 与 Stream 必须对同一输入产生语义等价的工具决策、错误分类和终态；事件顺序可因传输方式不同而分段，但不可改变事实含义。

验证入口：[Diagnostics Replay](../diagnostics-replay.md)、[主线契约测试索引](../mainline-contract-test-index.md)。阅读顺序见[架构总览](overview.md)和[运行生命周期](runtime-lifecycle.md)。
