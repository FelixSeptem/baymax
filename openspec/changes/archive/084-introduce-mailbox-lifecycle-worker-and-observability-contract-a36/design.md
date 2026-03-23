## Context

A35 解决的是 mailbox runtime 接线问题，重点在 shared mailbox 实例、配置接线与 publish 路径可观测。  
当前剩余缺口是 mailbox lifecycle 在运行时层面的“消费闭环原语”与“reason taxonomy 治理”：

- `orchestration/mailbox` 已实现 `Consume/Ack/Nack/Requeue` 与 TTL/DLQ 语义，但缺少统一 worker loop 抽象。
- 主链路诊断以 publish 为主，lifecycle 节点（consume/ack/nack/requeue/dead_letter/expired）缺少统一写入语义。
- reason code 目前可自由写入，缺少冻结集合与 gate 阻断，长期存在语义漂移风险。

## Goals / Non-Goals

**Goals:**
- 提供库级 mailbox lifecycle worker 原语（默认关闭）。
- 固化 worker 默认策略：
  - `enabled=false`
  - `poll_interval=100ms`
  - `handler_error_policy=requeue`
- 扩展 mailbox lifecycle 诊断写入并保证 query/aggregate 可追踪。
- 冻结 lifecycle reason taxonomy，并纳入 quality gate 阻断。

**Non-Goals:**
- 不引入外部 MQ、平台控制面或托管任务看板。
- 不重写 scheduler 语义，不把 worker 变成新的全局调度器。
- 不改变 A32 async-await 收敛仲裁规则。
- 不修改 A35 mailbox wiring 目标边界。

## Decisions

### 1) Worker 作为可选库原语，默认关闭
- 决策：新增 mailbox worker loop API，但默认不开启（需显式配置或显式注入）。
- 原因：保持 0.x 阶段行为稳定，避免默认引入额外并发消费副作用。
- 备选：默认开启 worker。拒绝原因：会改变既有运行面，回归风险高。

### 2) handler 错误默认 requeue
- 决策：worker handler 返回错误时默认 `requeue`，并按 mailbox 既有 retry/DLQ 语义收敛。
- 原因：与 at-least-once + idempotent convergence 主线一致，且更利于短暂错误自愈。
- 备选：默认 dead_letter 或 fail_fast。拒绝原因：过于激进，容易放大临时故障损失。

### 3) poll interval 默认 100ms
- 决策：`mailbox.worker.poll_interval=100ms`，并要求 `>0`。
- 原因：在响应性与空轮询开销之间平衡，适合作为通用默认值。
- 备选：10ms/1s。拒绝原因：分别带来高空转与响应滞后问题。

### 4) Lifecycle diagnostics 全量覆盖核心节点
- 决策：统一记录 consume/ack/nack/requeue/dead_letter/expired 事件到 mailbox diagnostics。
- 原因：只有全链路可观测，才能支持运行排障和 reason taxonomy 统计。
- 备选：仅记录失败路径。拒绝原因：缺少完整基线，难以比较成功/失败比例。

### 5) Reason taxonomy 冻结并 gate 阻断
- 决策：冻结 mailbox lifecycle reason taxonomy 最小集合，并在 shared gate 检测漂移。
- 原因：避免字符串自由扩散导致统计与契约不可回归。
- 备选：仅文档约定。拒绝原因：无法形成可执行治理。

## Risks / Trade-offs

- [Risk] worker 引入额外并发行为，可能影响现有测试假设  
  -> Mitigation: 默认关闭 + 显式启用；新增 worker 专项契约测试矩阵。

- [Risk] 默认 requeue 可能放大重复投递  
  -> Mitigation: 依赖既有 MaxAttempts + DLQ + reason_code 统计，并要求 handler 幂等。

- [Risk] taxonomy 过严影响扩展  
  -> Mitigation: 采用“最小冻结集合 + additive 扩展窗口”，新增 reason 必须伴随 spec 与 gate 更新。

## Migration Plan

1. 新增 mailbox worker API 与配置结构（默认关闭）。
2. 接入 runtime/config 解析、校验、热更新回滚。
3. 在 worker loop 与 mailbox lifecycle 操作中写入统一 diagnostics。
4. 增加 lifecycle reason taxonomy 常量集合与验证逻辑。
5. 新增 integration/contract suites，并接入 shared gate 与 quality gate。
6. 同步 README、runtime-config-diagnostics、mainline index、roadmap 文档。

回滚策略：
- 关闭 `mailbox.worker.enabled` 即可快速回退到无 worker 运行模式。
- 新增诊断字段保持 additive，不回滚历史记录结构。

## Open Questions

无阻塞项，按推荐值执行：
- 启用策略：默认关闭
- handler 错误策略：默认 requeue
- poll interval 默认值：100ms
- reason taxonomy：强约束并纳入阻断 gate
