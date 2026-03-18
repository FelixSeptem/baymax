# a2a 组件说明

## 功能域

`a2a` 提供 Baymax 的 Agent-to-Agent 互联基础能力，覆盖三类调用语义：

- 同步提交并等待：`Submit` + `WaitResult`
- 异步提交后回报：`SubmitAsync` + `ReportSink`
- 状态查询与结果解析：`Status` / `Result`

## 架构设计

当前实现采用“客户端策略 + 服务端任务状态机”结构：

- `InMemoryServer` 负责任务生命周期（`submitted/running/succeeded/failed/canceled`）
- `Client` 负责路由、版本协商、交付模式协商与等待策略
- `DeterministicRouter` 基于能力集与优先级做确定性 peer 选择
- 异步回报通过 `ReportSink` 抽象支持 callback/channel 两种落点

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
