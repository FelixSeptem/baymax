## Context

Baymax 已分别收口 Realtime event protocol、durable runtime event-stream binding、checkpoint/workspace provenance、diagnostics replay 和 terminal outcome taxonomy。它们各自定义了事件顺序、cursor、dedupe、恢复引用和终态分类，但尚未形成一条可验证的端到端恢复路径：观察者断连或停止消费时，source runtime 是否继续完成；恢复订阅是否能补回事实；catch-up 与 live tail 是否无缝切换；终态查询是否与事件流和 replay 一致。

本 change 采用 library-first、source-owned 设计。事件历史、游标、终态裁决、snapshot/recovery 和安全策略继续由现有模块拥有；新能力只定义跨模块的恢复协调、归一化结果和验证门禁，不创建事件存储、连接服务或第二套状态机。

### Source ownership audit

- `core/types` owns transport-neutral subscription validation, Realtime sequence/dedup normalization, and pure binding projection; it does not retain history or mutate a Run.
- Realtime owns event history, accepted cursors, sequence progression, interrupt/resume validation, retention, and slow-consumer outcomes.
- `core/types.TerminalOutcomeArbiter` owns normalized `first-terminal-wins`, duplicate idempotency, and conflict classification; retry/resume/attempt facts remain source-provided.
- `core/runner` projects the pure Realtime binding for Run and Stream callers without creating a listener, queue, connection manager, or recovery worker.
- `observability/event.RuntimeRecorder` is the only diagnostics writer; `runtime/diagnostics` stores additive query fields; `tool/diagnosticsreplay` validates normalized fixtures without live connectivity.

The recovery vocabulary frozen by this change is `catching_up`, `live`, `disconnected`, `stopped`, `terminal_available`, `expired`, `gap`, and `backpressure`. A recovery result carries bounded subscription/source/session/run correlation, source-provided cursor mode and sequence boundary, retained event facts, and an optional source-owned terminal outcome. Drift codes are `stream_sequence_gap`, `stream_recovery_terminal_drift`, `stream_recovery_retained_facts_drift`, and `stream_recovery_run_stream_parity_drift`.

## Goals / Non-Goals

**Goals:**

- 为观察者生命周期定义 `active -> disconnected|stopped -> catching_up -> live|terminal` 的可验证边界。
- 固化 source-owned cursor 恢复：断连后只允许从已接受的 cursor catch-up，并在有效边界后切换 live tail。
- 保证 catch-up/live-tail overlap 复用既有 event ID、sequence 和 dedupe 语义，不产生重复事实。
- 将 recovery result 与 terminal outcome、partial output、已完成 tool-call facts 和 run/session correlation 关联起来。
- 让事件流、终态查询、RuntimeRecorder 和 diagnostics replay 对同一运行产生语义等价结果。
- 覆盖 Run/Stream parity、retention expiry、backpressure、observer stop、cancel、timeout 和 Provider stream failure。
- 提供 versioned fixture、integration test、drift 分类和 shell/PowerShell parity gate。
- Example Impact Assessment：`修改示例`。更新 `realtime-interrupt-resume` 或等价 agent-runtime 示例的文档基线和运行断言，演示断连、catch-up、live-tail 去重与终态查询。

## Example Impact Assessment

修改示例

**Non-Goals:**

- 不引入 REST/SSE/WebSocket/gRPC/JSON-RPC listener、托管连接管理器或平台化控制面。
- 不新增 event store、全局 binding queue、retention worker、scheduler 或 exactly-once 承诺。
- 不改变 Realtime interrupt/resume、terminal outcome 或 snapshot 的 source-of-truth 和状态迁移。
- 不把观察者恢复误定义为 Runtime 任务重试；恢复订阅与 retry/resume action 保持分离。
- 不把所有 Provider 错误转换成成功返回，也不绕过 policy、sandbox、配置 fail-fast 边界。

## Decisions

### 1. 用 source-owned recovery session 组合现有 binding，而不是新增传输层

恢复协调器接收 bounded subscription、source/session/run filter、既有 cursor 和 delivery policy，向 source 请求 catch-up 结果，再在 source 声明的 handoff boundary 后进入 live tail。协调器只投影状态和分类，不持有跨订阅事件历史。

备选方案：

- 在 runtime 内增加统一事件队列：可简化消费端，但会复制 retention/backpressure 所有权并违反 library-first 边界。
- 直接为每种传输实现 reconnect：能快速适配单一宿主，但会产生 HTTP/SSE 与核心 Runtime 的平行语义。

选择组合现有 binding，因为它能复用 124 已归档的 cursor、dedupe、retention 和 backpressure 合同，并保持传输可替换。

