# observability 组件说明

## 功能域

`observability` 提供事件与追踪基础设施：

- `event`：事件分发、timeline 解析、RuntimeRecorder
- `trace`：OTel trace/span 轻量封装

当前进度（2026-03-19）：
- 诊断写入维持 `RuntimeRecorder` 单写入口。
- A16/A17 新增的多代理 additive 字段均经事件映射落到 diagnostics。
- A18 统一检索能力将复用现有写入模型，不引入第二写入口。

## 架构设计

关键角色：

- `event.Dispatcher`：fan-out 到多个 `types.EventHandler`
- `event.RuntimeRecorder`：将标准事件映射为 runtime diagnostics（单写入口）
- `trace.Manager`：统一 `StartRun` / `StartStep`，并提取 TraceID/SpanID

`RuntimeRecorder` 会在写入前应用 runtime 配置脱敏策略，保证诊断与日志口径一致。
`RuntimeRecorder` 对重复事件按幂等键进行收敛，避免 replay 导致计数放大。

## 关键入口

- `event/dispatcher.go`
- `event/runtime_recorder.go`
- `trace/trace.go`

## 边界与依赖

- 业务模块负责“发事件”，`observability` 负责“记录与分发”，保持职责分离。
- 诊断写入应仅经过 `RuntimeRecorder`，避免多写入口导致统计漂移。
- 该域不承载业务策略判定，只承载观测语义映射。

## 配置与默认值

- 观测行为默认由 `runtime/config` 提供的 diagnostics 与 redaction 策略控制。
- trace 子域默认使用轻量包装；若未注入 OTel exporter，则保持本地最小 trace 语义。
- RuntimeRecorder 默认启用幂等去重，避免 replay 导致计数膨胀。

## 可观测性与验证

- 关键验证：`go test ./observability/event -count=1`。
- 观测契约回归需覆盖 timeline parser、runtime recorder 幂等与字段兼容。
- 与 `runtime/diagnostics` 联动验证写入收敛和查询可见性。

## 扩展点与常见误用

- 扩展点：新增事件处理器、扩展 trace span 标签、增强 recorder 归档策略。
- 常见误用：直接在业务域写 diagnostics，绕过 dispatcher/recorder 链路。
- 常见误用：引入非标准事件字段而不更新 parser 与索引文档。
