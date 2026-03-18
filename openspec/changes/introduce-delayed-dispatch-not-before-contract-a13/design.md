## Context

A10 完成后 scheduler 已具备 QoS、公平性、DLQ、退避治理；A12 补齐异步回报闭环，但“定时延后执行”仍缺少一等契约。当前 `next_eligible_at` 仅服务于 retry transition，不能表达业务方“任务在 T 时间后再执行”的需求。

为保持 `library-first` 定位，A13 采用最小增量方案：引入 `not_before` 作为任务级调度属性，通过 scheduler claim 判定和持久化恢复补齐业务延后语义，不引入平台化 cron/控制面。

## Goals / Non-Goals

**Goals:**
- 新增任务级 `not_before` 并提供可测试的 delayed claim 语义。
- 确保 delayed 语义与既有 QoS/公平性/backoff 可组合且无语义冲突。
- 确保 memory/file 恢复后 `not_before` 不漂移、不提前执行。
- 通过 composer 暴露延后 child dispatch 能力。
- 补齐 timeline/diagnostics/gate 的契约与回归覆盖。

**Non-Goals:**
- 不实现 cron/周期调度。
- 不引入外部调度系统（MQ/定时服务）。
- 不改变 A12 异步回报主语义。
- 不改变现有 retry backoff 算法。

## Decisions

### 1) 任务字段采用 `not_before`（绝对时间）
- 方案：在 `scheduler.Task` 增加可选 `not_before` 时间戳。
- 原因：语义清晰，和首次 claim 关系直接，便于序列化与恢复。
- 备选：增加相对延迟字段 `delay_ms`。拒绝原因：恢复与跨进程场景更易出现时间基准歧义。

### 2) 可领取判定采用双门槛
- 方案：claim 条件同时满足：
  - 队列状态可领取；
  - `not_before` 为空或 `not_before <= now`；
  - 现有 `next_eligible_at` 条件满足（如存在）。
- 原因：确保业务延后与重试退避均被尊重。
- 备选：`not_before` 与 `next_eligible_at` 取其一。拒绝原因：会破坏重试治理或业务延后语义。

### 3) `not_before` 仅约束首次可领取，不覆盖重试退避
- 方案：首次 claim 由 `not_before` 约束；失败后重试继续使用 backoff 推导的 `next_eligible_at`。
- 原因：职责边界清晰，避免与 A10 重复。
- 备选：重试时继续叠加原 `not_before`。拒绝原因：语义复杂，收益低。

### 4) 默认行为保持不变
- 方案：`not_before` 为空即按现有行为立即入队可领取。
- 原因：最大化向后兼容。
- 备选：全局默认延迟窗口。拒绝原因：会改变所有任务现状时序。

### 5) 诊断与 timeline 采用 additive 扩展
- 方案：新增 delayed 相关 reason 与 summary 字段，旧字段语义不变。
- 原因：兼容既有消费者。
- 备选：替换现有 queue/requeue 字段。拒绝原因：破坏兼容窗口。

## Risks / Trade-offs

- [Risk] 系统时钟偏差导致领取时间抖动  
  → Mitigation: 统一以 scheduler `now()` 判定，并在文档声明时钟语义。

- [Risk] delayed 与 QoS/fairness 交互导致排序预期偏差  
  → Mitigation: 明确“先到期，再进入 QoS 选择”并补合同测试。

- [Risk] file backend 恢复后时间解析误差  
  → Mitigation: 使用统一时间序列化格式并增加恢复回归测试。

- [Risk] 运维误将 retry backoff 当作业务定时  
  → Mitigation: 文档明确两者边界并添加诊断字段区分。

## Migration Plan

1. 扩展 `scheduler.Task` 与 store 序列化模型，加入 `not_before`。  
2. 更新 claim 判定逻辑并保持现有 queue/QoS/backoff 行为兼容。  
3. 在 composer child dispatch 增加 `not_before` 透传。  
4. 扩展 timeline/diagnostics 的 delayed 字段。  
5. 补 integration/contract tests 并接入 shared gate。  
6. 更新 README 与 runtime diagnostics 文档。  

回滚策略：
- 将新增 `not_before` 字段留空并关闭调用侧使用，系统回退到现有即时领取行为；
- additive 字段保留，不影响旧消费者。

## Open Questions

- 当前无阻塞问题；A13 首版不引入 cron 或相对延迟 DSL。
