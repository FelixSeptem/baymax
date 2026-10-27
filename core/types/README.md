# core/types 组件说明

## Checkpoint contract compatibility（提案中）

现有 metadata-oriented `RunCheckpointStore` API 保持兼容。独立 `native_continuation.v1` DTO/extension 提供 bounded identity/digest/expiry/causality reference；source owner 同时持久化有界 private continuation material，并以摘要、依赖 profile/revision 和原子 claim 校验后恢复。reference 不含 tool arguments/results；存储应按执行数据保护。direct 三类 pause reason 有跨进程证据；composite forms 与 planbridge resumed 仍不支持。

## 功能域

`core/types` 定义跨模块共享契约，是 Baymax 的稳定接口层：

- Runner / Model / Tool / MCP / Skill 抽象接口
- Run / Model / Tool 调用数据结构
- 事件与 timeline 语义
- 错误分类与可重试语义

Canonical 架构入口：`docs/runtime-harness-architecture.md`

## 架构设计

该包保持“纯契约、零业务逻辑”定位：

- 上层编排使用 `Runner`、`ModelClient`、`EventHandler` 等接口解耦
- 下层实现通过 `RunRequest`、`ModelRequest`、`ToolCallOutcome` 等 DTO 对齐
- 可观测性使用统一 `Event` 与 `ActionTimelineEvent`
- 错误通过 `ErrorClass` + `ClassifiedError` 归一

策略契约同样集中在该包：

- 循环策略：`LoopPolicy`
- 本地并发策略：`LocalDispatchPolicy`
- MCP 策略：`MCPRuntimePolicy`
- Action Gate / HITL Clarification 契约

## 关键入口

- `types.go`
- `native_execution_event.go`：`NativeExecutionEvent` / `NativeExecutionProjector` / `NativeExecutionOptions`，纯投影和 `native_execution_event.v1` 能力声明。
- `native_event_history.go`：source-neutral `SourceRef`、capabilities、opaque cursor、bounded page、reader interface 与稳定 typed errors。`NativeHistoryIdentity` 独立于 callback `EventID` 和协议 `Sequence`。

## 边界与依赖

- 该包不依赖上层实现细节，供 `core/*`、`model/*`、`orchestration/*`、`runtime/*` 共同复用。
- 任意跨模块行为变化应先更新此处契约，再同步实现与测试。

## 配置与默认值

- 不持有 env/file/hot-update 配置键。`NativeExecutionLimits` 是纯投影的 constructor-only 合同限制：每 scalar 1024 UTF-8 bytes、16 citations、host references JSON 8 KiB、单条事件 JSON 64 KiB；显式 override 的四项均须正值。
- 输出为独立 JSON 快照，数字用 `json.Number` 保持精度，拒绝未知 reference 字段、非法 UTF-8、不可序列化值、循环/超深值和自定义 JSON/Text marshaler（标准 `time.Time` 除外）。Realtime 空 binding 的 subscription/outcome 元数据也受 event byte 限制。

直接 `ProjectEvent` 复用既有 `MapEventToProtocol`，保留 native type/version、iteration、trace/span 和 payload references。固定 source profiles 声明 mapping_reference/callback_order、无 history/cursor/dedup/gap；相同 ID 或重复回调不合并。独立 `ProjectRealtimeBinding` 复用已有 Realtime subscription 验证和 normalization，不提供 worker/history/cache。`SourceState` 读取 native terminal_outcome.state、terminal_state、state；`IsTerminal` 为无状态分类，`input_required` 不属终态。完整接入合同见 [eventadapter](../../host/eventadapter/README.md)。

## 可观测性与验证

