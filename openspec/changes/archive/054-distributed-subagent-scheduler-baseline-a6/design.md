## Context

当前代码已具备：
- `orchestration/teams` 与 `orchestration/workflow` 的单进程编排基线；
- `a2a` 的最小跨 agent 互联（并正在 A4 加固 delivery/version）；
- `runtime/config`、`runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 的统一配置与单写观测链路。

但现阶段仍缺乏“跨进程调度协调”核心能力：
- 无持久化任务队列与租约接管机制；
- 无 parent-child run 预算治理；
- worker 崩溃恢复依赖调用方自拼装，无法形成稳定调度语义。

A6 目标是补齐 distributed subagent scheduler baseline，作为后续平台化能力（RBAC/control-plane/多租户）之前的最小生产底座。

## Goals / Non-Goals

**Goals:**
- 提供分布式 subagent 调度最小闭环：enqueue/claim/heartbeat/complete/requeue。
- 提供 lease 过期接管语义，支持 worker 异常退出后的任务恢复。
- 提供 submit/claim/result 幂等语义，避免重复执行导致状态膨胀。
- 提供 parent-child run 治理基线（深度、并发、预算）并对接可观测字段。
- 通过 A2A 接入远端协作执行，同时保持 A2A/MCP 边界稳定。

**Non-Goals:**
- 不实现完整 control plane（UI、租户、RBAC、审计门户）。
- 不在本期引入跨地域全局调度与复杂负载均衡策略。
- 不改写核心 runner 单回合状态机为分布式编排器。
- 不将 scheduler 职责下沉到 MCP transport 包。

## Decisions

### 1) 新增独立调度模块，走库优先（library-first）
- 方案：新增 `orchestration/scheduler`（或等价路径）作为调度子域，定义 `QueueStore`、`LeaseManager`、`Dispatcher` 接口。
- 原因：保持与 runner/A2A 解耦，便于单元测试与后续后端替换。
- 备选：把调度逻辑塞进 `a2a`。拒绝原因：会耦合 peer 语义与调度语义，扩展困难。

### 2) 采用“至少一次投递 + 强幂等提交”语义
- 方案：任务分发采用 at-least-once，结果提交采用幂等键（`task_id + attempt_id`）去重；重复提交不重复放大聚合计数。
- 原因：在分布式故障下可恢复性优先，幂等保证副作用可控。
- 备选：exactly-once。拒绝原因：实现与运维复杂度过高，不适合作为 A6 基线。

### 3) lease + heartbeat + visibility timeout 作为接管机制
- 方案：worker claim 任务获得有界 lease，需周期 heartbeat；lease 超时任务自动回到可领取状态并增加 attempt。
- 原因：这是主流任务调度系统最小可靠性闭环，且便于契约测试。
- 备选：只依赖进程内重试。拒绝原因：无法覆盖进程崩溃和节点失联。

### 4) parent-child run 治理采用“硬阈值 fail-fast”
- 方案：引入 `max_depth`、`max_active_children`、`child_budget_timeout` 等硬阈值；超过阈值直接拒绝派生。
- 原因：避免 subagent fanout 失控，保持资源上界可解释。
- 备选：仅观测不拦截。拒绝原因：在压力场景下风险不可控。

### 5) 观测与边界治理沿用现有 single-writer 与 shared contract gate
- 方案：scheduler 仅发事件，不直接写 `runtime/diagnostics`；新增 reason namespace（`scheduler.*`、`subagent.*`）纳入 gate。
- 原因：延续现有诊断一致性与回放幂等机制。
- 备选：scheduler 自建诊断写口。拒绝原因：破坏单一事实源与边界约束。

## Risks / Trade-offs

- [Risk] at-least-once 可能带来重复执行风险  
  → Mitigation: 强制幂等键 + 去重提交 + 契约测试覆盖重复提交与重放。

- [Risk] lease 配置不当导致任务抖动（频繁回收/重领）  
  → Mitigation: 提供最小安全默认值与 fail-fast 校验，补充指标（lease_expired_count/reclaim_count）。

- [Risk] parent-child 治理阈值过严影响吞吐  
  → Mitigation: 阈值可配置，先默认保守并输出 budget 拒绝原因供调优。

- [Risk] 与 A4/A5 并行开发产生接口漂移  
  → Mitigation: A6 只依赖稳定 A2A 接口与配置快照，增加联调任务与 gate 校验。

## Migration Plan

1. 定义 scheduler 数据模型与接口（任务、attempt、lease、heartbeat、result、idempotency key）。
2. 增加 runtime config 字段与校验（scheduler/subagent guardrails），默认保持关闭或兼容路径。
3. 实现内存后端 + 一个持久化后端（建议 sqlite）并补并发安全测试。
4. 打通 A2A 与 scheduler 的 dispatch/claim/complete 链路。
5. 扩展 timeline/diagnostics 字段并验证 replay 幂等。
6. 补齐集成测试（worker crash、lease expire takeover、duplicate submit/result）。
7. 文档与 gate 收敛后启用默认策略。

回滚策略：
- 关闭 scheduler enable 开关，回退到当前本地执行路径；
- 保留新增字段为 additive 可空，不影响旧消费者；
- 如持久化后端异常，降级为内存后端并保持 fail-fast 报警。

## Open Questions

- A6 是否强制引入 sqlite 持久化后端，还是先内存 + 文件 WAL 过渡？
- parent-child 预算是否以“时间预算”为主，还是在 A6 同时引入 token/cost 预算？
- 是否在 A6 期内引入 scheduler 独立质量门禁脚本（如 `check-scheduler-contract.*`）？
