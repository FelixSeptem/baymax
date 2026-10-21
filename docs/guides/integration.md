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

### Workflow continuation（workflow-first）

当 capability manifest 明确声明 workflow 的 checkpoint、跨进程 restore 和 pause reason evidence 时，宿主可以使用 source-owned workflow continuation。恢复必须绑定原 graph/version/digest、parent/run/session/causation、step dependency、预算与 effect references，并通过 atomic claim；新 Engine 继续原 graph，不重新 Run parent、不重新 enqueue，也不重跑已完成 step/effect。`input_required`、`approval_required` 和 `budget_exhausted` 的 safe point、cancel/timeout/retry 竞争以及 Run/Stream parity 必须由 workflow owner 的 source evidence 支持。child/team/mailbox/planbridge resumed 仍返回 `unsupported_capability`，不得借用 workflow capability 或 history replay 模拟恢复。

默认部署边界仍是同机单写、持久化后的顺序重开；FileStore 不宣称多写者 CAS、跨机器协调或多实例 resume claim。checkpoint store 不可用、版本不兼容、foreign、stale、conflicting、expired 或 corrupt 时必须在副作用前 fail closed。

### Child/fanout continuation（child-owner phase）

Composer exposes an explicit source-owned child continuation boundary for local
and A2A child/fanout owners. Persist a bounded `ChildContinuation` at a real
safe point, reopen the same `ChildContinuationStore`, and call
`ResumeChildContinuation`. The owner validates the existing scheduler
task/attempt/lease and atomically claims the original logical child; it never
calls `SpawnChild`, re-enqueues, starts a new parent, or treats callback
history as restore material. MemoryStore is process-local; FileStore is
same-host, single-writer sequential reopen only. Duplicate terminal claims
return the retained projection, while lease loss, stale/foreign/expired/
corrupt records, store failure and in-progress claims fail closed.

`input_required`, `approval_required`, and `budget_exhausted` are independent
child capability cells. Each requires a source producer, persisted pending
action/consumed budget/deadline, terminal evidence and Run/Stream parity. The
plan bridge accepts a child resume binding only when a capability manifest and
`ChildContinuationOwner` are supplied. Team parallel/vote, mailbox host-plan
continuation and resumed planbridge remain `unsupported_capability`.

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

### Source-owned native history 与 restart replay

`types.NativeEventHistoryReader` 是只读 source-store extension。先由真实 owner 取得 `SourceRef`，查询 `Capabilities`，再用 `ReadAfter(ctx, source, confirmedCursor, limit)` 分页；page 的 events、next cursor 和 retention floor 均为 detached snapshot。不要以任意 run ID、callback EventID/TraceID 或 host sequence 替代 source/cursor。当前 source audit 如下：

| Source | Event namespace / sequence | Retention / cursor | Terminal recovery / restart | Unsupported reason |
| --- | --- | --- | --- | --- |
| runner | `runner.event.history.v1`；source instance 内递增 | bounded source cursor；retention floor | source owner journal；同机单写 FileStore 可重开 | 只有显式注入 runner journal 的 source instance 才支持；默认 engine 不自动创建 |
| isolated child | `child.event.history.v1`；source instance 内递增 | bounded source cursor；retention floor | source owner journal；同机单写 FileStore 可重开 | 只有显式注入 child journal 的 source instance 才支持；默认 composer 不自动创建 |
| team | `team.event.history.v1`；source instance 内递增 | bounded source cursor；retention floor | source owner journal；同机单写 FileStore 可重开 | 只有显式注入 team journal 的 source instance 才支持；不声明跨 member 全局顺序 |
| workflow | `workflow.event.history.v1`；source instance 内递增 | bounded source cursor；retention floor | source owner journal；同机单写 FileStore 可重开 | 只有显式注入 workflow journal 的 source instance 才支持；不等同 step checkpoint |
| mailbox MemoryStore | `mailbox.history.v1/<source-id>`；仅 source instance 内递增 | 最后 1024 lifecycle events；无时间保证；opaque `mailbox.history.v1` source-instance cursor | retained mailbox lifecycle；仅同进程 | 不支持 durable history / 跨进程 replay |
| mailbox FileStore | `mailbox.history.v1/<source-id>`；仅 source instance 内递增 | 最后 1024 lifecycle events；无时间保证；opaque `mailbox.history.v1` source-instance cursor | 同一 snapshot 的 source identity/journal 可重开读取；仅 mailbox lifecycle | checkpoint restore / business commit 不在此能力内 |

