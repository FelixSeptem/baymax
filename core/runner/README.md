# core/runner 组件说明

## 功能域

`core/runner` 是 Baymax 主循环引擎，统一实现：

- `Run` 非流式执行
- `Stream` 流式执行
- 模型调用、工具调用、终态收敛

Canonical 架构入口：`docs/runtime-harness-architecture.md`

## 架构设计

`Engine` 以状态机推进一次 run：

- `Init`：初始化策略与运行态
- `ModelStep`：模型请求、能力选择、上下文装配
- `DecideNext`：工具分发、背压与重试决策
- `Finalize/Abort`：生成 `run.finished` 与错误归类

运行时横切能力通过 Option 注入：

- `runtime/config.Manager`：策略与热更新快照
- `tool/local.Dispatcher`：本地工具调度
- `context/assembler.Assembler`：上下文装配语义阶段编排
- 安全治理：Action Gate、Model IO Filter、Security Alert

## 关键入口

- `runner.go`
- `security.go`
- `security_delivery.go`

## 边界与依赖

- `core/runner` 不承载 provider 协议细节；具体 SDK 适配在 `model/*`。
- 仅发射标准事件，不直接写诊断存储。
- 与 `Run/Stream` 相关的语义（取消、超时、背压、reason code）需保持契约一致。

## 配置与默认值

- Runner 行为默认由 `runtime/config.Manager` 快照驱动，遵循 `env > file > default`。
- 无独立 runner 配置文件，关键默认值通过 runtime config 子域提供（如 backpressure、security、recovery）。
- 未注入可选能力时采用保守默认：无中断恢复、无额外降级分支。

## 可观测性与验证

- 关键验证：`go test ./core/runner -count=1`，并在质量门禁中执行 `go test -race`。
- 与运行结果一致性相关的主线契约可在 `docs/mainline-contract-test-index.md` 查到映射。
- 事件发射由 `types.EventHandler` 驱动，最终通过 `RuntimeRecorder` 单写落盘。

## 扩展点与常见误用

- 扩展点：`runner.With...` Option 注入（registry/config/event handler/security hooks）。
- 常见误用：在 runner 层引入 provider 特有字段，破坏 `core/types` 统一契约。
- 常见误用：绕开事件链路直接写 diagnostics，导致统计口径漂移。
