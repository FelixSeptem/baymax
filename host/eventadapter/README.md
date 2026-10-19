# Native execution event adapter

## 定位与接入合同

此包提供显式 opt-in 的串行 EventHandler；事件来自真实 source，宿主 sink 负责持久化和业务投影。core/types 的纯 projector 提供 native_execution_event.v1、独立 HostReferences 与冻结能力 profile。RuntimeRecorder 是原始事件的独立消费者，不作为实时总线。

| Source owner | 回调接缝 | identity / order | history / cursor |
| --- | --- | --- | --- |
| runner | Engine.Run/Stream/ResumeDynamicAction 的 EventHandler | mapping_reference / serial callback_order | unsupported |
| workflow / teams | WithTimelineEmitter 的原生 timeline emitter | mapping_reference / 单 source 串行回调 | unsupported |
| composer | WithEventHandler 的 action.timeline | mapping_reference / 单 source 串行回调 | unsupported |
| scheduler | WithTimelineEmitter 的原生 timeline emitter | mapping_reference / 单 source 串行回调 | unsupported |
| a2a | NewClient / NewInMemoryServer 的 timeline 参数 | mapping_reference / 单 source 串行回调 | unsupported |
| timeline | 已授权、显式 source binding 的原生 Event | mapping_reference / 单 source 串行回调 | unsupported |
| realtime | core/types 的 source-normalized binding 纯投影入口 | source-owned EventID / sequence | 仅保留已有 source outcome；无订阅 worker |

source 必须显式选择，不能按事件名猜测。并行 child/source 使用独立 observer，保留 source 给出的 parent/causation references；没有全局顺序。直接 callback 中相同 mapped EventID/TraceID 也会交付，不可用于去重。Realtime cursor replay 不等于 runner resume。

## Source journal 与实时投影边界

`orchestration/mailbox` 的 `NativeEventHistoryReader` 是独立 source-owner extension；它返回的 `NativeExecutionEvent.History` 保留 journal identity、source-local sequence 与 lifecycle terminal evidence。adapter 不创建 journal、不解析 cursor、不调用 reader，也不把 callback EventID/TraceID 或 host observation sequence 转成 history identity。宿主可将 history page 交给自己的只读投影逻辑；这不赋予 callback observer durable history/cursor/dedup 能力。当前 runner、child、team、workflow 仍通过 source audit 明确报告 unsupported history；mailbox MemoryStore 也不能宣称跨进程 durable replay。replay 与 checkpoint restore、Run resume 和业务 commit 的语义各自独立。

## Sink 与失败反馈

Sink.Consume(ctx, event) 同步调用，默认 deadline 1 秒，必须合作响应 context；库无法中断不合作的 Go 回调。不建立队列、每事件 goroutine 或自动 retry。observer context 独立于 Run context，取消 Run 后仍能交付 canceled 事实。同 observer 重入/并发 fail-fast；并发失败前已经进入的 sink 可能发生写入。

调用 Run/Stream/resume 后读取 Status：首个失败停止后续交付，sink error/timeout 为 unconfirmed，不能推断业务写入不存在。Close 幂等，只停止观察；不能 cancel/resume/retry Run。Status 计数饱和、不包含正文或原始 error。没有成功交付的 terminal 是 incomplete observation，不能合成 completed。input_required 是 pause，resolution 是 decision 收到，最终 completed/failed/canceled 取 source facts。业务提交继续经过宿主 commit gate。

## 限制与隐私

构造参数默认每 scalar reference 1024 UTF-8 bytes、最多 16 citations、host references JSON 8 KiB、整条事件 JSON 64 KiB。显式 limits 和 deadline 必须正值；无 runtime env/file/hot-update 新键。引用只允许 task、occurrence、plan version/digest、projection、knowledge generation 和 citation，不覆盖执行身份或授权。

源在同步快照期间不得并发修改 payload；回调返回后修改嵌套 map/list 不影响 sink 快照。JSON 数字保留为 json.Number，避免大整数丢失。引用的 JSON 类型必须与声明一致；`null` 对象、字段或 citation 元素均拒绝，缺省字段可省略。真实事件正文仅向获授权的宿主 sink 交付，不写入新增 diagnostics 或离线 fixture。

快照预检有最大 128 层深度及遍历 byte budget，拒绝循环和不可序列化值；自定义 `json.Marshaler` / `encoding.TextMarshaler` 在执行前拒绝，标准 `time.Time` 除外。Realtime 空 batch 的 subscription/outcome 元数据也受 event byte 限制；单次 batch 大小仍由既有 source binding 校验。本 API 不负责业务正文脱敏，source/sink 授权由宿主确认。

构造错误分类：`native_execution.invalid`、`limit`、`correlation`、`unsupported`、`serialization`（均使用 `native_execution.` 前缀）。交付首错：`observation.sink_error`、`timeout`、`concurrent`、`closed`、`context_canceled`（均使用 `observation.` 前缀）。投影失败不调用 sink；sink 开始后无法确认的 receipt 计入 Unconfirmed；首次失败后 callback 计入 Skipped。Received 包含投影失败，因此不要求 Delivered + Unconfirmed + Skipped 总是等于 Received。FirstEventReference 最多 256 UTF-8 bytes，counter 饱和。

`Close` 前已准入的 delivery 可能写入，Close 取消其合作式 context，receipt 为 unconfirmed；Close 后的新 callback 不进入 sink。独立 observer context 被取消时首错是 context_canceled。所有 Status 均通过短锁读取，sink 不持有该锁。停用前已确认 terminal 且无失败时 Close 本身不将观察改成 incomplete；Close 后新回调会暴露 observation.closed。

## 验证与回滚

受影响包：core/types、host/eventadapter、tool/diagnosticsreplay、integration。专用 gate：scripts/check-native-execution-event-contract.ps1/.sh。离线示例：examples/native-execution-events。停用 adapter 恢复原 EventHandler；保留业务记录和既有 source 状态，不能回滚外部 effect。
