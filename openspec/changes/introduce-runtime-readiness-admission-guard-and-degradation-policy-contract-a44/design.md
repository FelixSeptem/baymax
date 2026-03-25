## Context

当前运行时已经具备：
- A40：`ReadinessPreflight()` 及 `ready|degraded|blocked` 分类；
- A43：adapter health 与 readiness finding 的融合路径（实施中）。

但编排执行主入口（composer Run/Stream）对 readiness 的使用仍以“透传 + 诊断写入”为主，缺少统一 admission 决策点。调用方如果希望“blocked 必须拒绝、degraded 策略化处理”，需要自行在外部拼装判断，导致语义分散且难以通过 contract gate 稳定回归。

## Goals / Non-Goals

**Goals:**
- 建立库级 readiness admission guard：在 managed Run/Stream 入口统一执行准入。
- 固化 admission 策略与默认值，支持 `blocked` fail-fast 与 `degraded` 策略化处理。
- 保证 admission 检查 read-only，不引入调度副作用（不改变 queue/claim/task state）。
- 为 admission 决策补齐 additive 诊断字段与回放幂等语义。
- 将 admission 套件纳入 quality gate 阻断路径。

**Non-Goals:**
- 不引入平台化准入策略中心或多租户 RBAC 控制面。
- 不改变 scheduler/task lifecycle 语义与终态分类。
- 不覆盖非 managed runtime 场景下的调用方自定义准入逻辑。

## Decisions

### Decision 1: admission 与 preflight 解耦，复用 preflight 结果

- 方案：admission 只消费 `ReadinessPreflight()` 结果并执行策略映射，不重复实现组件探测。
- 原因：避免重复逻辑与 taxonomy 漂移，保持 A40/A43 单一语义源。
- 备选：admission 内部重新探测组件。缺点是实现重复且结果可能不一致。

### Decision 2: admission 默认关闭，策略默认保守

- 方案默认值：
  - `runtime.readiness.admission.enabled=false`
  - `runtime.readiness.admission.mode=fail_fast`
  - `runtime.readiness.admission.block_on=blocked_only`
  - `runtime.readiness.admission.degraded_policy=allow_and_record`
- 原因：不破坏现有调用链；先可观测、再按需收紧。

### Decision 3: blocked 拒绝路径必须无副作用

- 方案：admission 在 enqueue/dispatch 前执行；拒绝时不生成 scheduler task、不写 mailbox、不改变 lease 计数。
- 原因：保证 fail-fast 语义与可回归验证（拒绝即零业务副作用）。

### Decision 4: diagnostics 使用 additive 汇总，不扩散高基数字段

- 方案：新增 admission 计数字段与策略/原因摘要，不写入高基数明细。
- 原因：控制查询成本，避免和 A42 的 query perf 治理冲突。

## Risks / Trade-offs

- [Risk] 开启 admission 后，历史“可运行但 degraded”路径可能被阻断  
  -> Mitigation: 默认 disabled + degraded 默认 allow_and_record，逐步启用。

- [Risk] admission 判断与调用方外部策略冲突  
  -> Mitigation: 提供可配置开关，未启用时不改变行为；启用后以 runtime 策略为准。

- [Risk] 额外 preflight 调用带来入口延迟  
  -> Mitigation: 复用现有 readiness 组件快照与既有探测结果，不增加网络依赖。

## Migration Plan

1. 新增 `runtime.readiness.admission.*` 配置模型、默认值与校验逻辑。
2. 在 runtime 层实现 admission 判定器（消费 preflight 结果并返回 admission 决策）。
3. 在 composer managed Run/Stream 入口接线 admission guard，确保拒绝路径无副作用。
4. 增加 admission diagnostics additive 字段与 replay idempotency 处理。
5. 增加 integration/contract 测试与 quality gate 映射。
6. 更新 README / runtime-config-diagnostics / roadmap / contract index。

## Open Questions

- 是否需要引入 `degraded` 的第三策略（例如 `shadow_deny`）用于灰度观测（当前不纳入本提案）。
- admission 结果是否需要暴露更细粒度枚举给 SDK 层（当前先保持运行时内部摘要字段）。