Native event history 是独立 source-owned contract：`NativeEventHistoryReader.Capabilities` 声明 source namespace、sequence scope、retention/floor、cursor scope/version、durability 与 terminal recovery；`ReadAfter` 返回 detached `NativeEventPage`。通用 `core/history` journal 只在 owner 显式注入时记录 runner、child、team、workflow 的 source facts；默认执行路径仍由 `UnsupportedNativeEventHistoryReader` 明确拒绝。mailbox 的 `History.TerminalOutcome` 仅将 `acked`、`dead_letter`、`expired` 分类为该 mailbox 生命周期的 terminal，`SourceState` 保留对应值；TTL 的 `mailbox.expired` transition 若 owner 已将记录送入 DLQ，terminal outcome 仍为 `dead_letter`。这些都不表示 runner 完成或 host business commit。`ErrHistoryUnsupported`、`ErrCursorMalformed`、`ErrCursorForeign`、`ErrCursorExpired`、`ErrHistoryGap` 和 `ErrSourceUnavailable` 可用 `errors.Is` 判定。Memory journal 只保证进程内 replay；FileStore 只保证同机单写、持久化后的顺序重开，不提供多写者 CAS 或跨机器协调。读 history 不等于 Run resume、checkpoint restore、retry 或业务 commit。source namespace/ID 各限 1024 UTF-8 bytes，cursor 限 4096 bytes，page limit 为 1–200；非法 source/limit 返回 `ErrHistoryInvalid`。详见 [mailbox](../../orchestration/mailbox/README.md)。

显式绑定 history writer 后，runner、child、team、workflow owner 的 callback append failure 通过 `NativeEventHistoryError()` 暴露为可 `errors.Is(ErrSourceUnavailable)` 判定、且携带 source kind/namespace/ID 的 health error。该 callback history 是观察投影，不是 effect admission gate；append 失败不证明相关 effect 未执行，不能自动重跑 parent/child/step，也不能宣称与 consumer projection 原子提交。workflow continuation 的 claim/result persistence 则由 workflow source owner 返回 unavailable 或 `recovery_required`，重开时不自动 redispatch 不确定 effect。

显式绑定 `core/history.ProjectionLedger` 后，source owner 还可以持久化 bounded source commit、digest、terminal evidence 与 projection acknowledgement。`ProjectionLedger.ReadCommittedAfter` 是只读 source evidence；重复 acknowledgement 按 source event identity 幂等，digest/reference/cursor 冲突返回 `ErrHistoryConflict`。`Journal` 与 ledger 的 source commit 窗口若无法共同确认，会返回 `ErrProjectionRecoveryRequired`，不得重跑 effect、parent/child/step 或把 callback health 当作外部业务投影的 exactly-once 证明。只有 file-backed journal 与 file-backed ledger 同时显式绑定时，history capability 才声明 `ProjectionRecovery`；该声明仍仅限同机单写、顺序重开，不提供 Danny 数据库的跨库原子事务、多写者 CAS 或跨机器协调。

fresh process 只能通过同一 source journal 的 retained event 调用 `Journal.ReconcileProjection` 补回 ledger；该调用返回 `ProjectionRecoveryResult`，以 `Duplicate` 明确幂等重放，并在 source identity、sequence、event identity、detached digest、terminal evidence 或 cursor 不一致时 fail closed。它只修复 source projection evidence，不恢复执行、不重试 effect、不改变 mailbox lifecycle，也不写 RuntimeRecorder 或业务数据库。

该契约不增加 runtime configuration key、默认值、诊断 schema 或 recorder 字段；离线 `examples/native-execution-events` 现展示 fresh-process reconciliation、duplicate 与 conflict/unsupported 拒绝。history 存储仍由 source owner 显式提供，failure/recovery 边界由离线 contract fixture 与故障注入测试验证。

- 关键验证：`go test ./core/types -count=1`。
- 契约变更后应同步更新主干契约索引与跨模块回归用例。
- 错误分类、事件字段、接口签名是回归重点。

## 扩展点与常见误用

- 扩展点：新增 DTO 字段时优先 additive，并配套文档与契约测试。
- 常见误用：在 types 包引入业务逻辑或 IO 依赖，破坏“纯契约层”边界。
- 常见误用：在未同步测试与文档情况下调整共享字段，导致跨模块漂移。

## Deployment scope contract

`DeploymentScope` is the source-neutral, bounded vocabulary for deployment
claims. Existing owners advertise only
`same_host_single_writer_sequential_reopen` with source-owned fencing and
sequential transactions. An absent or unsupported scope cannot be interpreted
as distributed support; `ErrDeploymentUnsupported`, `ErrForeignWriter` and
`ErrStaleWriter` remain fail-closed typed outcomes.