runner/child/team/workflow 的 source journal 通过各 owner 的 `NewNativeEventHistory` / `OpenNativeEventHistory` 显式构造；未注入 journal 的默认执行路径仍使用 `UnsupportedNativeEventHistoryReader`，不会订阅 callback 或激活执行。`planbridge.DefaultCapabilityManifest` 不推断具体 source；`CapabilityManifestWithHistory` 只基于匹配 source kind 的实际 facts 提升对应 form history/replay 能力，并将 namespace、sequence、retention/floor、cursor、terminal recovery 作为 `SourceHistory` 保留。source 不可用时返回 typed unavailable，不能将静态 manifest 或 DTO 编译成功当作可重放证据。

```go
source, err := owner.HistorySource(ctx) // owner is an existing *mailbox.Mailbox
if err != nil { return err }
caps, err := owner.Capabilities(ctx, source)
if err != nil { return err }
_ = caps // Check Supported and Durable against the host's actual requirement.
page, err := owner.ReadAfter(ctx, source, confirmedCursor, 100)
if err != nil { return err }
// Project page.Events read-only. Persist page.NextCursor only after host confirmation.
```

`limit` 必须在 1–200；source namespace/ID 各限 1024 UTF-8 bytes，opaque cursor 限 4096 bytes。`nil` cursor 从当前 retention floor 开始，floor 是第一条保留事件之前的 after-position（最后已删除 sequence）；它不证明更早历史完整。`Complete` 只表示当前 retained tail 已读完，并不声明 source 永久停止。

用 `errors.Is` 区分 `ErrHistoryUnsupported`、`ErrCursorMalformed`、`ErrCursorForeign`、`ErrCursorExpired`、`ErrHistoryGap` 与 `ErrSourceUnavailable`；非法 source/page limit 为 `ErrHistoryInvalid`。expiry/gap 可携带安全 floor evidence，但不能静默改读 floor/latest。FileStore 未提交批量写入、持久化失败或 legacy snapshot 缺 history 时 unavailable；读取不隐式 Flush。跨进程 consumer 必须重开同一已持久化 FileStore，再提交原 source 与已确认 cursor。

crash-before-consumer-projection 的恢复是重新读取 source 已保留事实；host 自行以 source-owned history identity 幂等投影并确认 cursor。它不调用模型、工具、handler、Run/Stream、claim/ack/nack、retry、resume 或 restore，不分配新 parent Run，也不改变 delivery、业务 commit 或 checkpoint 状态。并行 source 的顺序仅在各 source 内成立。离线可运行 [示例](../../examples/native-execution-events/README.md) 展示重开 replay、malformed/foreign/expired 与四类 unsupported；[contract index](../mainline-contract-test-index.md) 列出 gap/unavailable 等证据。

与 RuntimeRecorder 共用 source 时使用显式 fan-out，让 recorder 消费原始事件、observer 消费原始事件的独立快照；不要从 QueryRuns 或 RunResult 合成事件。关闭/停用 observer 即可回滚观察接入；保留业务记录，已经发生的外部写入需按业务规则处理。

## 宿主计划接入

