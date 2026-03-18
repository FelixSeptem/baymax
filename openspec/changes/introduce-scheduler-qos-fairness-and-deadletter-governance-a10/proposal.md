## Why

当前 scheduler 具备可靠性基线（lease/requeue/idempotency），但仍缺少 QoS、公平性与死信治理契约，在高并发混合负载下容易出现高优任务抢占失衡、低优任务饥饿与重试风暴。A10 需要在 `lib-first` 约束下补齐调度治理层，保证可解释调度顺序与可回归失败收敛。

## What Changes

- 为 scheduler 增加 QoS 策略：默认 `fifo`，可选启用 `priority` 调度模式。
- 优先级来源固定为 task 字段（不依赖外部回调），支持稳定序列化与重放。
- 增加公平性窗口：`max_consecutive_claims_per_priority=3`，防止高优队列长期独占。
- 增加 dead-letter（DLQ）治理语义：默认关闭，启用后超限任务进入 dead-letter 队列并停止常规重试。
- 增加重试退避治理：指数退避 + 抖动（jitter），降低失败重放抖动与雪崩重试风险。
- 扩展 timeline reason 与 run 摘要 additive 字段，覆盖 qos/fairness/dlq/retry-backoff 信号。
- 将 QoS/DLQ 契约测试并入现有 `check-multi-agent-shared-contract.*` 阻断门禁。

## Capabilities

### New Capabilities
- `distributed-subagent-scheduler-qos`: 定义 scheduler QoS、公平性窗口、DLQ 与退避治理契约。

### Modified Capabilities
- `distributed-subagent-scheduler`: 增加调度顺序、公平性、DLQ 转移与退避语义要求。
- `runtime-config-and-diagnostics-api`: 增加 scheduler QoS/DLQ/backoff 配置域与 additive 摘要字段。
- `action-timeline-events`: 增加 qos/fairness/dlq/retry-backoff reason taxonomy 与关联字段要求。
- `runtime-module-boundaries`: 增加 QoS/DLQ 逻辑边界与 single-writer 约束声明。
- `go-quality-gate`: 增加 QoS/DLQ 回归契约并纳入共享阻断门禁。

## Impact

- 代码：`orchestration/scheduler/*`、`runtime/config/*`、`runtime/diagnostics/*`、`observability/event/*`、`integration/*`、`tool/contributioncheck/*`、`scripts/check-multi-agent-shared-contract.*`。
- 测试：新增优先级调度、公平性窗口、DLQ 转移、指数退避抖动、Run/Stream 等价契约测试。
- 文档：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/mainline-contract-test-index.md`、`docs/v1-acceptance.md`、`docs/development-roadmap.md`。
- 兼容性：默认 `fifo` 与 `dlq disabled`，新增能力全部 additive，不破坏现有行为。
