## Context

当前仓库已在 H1 阶段交付 Action Timeline 结构化事件（默认启用、Run/Stream 语义一致、`context_assembler` 独立 phase）。文档中已明确 TODO：H1 不落地 diagnostics 聚合字段。H1.5 目标是在不改 runner 主状态机、不引入 CLI 的前提下，将 timeline 可观测性收敛到现有 diagnostics 主路径。

## Goals / Non-Goals

**Goals:**
- 在 `runtime/diagnostics` 提供 run 级 timeline phase 聚合字段。
- 支持 `p95` 延迟统计（phase 级）。
- 强化重放幂等：重复 timeline 事件不重复累计。
- 保障 Run/Stream 同场景 phase 分布等价。
- 保持默认启用，不引入新开关。

**Non-Goals:**
- 不实现 HITL Action Gate。
- 不实现 pause/resume 状态机。
- 不增加新存储后端或独立观测数据库。
- 不改现有非 timeline 事件契约。

## Decisions

### Decision 1: 聚合写入落在现有 `runtime/diagnostics` run record
- 方案：直接扩展 run record，避免平行存储。
- 理由：符合现有 single-writer + idempotency 架构，迁移成本最低。
- 备选：新增 timeline 专用存储。
  - 放弃原因：引入额外一致性与查询复杂度。

### Decision 2: p95 基于单 run、单 phase 内采样窗口计算
- 方案：在 run 完成时汇总 phase duration 样本，计算 p95 并写入 run record。
- 理由：满足可观测需求且不引入跨 run 状态依赖。
- 备选：仅记录平均值。
  - 放弃原因：无法识别长尾延迟。

### Decision 3: 幂等锚点使用 timeline `sequence + phase + status`
- 方案：聚合器对同 run 内重复 timeline 事件进行去重，重放不重复计数。
- 理由：sequence 已是 H1 稳定序语义，适合做最小幂等键。
- 备选：仅按事件时间去重。
  - 放弃原因：时间戳不稳定，重放易漂移。

### Decision 4: Run/Stream 一致性按“phase 状态分布等价”验收
- 方案：不要求逐事件完全一致，要求聚合统计等价。
- 理由：Run/Stream 细节事件序列天然不同，但业务可观测语义应一致。
- 备选：严格逐事件一致。
  - 放弃原因：约束过强且不必要。

## Risks / Trade-offs

- [风险] 事件到聚合映射出错导致统计偏差  
  - Mitigation: 增加契约测试（成功/失败/取消/跳过）+ 回放幂等测试。

- [风险] p95 算法实现不稳定  
  - Mitigation: 固化 percentile 计算规则并加入边界用例（空样本/单样本/重复样本）。

- [风险] 字段扩展引发下游解析兼容问题  
  - Mitigation: 新字段均为可选新增，保持原字段不变并更新文档映射。

## Migration Plan

1. 扩展 run diagnostics 模型与 recorder 聚合结构。
2. 接入 timeline 事件到聚合器，并实现重放幂等。
3. 增加 Run/Stream 等价与 p95 计算测试。
4. 更新 README/runtime-config-diagnostics/roadmap。
5. 通过质量门禁后合并。

回滚策略：
- 如聚合逻辑异常，可仅回退聚合写入路径；timeline 事件发射保持不变。

## Open Questions

- phase 聚合字段最终在 API 层采用平铺字段还是结构化 map（本提案先按结构化 map 实现，保持可扩展）。