`orchestration/planbridge` 提供 opt-in 的 `host_execution_plan.v1` API：先以 `Describe` 生成 reference-only source facts，再以独立、受信的 native binding 和 live `Admit` callback 调用 `Compile`，最后使用 `Compiled.Run` 或 `Compiled.Stream` 委托一个既有 owner。Compile 无副作用地校验整份计划和 binding；它不把 composer.Run/Stream 解释为自动执行通用 graph。

执行形态 direct/subagent/team/workflow、投递 inline/mailbox 和触发 manual/scheduled/resumed/callback 独立。首版目标：inline direct、teams parallel/vote、flat workflow、隔离独占 scheduler 的 local child；manifest 明确允许且 mailbox owner 可用时，`mailbox_async` 通过 source-owned `Mailbox.AdmitHostPlan` 接入，保留 message/delivery/lease/retry/DLQ/causation facts。shared-queue child、resumed/native checkpoint resume 和组合动态 action continuation 在 dispatch 前拒绝。Stream 只承接实际 source streaming/callback 能力；同步 child handler 不自动获得 token streaming。

宿主先授权实际 input/handler/tool 与业务写范围，再提交 plan identity/version、authority/tool digests、真实 parent references 和完整预算事实；digest 仅证明规范化描述一致，不认证 callback 或权限。编译快照不能替代实际 activation 的权限/readiness/lease/parent budget 重检。replan 使用新版本；恢复继续使用明确的 source-native API，不以相同 run ID 或 replay 伪装恢复。

真实 source 事件通过上文 observer 投影，PlanVersion/PlanDigest 使用独立 HostReferences；plan ID 由 application execution record 关联。direct input_required 保留 pause；team/workflow/child 首版要求 terminal-only binding。mailbox ack/nack/retry/DLQ、source terminal 和 host business commit 仍是独立事实；planbridge 不通过重新 Run parent 恢复 lease loss。聚合终态、scheduler commit 与 observer status 均不替代业务 commit gate。limits、manifest、错误、opaque binding 责任及拒绝反例见 [package README](../../orchestration/planbridge/README.md)，可运行的离线 fake model/tool 示例见 [host-execution-plan](../../examples/host-execution-plan/README.md)。

停用新增 bridge 回到既有 owner 显式调用；保留 task/attempt/checkpoint/outbox 和业务记录，不能因此撤销已发生 effects。公开 API/默认值/限制/错误/支持矩阵发生变化时，必须在同一 OpenSpec 同步维护文档并通过治理门禁。

## 宿主 context assembly/handoff（进行中提案；direct contract 已提供）

`establish-host-context-assembly-handoff-contract` 已提供显式 opt-in 的 `host_context_projection.v1` direct assembly contract。宿主仍先完成 role/source-trust/business-reference/tool allowlist 过滤，然后把 detached descriptor 和 current-authority resolver 放入非序列化的 `types.RunRequest.HostContextProjection` / `HostContextResolver`；direct Run 和 Stream 仅在现有 Context Assembler 启用时复用该 owner。若 assembler 禁用，既有 bypass 语义保持不变，projection 不会被 admit 或 resolve。Baymax 不保存私有 memory/knowledge、业务 revision、授权、application execution record 或 resolver 返回的 source body。

接入要求宿主提供版本化、bounded、reference-only descriptor 和当前授权的 source resolver。固定 constructor/API 限制是：serialized descriptor 最多 32 KiB、最多 64 个 opaque references、每个 reference 最多 256 UTF-8 bytes、每个 digest 最多 128 UTF-8 bytes。Baymax 按 structure/limits → canonical digest → optional plan correlation + parent/child capability subset → resolver → existing assembler 的顺序处理；仅在前述验证通过且 assembler enabled 时复用既有 assembler、compaction 和 handoff owner。digest 只证明投影一致性，不证明 source access、业务权限或 commit；handoff/replay 也不是恢复授权。child 必须收窄 capability；关联 restore 先走既有 source validation，再通过 `runner.RestoreHandoffWithHostProjection` 由宿主用 fresh resolver 重新授权，并只返回供 caller-owned injection 使用的 transient resolution；该 API 不自行注入 context、resume 或 run。不会增加 host wire command、outbox/mailbox worker、Provider SDK、业务 store、RuntimeRecorder/QueryRuns 字段、平行 terminal 或 recovery coordinator。

