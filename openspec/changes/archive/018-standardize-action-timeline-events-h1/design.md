## Context

当前系统已有 runner 事件、diagnostics 持久化与 single-writer 基线，但用户侧对“动作执行路径”的消费仍依赖非稳定字段拼装，Run/Stream 间也缺少统一 phase/status 语义。下一阶段将引入 HITL/A2A/CA3，这些能力都需要稳定的执行时间线作为上层编排输入，因此先收敛 Action Timeline H1。

约束如下：
- 保持 library-first；不新增 CLI 依赖能力。
- 默认启用 timeline 输出。
- 不改变现有 diagnostics 聚合字段与既有对外语义。
- context assembler 需要作为独立 phase 暴露，便于后续 CA3/CA4 可观测扩展。

## Goals / Non-Goals

**Goals:**
- 定义统一 Action Timeline 结构化事件契约（phase/status/reason/order）。
- 统一 Run 与 Stream 路径的事件语义与状态枚举。
- 扩展 phase 集合，明确包含 `context_assembler`。
- 增加 `canceled` 状态枚举并纳入契约测试。
- 在 docs 中补充 observability 聚合 TODO 留痕，确保后续收敛路径明确。

**Non-Goals:**
- 不引入 runner pause/resume 状态机改造（H2/H3 范围）。
- 不新增 diagnostics 聚合字段与 dashboard 指标实现。
- 不实现 CLI 可视化或前端渲染层。

## Decisions

### Decision 1: 新增 timeline 事件契约，保持现有事件兼容
- 方案：新增结构化 Action Timeline 事件类型，附带统一字段（`run_id`, `phase`, `status`, `reason`, `timestamp`, `sequence`），不替换现有事件。
- 理由：避免破坏已有集成，允许调用方渐进迁移到新契约。
- 备选方案：直接重构既有事件 schema。
  - 放弃原因：变更面过大、兼容风险高。

### Decision 2: 状态枚举固定为 6 个并允许未来扩展
- 枚举：`pending|running|succeeded|failed|skipped|canceled`。
- 理由：覆盖当前运行态与可预见中断态，满足后续 HITL 的最小前置语义。
- 备选方案：本期只保留 5 个状态。
  - 放弃原因：后续引入用户中断时会产生语义破坏性扩展。

### Decision 3: 默认启用 timeline 事件，不新增开关
- 方案：H1 默认发射 timeline 事件；调用方不需要额外配置即可消费。
- 理由：减少接入分叉，避免“环境 A 有事件/环境 B 无事件”的排障复杂度。
- 备选方案：增加 runtime 配置开关。
  - 放弃原因：当前范围内无必要，且增加文档与测试复杂度。

### Decision 4: `context_assembler` 单独建模为 phase
- 方案：将 `context_assembler` 与 `model/tool/mcp/skill` 同级纳入 timeline。
- 理由：CA2/CA3 是后续重点演进域，独立 phase 能保证观测可追踪、可比较。
- 备选方案：将 assembler 合并到 model phase。
  - 放弃原因：会掩盖前置装配耗时与降级行为。

### Decision 5: 诊断聚合字段延期，但必须留 TODO 轨迹
- 方案：本期不新增 diagnostics 聚合字段；在 README/docs/roadmap 明确 TODO（后续 change 收敛 timeline 指标聚合）。
- 理由：控制 H1 范围，先稳定事件契约。
- 备选方案：本期同时落地聚合字段。
  - 放弃原因：会显著扩大改动面并延长验证周期。

## Risks / Trade-offs

- [风险] Timeline 与既有事件并行期间可能出现重复消费
  - Mitigation: 在文档中明确“timeline 为结构化主路径，旧事件为兼容路径”，并提供字段映射说明。

- [风险] Run/Stream 实现细节差异导致状态顺序不一致
  - Mitigation: 增加契约测试，覆盖成功/失败/跳过/取消路径，强约束序列一致性。

- [风险] 默认启用导致事件量上升
  - Mitigation: H1 仅新增最小必要字段；聚合与采样策略在后续 TODO change 中收敛。

## Migration Plan

1. 定义 timeline 事件 DTO 与状态/phase 枚举。
2. 在 runner + event recorder 接入 timeline 事件发射。
3. 增加 Run/Stream 契约测试，保证顺序和语义一致。
4. 更新 README/docs 与 roadmap，添加 diagnostics 聚合 TODO。
5. 通过质量门禁后合并。

回滚策略：
- 若 timeline 发射导致兼容问题，可仅回退 timeline 发射路径；既有事件契约保持不变。

## Open Questions

- 后续 timeline 聚合指标是否进入 `runtime/diagnostics` 主记录，或以独立诊断索引承载（TODO，后续提案决策）。
