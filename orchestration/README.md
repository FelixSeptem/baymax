# orchestration 组件说明

## 功能域

`orchestration` 是多代理编排域，负责把基础执行能力组织成协作流程：

- `composer`：统一组合入口与运行时桥接
- `workflow`：DAG 工作流执行与 checkpoint/resume
- `teams`：多角色协作（serial/parallel/vote）
- `scheduler`：任务队列、lease、重试、QoS、DLQ、子任务护栏
- `invoke`：统一 A2A 同步调用抽象

## 架构设计

设计原则是“组合优先，不吸收下层细节”：

- `composer` 负责装配 `runner + workflow + teams + scheduler + a2a`
- `workflow` 负责 step DSL 解析、校验、重试/超时/恢复语义
- `teams` 负责本地/远程任务执行与结果收敛
- `scheduler` 负责任务生命周期状态机与治理策略
- `invoke` 负责 `submit + wait + normalize` 的 A2A 同步调用统一口径

所有编排路径通过标准 `action.timeline` / `run.finished` 事件暴露状态。

## 关键入口

- `composer/composer.go`
- `workflow/engine.go`
- `teams/engine.go`
- `scheduler/scheduler.go`
- `invoke/sync.go`

## 边界与依赖

- 编排层不承载 provider 协议或 MCP transport 细节。
- 编排层不直接写 `runtime/diagnostics`，必须经 `observability/event.RuntimeRecorder` 收口。
- reason namespace（如 `team.*`、`workflow.*`、`scheduler.*`、`subagent.*`）需保持稳定以支持契约测试。
