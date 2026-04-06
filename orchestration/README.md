# orchestration 组件说明

## 功能域

`orchestration` 是多代理编排域，负责把基础执行能力组织成协作流程：

- `composer`：统一组合入口与运行时桥接
- `workflow`：DAG 工作流执行与 checkpoint/resume
- `teams`：多角色协作（serial/parallel/vote）
- `scheduler`：任务队列、lease、重试、QoS、DLQ、子任务护栏
- `mailbox`：统一消息协调契约（command/event/result + lifecycle/query）
- `invoke`：A2A 调用桥接层（仅保留 mailbox bridge 作为公开 canonical 入口）
- `collab`：协作原语（handoff/delegation/aggregation）统一抽象
- `snapshot`：统一 state/session snapshot 导入导出合同

Canonical 架构入口：`docs/runtime-harness-architecture.md`

## 架构设计

设计原则是“组合优先，不吸收下层细节”：

- `composer` 负责装配 `runner + workflow + teams + scheduler + a2a`
- `workflow` 负责 step DSL 解析、校验、重试/超时/恢复语义
- `teams` 负责本地/远程任务执行与结果收敛
- `scheduler` 负责任务生命周期状态机与治理策略
  - async-await 路径支持 `awaiting_report` + callback/poll 双来源终态收敛；
    poll fallback 仅作为 callback 缺失时的补偿路径，仲裁规则固定 `first_terminal_wins + record_conflict`。
  - 子任务路径统一遵循 timeout resolver（`profile -> domain -> request`），并执行父子预算收敛 `min(parent_remaining, child_resolved)`。
- `composer` managed 入口支持 readiness admission guard：
  - `blocked` 默认 fail-fast 拒绝；
  - `degraded` 按策略 `allow_and_record|fail_fast` 控制；
  - deny 路径不触发 enqueue / mailbox publish / lifecycle mutation。
- `mailbox` 负责 command/event/result envelope、ack/retry/ttl/dlq 与查询语义，
  并提供可选 lifecycle worker 原语（`consume -> handler -> ack|nack|requeue`）
- `invoke` 负责与 mailbox 对齐的 A2A 调用桥接；公开入口固定为 `MailboxBridge`
- `collab` 负责跨路径一致的 handoff/delegation/aggregation 语义
  - 支持默认关闭、可显式开启的有界 primitive retry（sync delegation + async submit）
- `snapshot` 负责版本化 manifest、digest 校验、strict/compatible 恢复决策与 operation 幂等收敛。

所有编排路径通过标准 `action.timeline` / `run.finished` 事件暴露状态。

## 关键入口

- `composer/composer.go`
- `workflow/engine.go`
- `teams/engine.go`
- `scheduler/scheduler.go`
- `mailbox/mailbox.go`
- `invoke/mailbox_bridge.go`
- `collab/primitives.go`
- `snapshot/manifest.go`
- `snapshot/contract.go`

## 边界与依赖

- 编排层不承载 provider 协议或 MCP transport 细节。
- 编排层不直接依赖 `runtime/diagnostics` 包；run/timeline 汇总仍经 `observability/event.RuntimeRecorder` 单写收口。
- mailbox publish/lifecycle 诊断通过 `runtime/config.Manager.RecordMailboxDiagnostic` 写入，保持配置域边界与可查询性。
- `snapshot` 只定义合同层语义，不重写既有 checkpoint/snapshot 存储事实源。
- reason namespace（如 `team.*`、`workflow.*`、`scheduler.*`、`subagent.*`）需保持稳定以支持契约测试。

## 配置与默认值

- 编排默认配置由 `runtime/config` 提供：如 collab 开关、scheduler QoS、recovery 边界。
- `composer.collab.enabled=false`、`scheduler.dlq.enabled=false` 等保守默认保证 pre-1 行为稳定。
- `composer.collab.retry.enabled=false`；默认治理参数为 `max_attempts=3`、`backoff_initial=100ms`、`backoff_max=2s`、`multiplier=2.0`、`jitter_ratio=0.2`、`retry_on=transport_only`。
- mailbox worker 默认关闭：`mailbox.worker.enabled=false`，默认轮询 `100ms`，默认错误策略 `requeue`；
  lease/recover 默认值为 `inflight_timeout=30s`、`heartbeat_interval=5s`、`reclaim_on_consume=true`、`panic_policy=follow_handler_error_policy`。
- scheduler 托管路径保持单一重试 owner；不叠加 primitive retry，避免 compounded retries。
- composer/scheduler 子任务 dispatch 支持 operation profile 透传与 timeout-resolution metadata（用于 QueryRuns/Task Board 解释最终生效预算来源）。
- async-await reconcile 默认关闭（`scheduler.async_await.reconcile.enabled=false`），启用后按 `interval/batch_size/jitter_ratio` 节流对账。
- readiness admission 默认关闭（`runtime.readiness.admission.enabled=false`），启用后在 managed Run/Stream 执行前统一准入。
- workflow graph composability 默认关闭，需显式开启。
- unified snapshot 默认关闭（`runtime.state.snapshot.enabled=false`），默认恢复模式为 `strict`，兼容窗口默认 `1`。

## 当前非目标

- 不在编排层引入 MQ/控制面能力（Kafka/NATS/RabbitMQ/UI/RBAC）。
- 不承诺 exactly-once，仅保证 at-least-once 下的幂等收敛。

## 可观测性与验证

- 关键验证：`go test ./orchestration/... -count=1` 与 `go test ./integration -run 'TestComposer|TestScheduler|TestWorkflow' -count=1`。
- snapshot 合同回归：`go test ./orchestration/snapshot -count=1` 与 `go test ./integration -run '^TestUnifiedSnapshot' -count=1`。
- 质量门禁中覆盖多代理性能基线、full-chain smoke、shared contract suites。
- 关键观测字段：dispatch reason、attempt id、recovery replay counters。

## 扩展点与常见误用

- 扩展点：新增协作原语策略、扩展 scheduler store、引入新的 workflow step 执行器。
- 常见误用：在编排层直接依赖模型或传输 SDK，导致跨域耦合。
- 常见误用：修改 reason namespace 而不更新契约索引与回归测试。
