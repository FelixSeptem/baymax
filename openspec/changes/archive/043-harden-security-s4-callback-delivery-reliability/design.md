## Context

S3 已完成安全事件 taxonomy 与 deny-only callback 告警契约，但当前 callback 仍在 runner 主路径同步执行，且缺少投递可靠性治理（超时、重试、队列背压、熔断）。在高并发或下游不稳定场景下，主流程延迟与告警可达性都会出现明显退化。

约束（已确认）：
- 默认投递模式为 `async`；
- 队列溢出策略为 `drop_old`；
- 重试次数固定为 3 次；
- 断路器采用 Hystrix 风格状态机；
- 保持 deny-only 触发规则；
- `event_id` 策略不变。

## Goals / Non-Goals

**Goals:**
- 引入 S4 callback 投递可靠性治理：异步队列、超时、重试、断路器。
- 通过 runtime config 提供可校验、可热更新、可回滚的投递治理配置。
- 扩展诊断字段，确保告警投递链路可观测、可排障。
- 保持 Run/Stream 在等价输入下的安全事件与告警语义等价。
- 新增独立 CI 门禁，防止投递治理语义回归。

**Non-Goals:**
- 不新增外部告警 sink（webhook/slack/email）。
- 不修改 deny-only 触发策略。
- 不引入跨进程持久化消息队列或分布式投递协议。
- 不调整 `event_id` 生成策略。

## Decisions

### Decision 1: 默认 `async` + 有界内存队列
- Choice: 默认将 deny 告警事件异步投递到内存有界队列，worker 后台消费。
- Rationale: 将 callback 的不确定耗时从主执行路径隔离，降低对 Run/Stream 尾延迟的影响。
- Alternative considered: 默认 `sync`。
- Rejected because: 仍会把 callback 延迟/抖动直接暴露给主流程。

### Decision 2: 队列溢出采用 `drop_old`
- Choice: 队列满时丢弃最旧待发送事件，保留最新高风险态势。
- Rationale: 安全告警更关注当前态势，`drop_old` 在告警风暴时可提高“新鲜告警”可达性。
- Alternative considered: `drop_new`。
- Rejected because: 可能导致最新 deny 事件在高压阶段持续丢弃。

### Decision 3: 固定 3 次重试 + 退避
- Choice: callback 失败后最多重试 3 次，采用指数退避与轻量抖动。
- Rationale: 在短时波动场景提高可达性，同时限制重试风暴。
- Alternative considered: 无限重试或 1 次快速失败。
- Rejected because: 前者不可控，后者可达性不足。

### Decision 4: Hystrix 风格断路器
- Choice: 引入 `closed/open/half_open` 状态机；`open` 期间快速失败，`sleep window` 到期后进入 `half_open` 试探恢复。
- Rationale: 对持续失败下游进行快速隔离，减少主系统资源消耗与级联故障。
- Alternative considered: 无熔断，仅重试。
- Rejected because: 持续失败时仍会造成无效重试与资源放大。

### Decision 5: 语义不变优先
- Choice: 告警投递结果只影响观测字段，不影响安全决策（deny 仍 deny）。
- Rationale: 安全策略执行与告警投递职责解耦，避免因下游告警系统故障改变阻断行为。
- Alternative considered: 告警失败时升级执行策略。
- Rejected because: 会造成不可预期的业务行为漂移。

### Decision 6: 独立门禁 `security-delivery-gate`
- Choice: 在 CI 中单独增加 S4 契约门禁，覆盖 async/drop_old/retry/circuit/Run-Stream 等价场景。
- Rationale: 将告警投递语义从通用测试中分离，提升可见性与治理强度。
- Alternative considered: 合并到 `test-and-lint`。
- Rejected because: 观测粒度与分支保护配置能力不足。

## Risks / Trade-offs

- [Risk] `async` 模式下进程崩溃可能导致内存队列中事件丢失。
  - Mitigation: 明确非持久化语义，提供 `queue_drop` 与 `dispatch_status` 诊断字段用于审计与容量调优。

- [Risk] `drop_old` 可能丢弃某些早期关键事件。
  - Mitigation: 增加 drop 计数、按 reason/policy 维度聚合，配合队列容量与限流阈值调参。

- [Risk] 熔断阈值配置不当造成误熔断或恢复过慢。
  - Mitigation: 提供 fail-fast 校验与热更新回滚，默认阈值采用保守基线并在门禁测试中固化。

- [Risk] 异步 worker 与主流程产生 Run/Stream 语义不一致。
  - Mitigation: 契约测试强制验证等价输入下 `severity/dispatch_status/circuit_state` 语义一致。

## Migration Plan

1. 在 `runtime/config` 增加 `security.security_event.delivery.*` schema、默认值、校验逻辑与热更新回滚覆盖。
2. 在 `core/runner` 引入 delivery executor（queue + worker + retry + circuit breaker）。
3. 将现有 S3 callback 调用接入 delivery executor，保持 deny-only 触发判定不变。
4. 在 `runtime/diagnostics` 与 `observability/event` 增加 S4 诊断字段映射（additive）。
5. 增加 S4 契约测试与 `security-delivery-gate` CI job/脚本。
6. 更新 runtime 文档，补充配置示例、状态机语义、调参建议。

## Open Questions

- 当前无阻断级开放问题；外部 sink 扩展与持久化投递留作后续里程碑。
