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

## 宿主计划接入

`orchestration/planbridge` 提供 opt-in 的 `host_execution_plan.v1` API：先以 `Describe` 生成 reference-only source facts，再以独立、受信的 native binding 和 live `Admit` callback 调用 `Compile`，最后使用 `Compiled.Run` 或 `Compiled.Stream` 委托一个既有 owner。Compile 无副作用地校验整份计划和 binding；它不把 composer.Run/Stream 解释为自动执行通用 graph。

执行形态 direct/subagent/team/workflow、投递 inline/mailbox 和触发 manual/scheduled/resumed/callback 独立。首版目标：inline direct、teams parallel/vote、flat workflow、隔离独占 scheduler 的 local child；shared-queue child、mailbox delivery、resumed/native checkpoint resume 和组合动态 action continuation 在 dispatch 前拒绝。Stream 只承接实际 source streaming/callback 能力；同步 child handler 不自动获得 token streaming。

宿主先授权实际 input/handler/tool 与业务写范围，再提交 plan identity/version、authority/tool digests、真实 parent references 和完整预算事实；digest 仅证明规范化描述一致，不认证 callback 或权限。编译快照不能替代实际 activation 的权限/readiness/lease/parent budget 重检。replan 使用新版本；恢复继续使用明确的 source-native API，不以相同 run ID 或 replay 伪装恢复。

真实 source 事件通过上文 observer 投影，PlanVersion/PlanDigest 使用独立 HostReferences；plan ID 由 application execution record 关联。direct input_required 保留 pause；team/workflow/child 首版要求 terminal-only binding。聚合终态、scheduler commit 与 observer status 均不替代业务 commit gate。limits、错误、opaque binding 责任及拒绝反例见 [package README](../../orchestration/planbridge/README.md)，可运行的离线 fake model/tool 示例见 [host-execution-plan](../../examples/host-execution-plan/README.md)。

停用新增 bridge 回到既有 owner 显式调用；保留 task/attempt/checkpoint/outbox 和业务记录，不能因此撤销已发生 effects。公开 API/默认值/限制/错误/支持矩阵发生变化时，必须在同一 OpenSpec 同步维护文档并通过治理门禁。

## 暂停后的恢复准入

native dynamic action resume 由原 Engine checkpoint owner 处理；Host 通用 Authorizer 不自动证明暂停前的业务 revision、plan/write scope 和当前权限仍匹配。`runner.WithHostResumeVerifier` 显式安装 `host_resume_admission.v1`，由宿主 callback 读取可信暂停绑定与当前事实，在 decision commit 前重验证；direct API 与既有 run.action.resume 共享该入口，不增加 command kind。默认不安装；安装 nil verifier 会 fail-closed。

完整 normalized identity 与 mode 必须保持一致才能读取 duplicate result，执行中返回 `runner.ErrDynamicActionInProgress`。接线/limits/错误/支持矩阵见 [恢复准入指南](host-resume-admission.md)，[离线示例](../../examples/host-resume-admission/README.md) 使用真实 runner 验证拒绝、恢复、duplicate 和 CAS。不要把 metadata/snapshot/replay 或同 run ID 当成新 Engine 的 durable continuation；planbridge resumed 仍 unsupported。宿主实际 effect 提交仍需 authority/revision/effect CAS，digest、resolved、command acceptance 与 runner completion 均不是业务 commit。

## 边界与回滚方式

宿主拥有 credential、网络、持久化和全局路由。接入失败时禁用新增 adapter/profile，恢复上一个显式配置；不要通过降级隐藏 contract 不匹配。
# Host execution plan bridge

For an opt-in bounded host plan, call `planbridge.Describe`, persist or review only its reference-only `SourceFacts`, then call `Compile` with the explicitly trusted native binding and a live `Admit` callback. Invoke `Compiled.Run` or `Stream`; verify source outcome independently before any business commit. A plan digest proves descriptor/binding consistency, never business authority. See `orchestration/planbridge` and `examples/host-execution-plan`; mailbox, resumed execution and shared child queues are rejected rather than adapted.
