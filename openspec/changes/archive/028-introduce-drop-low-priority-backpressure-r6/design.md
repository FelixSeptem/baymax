## Context

当前 `runtime-concurrency-control` 在 R5 阶段已经收敛默认 `block` 语义、取消传播与最小并发诊断字段。现阶段需要在保持默认行为稳定与兼容的前提下，引入可选的降载策略，用于高 fanout 队列拥塞场景下控制尾延迟与资源占用。

边界约束（已确认）：
- 新策略命名：`drop_low_priority`。
- 优先级判定仅通过配置规则，不新增请求参数显式优先级。
- 首期范围仅 `tool/local` dispatch。
- 单轮全部被 drop 时立即 fail-fast。
- timeline reason 固定为 `backpressure.drop_low_priority`。
- 文档与实现统一更新。
- 验收包含相对提升百分比 + `p95` + `goroutine-peak`。

## Goals / Non-Goals

**Goals:**
- 在 `concurrency.backpressure` 增加 `drop_low_priority` 可选模式。
- 提供规则化优先级判定（tool/keyword）与“可被 drop 的优先级集合”配置。
- 保持 Run/Stream 语义一致与失败分类一致。
- 增加可观测字段和 timeline reason，便于分析降载行为。
- 增加契约测试与 benchmark 口径，支撑回归治理。

**Non-Goals:**
- 不扩展 `mcp` 与 `skill` 路径的 drop 行为。
- 不引入参数层显式优先级输入或外部策略引擎。
- 不改变默认背压模式（默认仍 `block`）。

## Decisions

1. 枚举扩展而非替换
- 决策：`concurrency.backpressure` 增加 `drop_low_priority`，保留现有 `block|reject`。
- 理由：兼容已有配置与行为基线，降低迁移风险。

2. 规则驱动优先级判定
- 决策：仅支持配置规则（`decision_by_tool` / `decision_by_keyword` 风格）判定 call 优先级。
- 理由：满足可控性与可审计性，避免请求面扩展和语义漂移。

3. 范围收敛到 tool/local
- 决策：首期仅在 local dispatcher 队列拥塞时执行低优先级 drop。
- 理由：影响面最小，便于验证收益并控制回归。

4. 全量 drop fail-fast
- 决策：若某轮 tool calls 全部被 drop，runner 立即 fail-fast 结束本轮运行。
- 理由：防止 silent degrade 与误导性“成功”结果。

5. 可观测口径统一
- 决策：timeline 使用 `backpressure.drop_low_priority`；诊断复用并扩展 `backpressure_drop_count` 等字段，不破坏现有契约。
- 理由：便于与 R5 指标体系对齐，减少观测口径分叉。

## Risks / Trade-offs

- [风险] 规则配置不当导致过度丢弃 → [缓解] 默认仍 `block`，并提供 fail-fast 与观测告警字段。
- [风险] drop 策略影响用户感知稳定性 → [缓解] 仅限 local 路径，契约测试覆盖 Run/Stream 等价语义。
- [风险] 指标优化不显著 → [缓解] 以相对百分比 + p95 + goroutine-peak 作为硬验收，未达标则回滚配置策略。