应用可以将 `*hostprojection.Error` 作为有限的拒绝面处理；稳定 `Code` 包括 `host_context.unsupported_version`、`host_context.invalid_descriptor`、`host_context.limit_exceeded`、`host_context.duplicate_reference`、`host_context.invalid_capability`、`host_context.digest_mismatch`、`host_context.capability_expansion`、`host_context.parent_evidence_missing`、`host_context.invalid_plan_correlation`、`host_context.resolver_denied` 与 `host_context.source_not_found`。这些错误均 fail-closed，且不携带 raw callback error 或 source body；不得按字符串重试或把 historical digest 当作授权。

当前只支持 assembler enabled 时的 direct Run/Stream assembly、legal existing `handoff.v1` cuts 与 explicit correlated restore validation/material resolution；cross-process continuation、resolver-independent restore、mailbox/workflow/team recovery 和 dynamic-action continuation 均不支持。可运行的离线示例以 fake model/tool、synthetic opaque reference 和 current-authority resolver 展示 admitted assembly、capability widening rejection、revoked restore 与 legacy rollback；reference-only replay、shell/PowerShell dedicated gates、标准 quality-path 接线与验证记录已交付。可追踪范围、拒绝矩阵和回滚在归档 [proposal](../../openspec/changes/archive/165-establish-host-context-assembly-handoff-contract/proposal.md)，运行方式见 [host-context-assembly](../../examples/host-context-assembly/README.md)。

## 暂停后的恢复准入

独立 `native_continuation.v1` durable profile 已实现 direct restore；旧 metadata-oriented `RunCheckpointStore` 不改变签名。以 `checkpoint.OpenFileStore` 建立本机单写 source，向每个 Engine 注入相同 `runner.WithNativeContinuation` 的 owner namespace、dependency profile/revision 与原始 binding。保留暂停结果的 `RunResult.NativeCheckpoint`，新 Engine 调用 `ResumeFromCheckpoint`；不要再次调用 public Run。存储缺失、损坏、过期、foreign、不可用或 interrupted attempt 均 fail closed，不降级。可运行离线入口见 [durable continuation](../../examples/durable-continuation-restore/README.md)。

`Approval: true` 将 Action Gate 的 require-confirm 边界暂停在 tool dispatch 前；只有经当前宿主验证的 confirm 才可继续。`Budget: true` 在 iteration/tool-call safe point 暂停，恢复需要 source `BudgetOwner` 批准带新 revision 的 `NativeBudget`，consumed 与原 deadline 必须不变。此 profile 不宣称支持其它无可靠 source counter 的预算维度。`input_required` 保留已完成 outcomes 与原 opaque action。host verifier 在 source claim 前且不持有 store 锁，不能代替 source restore；开启 `WithHostResumeVerifier` 才要求宿主授权事实检查，digest 本身不授予权限。

FileStore 范围为同一台机器的单写进程顺序重开，第二 writer 返回 `ErrCheckpointUnavailable`；不宣称跨机器 CAS。默认 checkpoint 24 小时且不晚于原绝对 deadline，completed duplicate result 从完成时保留 24 小时。公开 envelope 为 64 KiB、reference 256 UTF-8 bytes、digest 128 bytes、references 256 项；private continuation material 为 1 MiB、completed/pending entries 1024 项。constructor limits 只可收紧，容量必须为正，满时不驱逐活跃记录。文件含 normalized request、tool outcomes 和 pending arguments，应按执行数据保护，不写 credential。

