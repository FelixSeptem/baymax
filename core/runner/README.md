# core/runner 组件说明

## 功能域

`core/runner` 是 Baymax 主循环引擎，统一实现：

- `Run` 非流式执行
- `Stream` 流式执行
- 模型调用、工具调用、终态收敛

## 架构设计

`Engine` 以状态机推进一次 run：

- `Init`：初始化策略与运行态
- `ModelStep`：模型请求、能力选择、上下文装配
- `DecideNext`：工具分发、背压与重试决策
- `Finalize/Abort`：生成 `run.finished` 与错误归类

运行时横切能力通过 Option 注入：

- `runtime/config.Manager`：策略与热更新快照
- `tool/local.Dispatcher`：本地工具调度
- `context/assembler.Assembler`：CA1-CA4 上下文装配
- 安全治理：Action Gate、Model IO Filter、Security Alert

## 关键入口

- `runner.go`
- `security.go`
- `security_delivery.go`

## 边界与依赖

- `core/runner` 不承载 provider 协议细节；具体 SDK 适配在 `model/*`。
- 仅发射标准事件，不直接写诊断存储。
- 与 `Run/Stream` 相关的语义（取消、超时、背压、reason code）需保持契约一致。