### 2. 恢复状态只表达观察者交接，不扩展 Run 状态机

恢复状态使用独立的 observation outcome：`catching_up`、`live`、`expired`、`gap`、`backpressure`、`disconnected`、`terminal_available`。Run 仍只能通过既有 terminal outcome 结束；观察者断连不得把 Run 改回 `working`，也不得自动触发 retry/resume。

备选方案：把断连视为 Run cancel 或 retry。该方案会把消费端生命周期错误地传播到业务执行，并破坏 terminal first-wins，因此不采用。

### 3. 终态采用双重确认但单一裁决

事件流可以先收到终态事件，也可以在恢复时先查询 source-owned terminal snapshot。实现必须通过现有 terminal outcome arbiter 归一化，遵守 `first-terminal-wins`；重复终态只产生幂等结果，晚到冲突写入 diagnostics，不覆盖业务终态。恢复结果必须携带 `terminal_available`、terminal outcome reference 和 observed cursor，但不重新推断 retry/resume 信息。

备选方案：以最后收到的终态为准。该方案无法处理乱序和晚到冲突，也会造成查询与事件流不一致。

### 4. Catch-up/live-tail handoff 使用明确边界和去重键

source 必须返回最后一个 catch-up cursor/sequence 与 live handoff boundary。协调器验证边界单调性，使用既有 event ID/dedup key 消除重叠；缺失必要 progression 时返回 deterministic `stream_sequence_gap`，不声称已进入 live。协调器不合成 cursor、sequence 或历史事件。

### 5. 失败分类与观测字段采用 additive schema

新增 recovery outcome、source resolution、cursor state、terminal correlation 和 conflict marker 均为 additive、nullable、defaultable 字段。诊断写入只经过 `RuntimeRecorder`；OTel 保持低 cardinality，不输出高基数 cursor 或 causation ID。非法订阅、过期 cursor、handoff gap 和不兼容 backpressure 在 source mutation 前 fail-fast。

### 6. 用 fixture 驱动 drift gate，而不是依赖网络或真实 Provider

新增 `runtime_event_stream_terminal_recovery.v1` fixture，覆盖成功恢复、overlap dedupe、gap、expired cursor、disconnect、backpressure、cancel/timeout/provider failure 和 Run/Stream parity。replay 只读取 fixture 与 source-owned normalized output，不连接外部服务；shell/PowerShell gate 使用同一分类码和退出语义。

## Risks / Trade-offs

- [Risk] 不同 source 对 cursor、retention 或 live handoff 的能力声明不一致 → [Mitigation] 所有能力显式声明 source outcome；未知能力分类为 `unresolved`，不得静默降级到 latest。
- [Risk] catch-up 与 live tail 之间存在事件乱序或重复 → [Mitigation] 强制验证 monotonic boundary，复用 event ID/dedupe，并为 gap/duplicate 建立 drift fixture。
- [Risk] 观察者恢复与业务 retry/resume 被调用方混淆 → [Mitigation] 使用独立 observation outcome，文档和 API 名称避免使用 task retry 术语，禁止自动触发 action。
- [Risk] 终态已写入但终态事件尚未送达 → [Mitigation] recovery 可查询 source-owned terminal snapshot；最终裁决仍通过 terminal arbiter，保持 first-terminal-wins。
- [Risk] 新增 diagnostics 字段导致历史 replay 失败 → [Mitigation] 字段全部 nullable/defaultable，混合版本 fixture 回归作为 gate 必跑项。
- [Risk] 示例只覆盖正常连接，未证明真实恢复 → [Mitigation] 先更新 MATRIX/README 文档基线，再添加可执行断连、补收和终态查询断言。

## Migration Plan

1. 先新增 capability spec、fixture schema、drift taxonomy 和 gate 约束，不改变现有运行路径。
2. 为 durable binding/realtime source 增加 recovery adapter 与 normalized observation outcome，默认沿用现有行为；不启用新传输。
3. 接入 terminal arbiter、diagnostics correlation 和 replay，完成 Run/Stream parity 与故障注入测试。
4. 更新 realtime 示例文档和运行断言，随后接入 quality/docs gate。
5. 若回归或宿主不支持恢复能力，回滚点是禁用新 recovery projection；既有 event protocol、binding、terminal outcome 与 snapshot 语义保持不变。

## Open Questions

- source-owned terminal snapshot 查询是否需要统一最小接口，还是继续由各 source adapter 提供等价 projection？
- backpressure 的 `paused` 与 `dropped` 是否需要在本 change 中冻结跨 source 的最小字段集合，还是只冻结分类码？
- retention expiry 后是否允许宿主显式请求从新的 cursor 开始，或必须由宿主重新建立 latest subscription？
