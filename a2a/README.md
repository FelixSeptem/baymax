# a2a 组件说明

## 功能域

`a2a` 提供 Baymax 的 Agent-to-Agent 互联基础能力，覆盖三类调用语义：

- 同步提交并等待：`Submit` + `WaitResult`
- 异步提交后回报：`SubmitAsync` + `ReportSink`
- 状态查询与结果解析：`Status` / `Result`

当前主线路径：
- 同步/异步/延后协作统一由 `orchestration/mailbox` + `orchestration/invoke/mailbox_bridge` 收口。
- `a2a` 继续提供 submit/status/result 互联语义与互操作契约。

Canonical 架构入口：`docs/runtime-harness-architecture.md`

## 架构设计

当前实现采用“客户端策略 + 服务端任务状态机”结构：

- `InMemoryServer` 负责任务生命周期（`submitted/running/succeeded/failed/canceled`）
- `Client` 负责路由、版本协商、交付模式协商与等待策略
- `DeterministicRouter` 基于能力集与优先级做确定性 peer 选择
- 异步回报通过 `ReportSink` 抽象支持 callback/channel 两种落点

异步回报默认由运行时配置控制（`a2a.async_reporting.enabled` 默认 `false`），开启后遵循重试与去重语义。

交付与兼容策略已内建：

- delivery mode：`callback|sse`，支持 fallback
- card version：`strict_major` + `min_supported_minor`
- 错误归一：`transport|protocol|semantic` 三层

## 关键入口

- `interop.go`
- `async_reporting.go`

## 边界与依赖

- 只依赖 `core/types` 契约与事件接口，不直接写 `runtime/diagnostics`。
- 观测通过 `types.EventHandler` 发射 `action.timeline` 事件，后续由 `observability/event.RuntimeRecorder` 收口。
- A2A 语义不下沉到 `mcp/*` 传输层，保持协作语义与工具传输语义解耦。
- 主线建议经 `orchestration/invoke/mailbox_bridge` 接入 A2A；直接 `Submit+WaitResult` / `SubmitAsync+ReportSink` 属于兼容入口。

## 配置与默认值

- `a2a.async_reporting.enabled=false`（默认关闭异步回报管道）。
- `a2a.card.version_policy=strict_major`，`a2a.card.min_supported_minor` 控制版本协商下界。
- `a2a.delivery.mode=callback` 可切换 `sse` 并结合 fallback 策略。

## 可观测性与验证

- 关键验证：`go test ./a2a -count=1`。
- 质量门禁中通过 `scripts/check-quality-gate.*` 间接覆盖 A2A 协议回归。
- 事件口径：提交/回报/重试会发射标准 `action.timeline` 原因码，供 diagnostics 聚合。

## 扩展点与常见误用

- 扩展点：自定义 `ReportSink`、自定义 peer router、定制 version negotiation 策略。
- 常见误用：在业务层重复实现回报重试/去重，导致与 A2A 内建幂等语义冲突。
- 常见误用：把 transport 错误直接映射为业务终态，跳过 `transport|protocol|semantic` 分层。
