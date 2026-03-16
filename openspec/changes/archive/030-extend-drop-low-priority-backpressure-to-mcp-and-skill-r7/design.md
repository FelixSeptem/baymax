## Context

R6 已在 `tool/local` 完成 `drop_low_priority`：规则分类、可丢弃集合、全量 drop fail-fast、timeline reason 与基础 benchmark。当前缺口是 `mcp` 与 `skill` 路径未执行同语义，导致运行时行为依赖调用来源，破坏配置一致性与可观测一致性。

本提案基于已上线规则扩展适用域，不引入新配置维度。

## Goals / Non-Goals

**Goals:**
- 将 `drop_low_priority` 扩展到 `local + mcp + skill` 三路径。
- 保持 Run/Stream、错误分类、终止行为一致。
- 增加 diagnostics 分桶计数（local/mcp/skill）。
- 完整补齐契约测试与 p95 benchmark。

**Non-Goals:**
- 不改默认背压策略（仍 `block`）。
- 不新增优先级规则维度（例如参数规则、动态评分）。
- 不做分布式调度器或跨进程队列改造。

## Decisions

### Decision 1: 规则复用，不扩展配置模型
- 方案：沿用 `priority_by_tool`、`priority_by_keyword`、`droppable_priorities`。
- 原因：降低迁移成本并避免配置膨胀。
- 备选：为 mcp/skill 增设独立规则；否决，短期会产生多套语义。

### Decision 2: timeline reason 保持单一
- 方案：三路径统一使用 `backpressure.drop_low_priority`。
- 原因：下游消费方无需新增 reason 分支。
- 备选：新增 phase-specific reason；否决，增加兼容面。

### Decision 3: 分桶计数落在 diagnostics
- 方案：在现有并发诊断中补充 `backpressure_drop_count_by_phase`（`local/mcp/skill`）。
- 原因：满足可观测细分且不改变既有总量字段语义。
- 备选：仅保留总量；否决，不满足排障颗粒度。

### Decision 4: 全量 drop fail-fast 跨路径统一
- 方案：同一轮工具决策若在当前路径全量 drop，则立即终止。
- 原因：与 R6 语义一致并降低不可见退化风险。
- 备选：mcp/skill 继续执行后续流程；否决，语义分裂。

## Risks / Trade-offs

- [Risk] mcp/skill 路径引入 drop 后可能改变既有吞吐分布
  -> Mitigation: 增加 benchmark（含 p95）与回归阈值。

- [Risk] 分桶字段新增导致 diagnostics 消费方解析差异
  -> Mitigation: 保留旧总量字段并将分桶字段作为增量可选读取。

- [Risk] Run/Stream 三路径一致性测试复杂度上升
  -> Mitigation: 用主干契约索引登记最小集合并长期门禁。

## Migration Plan

1. 扩展调度路径：将 drop_low_priority 判定接入 mcp/skill。
2. 对齐 runner 终止逻辑：全量 drop fail-fast 统一触发。
3. 增加 diagnostics 分桶字段并保持兼容总量字段。
4. 更新 timeline 与错误分类契约测试（Run/Stream）。
5. 补 benchmark（mcp/skill 场景 + p95）。
6. 同步 README/docs/roadmap/contract index。

回滚策略：
- 若出现不可接受回归，可通过配置回退到默认 `block` 语义。
- 必要时可临时限制 drop 生效域，但需同步文档声明。

## Open Questions

- 分桶字段命名是否采用 map 结构（`by_phase`）还是平铺字段（`_local/_mcp/_skill`）；实现阶段按现有 diagnostics 风格收敛。
