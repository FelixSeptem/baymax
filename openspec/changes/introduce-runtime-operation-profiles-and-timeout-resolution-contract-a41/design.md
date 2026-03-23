## Context

当前 runtime 在多个域分别维护 timeout 控制：`teams.task_timeout`、`workflow.default_step_timeout`、`a2a.client_timeout`、`scheduler.async_await.report_timeout`、`mailbox.worker.inflight_timeout` 等。各域配置本身已具备 fail-fast 与热更新回滚语义，但在一次多代理调用跨越 composer/scheduler/mailbox/a2a 时，缺少“统一解析与来源追踪”层。

这会造成三个工程问题：
- 调用链难以解释“最终生效 timeout 从哪里来”；
- 父子任务预算可能出现上层短、下层长的隐式冲突；
- diagnostics 能看到局部字段，难以直接复盘跨层预算收敛过程。

本设计在不引入平台化控制面的前提下，补齐库级 operation profile + timeout resolution 契约。

## Goals / Non-Goals

**Goals:**
- 提供库级 operation profile（`interactive|background|batch|legacy`）并形成稳定的 timeout baseline。
- 固化跨域 timeout 解析顺序：`profile baseline -> domain override -> request override`。
- 固化父子预算收敛规则：子任务生效 timeout 不得超过父任务剩余预算（`min` 收敛）。
- 增加可观测字段，明确 profile、生效预算来源与解析轨迹，支持 QueryRuns/Task Board 排障。
- 将上述契约纳入 shared gate/quality gate，覆盖 Run/Stream、memory/file、replay idempotency。

**Non-Goals:**
- 不改变既有 A31/A32 async-await 终态仲裁语义（`first_terminal_wins` 等保持不变）。
- 不引入外部 MQ、任务控制平台或多租户控制面能力。
- 不做智能自适应 timeout 学习或在线调优算法。

## Decisions

### Decision 1: 引入 `runtime.operation_profiles.*`，默认保持 `legacy`

- 方案：新增 profile 配置域，默认 profile 为 `legacy`，等价当前行为。
- 原因：减少迁移扰动，先建立统一解析能力与可观测性，再逐步切换 profile。
- 备选：
  - 直接默认切到 `interactive`：迁移风险高，容易在现网触发隐式行为变化。
  - 不设 profile，仅靠 request override：无法形成跨团队一致基线。

### Decision 2: 采用三层解析模型并禁止隐式兜底漂移

- 方案：解析顺序固定为
  1. profile baseline
  2. domain override（teams/workflow/a2a/scheduler/mailbox）
  3. request override
- 规则：冲突或非法组合直接 fail-fast；不得静默回退到不透明默认值。
- 原因：排序稳定、可测试、可回放，便于 gate 阻断 drift。
- 备选：
  - 按调用路径动态优先级：语义不稳定，难以在 replay 下保证一致。

### Decision 3: 父子预算采用 `min(parent_remaining, child_resolved)` 收敛

- 方案：在 composer/scheduler spawn 路径上统一执行预算夹紧（clamp）。
- 原因：保证子任务不会绕过父任务时间边界，减少 orphan/late 竞争窗口。
- 备选：
  - 子任务独立预算：实现简单但会放大跨层 timeout 不一致问题。

### Decision 4: 诊断字段采用 additive 且限制高基数字段

- 方案：新增字段
  - `effective_operation_profile`
  - `timeout_resolution_source`
  - `timeout_resolution_trace`
- 要求：遵循 `additive + nullable + default`，并对 trace 的字段集合做有界约束，避免高基数失控。
- 原因：兼容旧消费者，同时提升排障可解释性。
- 备选：
  - 输出完整原始配置快照：信息过量且可能引入敏感字段暴露风险。

### Decision 5: readiness 联动采用“策略映射”而非硬依赖

- 方案：当检测到 timeout 解析冲突时，输出 readiness finding；`strict=true` 时升级阻断，`strict=false` 时降级可运行并可观测。
- 原因：与 A40 的 readiness 分级一致，但不强耦合实现模块边界。
- 备选：
  - 在 timeout 解析层直接决定 blocked：会与 readiness 分层职责重叠。

## Risks / Trade-offs

- [Risk] 解析规则引入后，旧调用路径可能暴露历史“隐式超时”问题  
  -> Mitigation: 默认 `legacy`；先开观测字段，再分批切 profile。

- [Risk] `timeout_resolution_trace` 字段导致诊断 cardinality 增长  
  -> Mitigation: 固定 key 集合与枚举值，限制自由文本与动态标签。

- [Risk] 父子预算夹紧使某些长任务更早超时，触发回归  
  -> Mitigation: 增加 contract matrix（长任务、延后任务、awaiting_report）并提供明确回滚点。

- [Risk] A39/A40 实施中导致短期文档状态口径漂移  
  -> Mitigation: 在本变更中同步更新 roadmap/mainline index，并通过 docs-consistency gate 阻断。

## Migration Plan

1. 增量引入配置结构与解析器，实现 `legacy` 默认等价路径。
2. 在 composer/scheduler 接线 timeout resolver，保持旧字段仍可生效。
3. 增加 diagnostics additive 字段与 query 透传，不移除旧字段。
4. 接入 shared gate/quality gate 后再开放 profile 切换建议。
5. 回滚策略：关闭 profile 入口或切回 `legacy`，并保留既有域级 timeout 字段。

## Open Questions

- `batch` profile 的默认 timeout 是否需要与后续 workflow DAG 复杂度分级联动（当前先固定静态默认值）。
- request override 的暴露面是否仅限 composer/scheduler API，或扩展到更多高级入口（当前建议先限定）。
