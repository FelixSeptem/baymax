## Context

Scheduler 当前提供 durable queue + lease claim + heartbeat + requeue + idempotent terminal commit 的可靠性基线，但调度策略仍偏基础：
- 默认单队列 FIFO 顺序领取；
- 缺少优先级与公平性约束；
- 失败任务重试缺少退避治理；
- 超限失败任务缺少 dead-letter 终态隔离。

随着 A8/A9 持续推进多代理组合与恢复语义，调度治理缺口会成为下一阶段稳定性瓶颈。A10 目标是在不平台化、不引入控制面的前提下，为 scheduler 增加可回归的 QoS、公平性与 dead-letter 治理。

## Goals / Non-Goals

**Goals:**
- 保持默认行为兼容：`fifo` 仍为默认调度模式。
- 在启用 QoS 时支持 task 字段优先级调度。
- 引入公平性窗口 `max_consecutive_claims_per_priority=3`，避免高优长期独占。
- 引入 `dlq` 语义（默认关闭），启用后超限任务进入 dead-letter 队列并停止常规重试。
- 引入重试退避 `exponential + jitter`，减少重试风暴。
- 补齐 timeline/diagnostics/additive 字段与共享门禁契约。

**Non-Goals:**
- 不引入多租户、全局负载均衡、控制面调度能力。
- 不新增外部调度依赖或消息系统（保持库内最小实现）。
- 不改变 single-writer 诊断写入路径。
- 不在本提案内引入策略回调 DSL（优先级来源仅 task 字段）。

## Decisions

### 1) 默认调度模式保持 FIFO
- 方案：`scheduler.qos.mode` 默认 `fifo`，显式启用后才进入 `priority`。
- 原因：你确认默认 FIFO；可保持升级无行为突变。
- 备选：默认 priority。拒绝原因：会改变现有任务领取顺序，回归风险高。

### 2) 优先级来源固定为 task 字段
- 方案：优先级仅来自 task payload/字段，不依赖 host callback。
- 原因：你确认使用 task 字段；更利于持久化与重放确定性。
- 备选：host callback 动态优先级。拒绝原因：重放可复现性变差。

### 3) 公平性窗口固定默认值 3
- 方案：同优先级连续 claim 默认上限为 3，触发后让渡到下一优先级可领取任务。
- 原因：你确认值为 3；这是高优吞吐与低优活性之间的实用平衡点。
- 备选：无限连续 claim。拒绝原因：低优饥饿风险不可控。

### 4) DLQ 默认关闭，显式启用后生效
- 方案：`scheduler.dlq.enabled=false` 默认值；开启后超限任务入 DLQ。
- 原因：你确认默认关闭；保证兼容性与渐进上线。
- 备选：默认开启。拒绝原因：现有依赖“持续重试”语义的链路可能受影响。

### 5) 重试退避采用指数 + 抖动
- 方案：失败重试等待按指数增长，并施加有界随机抖动。
- 原因：你确认指数 + jitter；可缓解同时失败任务的重试同步峰值。
- 备选：固定 backoff。拒绝原因：在并发失败场景更容易形成脉冲。

### 6) 门禁并入既有 shared-contract gate
- 方案：扩展 `check-multi-agent-shared-contract.*`，纳入 qos/fairness/dlq suite。
- 原因：维持阻断入口统一，减少 CI 分裂。
- 备选：新增独立脚本。拒绝原因：维护成本与审查复杂度上升。

## Risks / Trade-offs

- [Risk] 启用 priority 后调度行为更复杂，排查难度上升  
  → Mitigation: 增加显式 reason 和摘要字段（claim 来源、fairness 触发、dlq 转移）。

- [Risk] 公平窗口过小影响高优吞吐  
  → Mitigation: 参数可配置，默认 3，允许按 workload 调整。

- [Risk] DLQ 默认关闭可能延续重试风暴  
  → Mitigation: 文档明确推荐生产启用，并提供超限观测与告警字段。

- [Risk] 退避抖动使调试时序不稳定  
  → Mitigation: 提供 deterministic seed/测试桩，契约测试固定时序输入。

## Migration Plan

1. 扩展 scheduler 配置模型（qos/fairness/dlq/backoff）并保持默认兼容。  
2. 实现 priority + fairness 领取策略（保留 fifo 路径）。  
3. 实现 retry backoff（指数+抖动）与失败超限 DLQ 转移。  
4. 扩展 timeline/diagnostics 映射与 additive 字段。  
5. 补 integration + contract tests（顺序、公平、DLQ、Run/Stream 等价）。  
6. 将 suite 并入 shared-contract gate，更新 docs/index。  

回滚策略：
- 关闭 qos 或恢复 `fifo` 模式；
- 关闭 dlq；
- 保留新增字段 additive，不影响旧消费者。

## Open Questions

- priority 模式下是否需要对“同优先级内部顺序”固定为稳定 FIFO（建议保持稳定 FIFO）。
- DLQ 记录是否在 A10 首版只保留最小元数据，还是直接包含完整 attempt 历史。
- 指数退避的默认参数（initial/max/jitter ratio）是否按配置 profile 分级提供。
