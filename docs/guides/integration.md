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

## 接入真实执行事件

用 `host/eventadapter.New` 构造 opt-in `types.EventHandler`，将它传入 `Engine.Run`、`Engine.Stream` 和 `ResumeDynamicAction`。先授权 sink 对 source payload 的访问，再显式绑定 `Projection.Source`、`ExpectedRunID` 和可选 `SessionID`；业务关联只放入 `HostReferences`。同一个 observer 面向一个串行 source stream，并行 child/source 分别构造 observer，依靠已有 parent/causation references 关联。

```go
observer, err := eventadapter.New(eventadapter.Options{
    Projection: types.NativeExecutionOptions{
        Source: types.ProtocolSourceRunner,
        ExpectedRunID: request.RunID,
        SessionID: request.SessionID,
        HostReferences: types.NativeExecutionHostReferences{TaskID: "task-reference"},
    },
    Sink: eventadapter.SinkFunc(func(ctx context.Context, event types.NativeExecutionEvent) error {
        return storeAuthorizedEvent(ctx, event) // 宿主存储必须合作响应 ctx。
    }),
})
if err != nil { return err }
defer observer.Close()
result, runErr := engine.Stream(ctx, request, observer)
status := observer.Status() // 独立评估 runErr/source outcome 与观察结果。
```

上面是接线片段；完整可运行的 fake model/tool、内存 sink 与 Run/Stream parity 示例见 [native-execution-events](../../examples/native-execution-events/README.md)。最小 `ModelClient` 的 Stream preflight 仍须显式 provider identity/capability discovery，可使用 `adapter/modelcapability`。

默认交付 deadline 为 1 秒；`DeliveryTimeout=0` 选择默认，负值拒绝，正值为 override。`Context=nil` 使用独立 Background context；不要复用可能被取消的 Run context，否则 source canceled 事实可能无法交付。`Sink.Consume` 同步执行，无队列、自动 retry 或 detached goroutine；任意不合作的 Go sink 无法被强制中断。source 不得在同步快照期间并发修改 payload。

每次 Run/Stream/resume 返回后检查 `Status`：首错停止后续交付；sink error/timeout 是 unconfirmed，可能已经写入，不能自动重试。`Incomplete` 表示观察失败或尚未确认收到 source terminal，不能据此改写 source outcome。`input_required` 是 pause，`run.dynamic_action.resolved` 是 decision 被接收；completed/failed/canceled 由 source terminal 决定，业务提交另走宿主 commit gate。

Direct callbacks 使用既有 mapper 的 mapping-reference ID、`sequence=0` 和串行 callback order，不能按 EventID/TraceID 去重，也没有 durable history/cursor 或全局顺序。必需的 unsupported 能力在构造时拒绝。Realtime 只经 `ProjectRealtimeBinding` 保留既有 source binding 的 identity/sequence/overlap/gap/expired outcome；该入口不订阅事件，也不恢复执行。各 owner 的接线矩阵、默认 limits 和失败分类见 [adapter 合同](../../host/eventadapter/README.md)。

与 RuntimeRecorder 共用 source 时使用显式 fan-out，让 recorder 消费原始事件、observer 消费原始事件的独立快照；不要从 QueryRuns 或 RunResult 合成事件。关闭/停用 observer 即可回滚观察接入；保留业务记录，已经发生的外部写入需按业务规则处理。

## 边界与回滚方式

宿主拥有 credential、网络、持久化和全局路由。接入失败时禁用新增 adapter/profile，恢复上一个显式配置；不要通过降级隐藏 contract 不匹配。