恢复仅支持匹配的 Run/Stream mode。exact duplicate 返回 source-retained projection，不重发 callback 或重执行；conflict 拒绝。claim/start 后无 source result 的 attempt 返回 in-progress/recovery-required，不自动接管。successor 与 predecessor result 同一 source commit 发布。`CancelNativeCheckpoint` 只持久化未 claim checkpoint 的取消 fence，accepted 不是 terminal；claim 先获胜后的取消通过 live source safe point 确认。terminal commit 失败不向调用方或 callback 宣告 durable terminal。child/team/workflow/mailbox 与 planbridge resumed 仍 unsupported，不能通过重新 Run parent 或 history replay 补恢复。Realtime、host-context projection/resolver、skill loader、lifecycle hooks、active runtime-input control、Plan Notebook、启用的 Context Assembler 等含未持久化私有状态的 dependency profile 在 admission 前拒绝。

native dynamic action resume 由原 Engine checkpoint owner 处理；Host 通用 Authorizer 不自动证明暂停前的业务 revision、plan/write scope 和当前权限仍匹配。`runner.WithHostResumeVerifier` 显式安装 `host_resume_admission.v1`，由宿主 callback 读取可信暂停绑定与当前事实，在 decision commit 前重验证；direct API 与既有 run.action.resume 共享该入口，不增加 command kind。默认不安装；安装 nil verifier 会 fail-closed。

完整 normalized identity 与 mode 必须保持一致才能读取 duplicate result，执行中返回 `runner.ErrDynamicActionInProgress`。接线/limits/错误/支持矩阵见 [恢复准入指南](host-resume-admission.md)，[离线示例](../../examples/host-resume-admission/README.md) 使用真实 runner 验证拒绝、恢复、duplicate 和 CAS。不要把 metadata/snapshot/replay 或同 run ID 当成新 Engine 的 durable continuation；planbridge resumed 仍 unsupported。宿主实际 effect 提交仍需 authority/revision/effect CAS，digest、resolved、command acceptance 与 runner completion 均不是业务 commit。

## 边界与回滚方式

宿主拥有 credential、网络、持久化和全局路由。接入失败时禁用新增 adapter/profile，恢复上一个显式配置；不要通过降级隐藏 contract 不匹配。

## Application outbox dispatch（归档 166）

`establish-application-outbox-mailbox-recovery-contract` 已交付 `application_outbox_dispatch.v1` constructor-only 接入面。宿主通过 `outboxbridge.New` 注入当前 admission callback 与一个或多个 native owner，构造 bounded reference-only descriptor 后显式选择一个 target：mailbox delivery 或 scheduler task dispatch。宿主仍拥有 outbox、cron/DST/misfire、业务授权、effect 与 outbox confirmation；不得把业务 body、credential、callback error 或 effect 写入 descriptor。可运行 offline example、replay fixture 和 contract gate 见 `examples/application-outbox-recovery`、`tool/diagnosticsreplay` 与 `scripts/check-application-outbox-mailbox-recovery-contract.ps1`。

集成方必须分别判断 dispatch accepted、mailbox/scheduler lifecycle、host effect committed 和 host outbox confirmed。ack、任务 terminal、Run/Stream terminal、digest 和 replay 都不是业务 commit。恢复仅承诺 at-least-once：mailbox delivery retry、scheduler execution retry 与 host business-commit retry 不可叠加为 bridge retry。未来停用 bridge 时回到既有原生 owner 调用，并保留 pending outbox、message/task/attempt 及已提交 effect，交由宿主幂等对账。
# Host execution plan bridge

For an opt-in bounded host plan, call `planbridge.Describe`, persist or review only its reference-only `SourceFacts`, then call `Compile` with the explicitly trusted native binding and a live `Admit` callback. Invoke `Compiled.Run` or `Stream`; verify source outcome independently before any business commit. A plan digest proves descriptor/binding consistency, never business authority. See `orchestration/planbridge` and `examples/host-execution-plan`; mailbox requires an explicit manifest-approved native binding, while resumed execution and shared child queues are rejected.
