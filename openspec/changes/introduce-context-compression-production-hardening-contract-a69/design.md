## Context

A67-CTX 已定义 context 组织语义，但在长期运行和高频工具调用场景下，context 压缩仍存在生产治理缺口：

- semantic compaction 成功率和收益波动较大，故障分类与 fallback 不够可审计；
- swap-back 在部分路径仍偏“顺序读文件”，未形成可解释的检索策略；
- 冷存文件缺少系统性生命周期治理，容量与碎片风险不可控；
- crash/restart/replay 下缺少统一幂等断言，恢复路径可验证性不足。

A69 聚焦“生产可用合同稳定性”，不扩展新的 context 语义族，也不替代 A64 的语义不变性能优化职责。

## Goals / Non-Goals

**Goals:**

- 为 semantic + rule-based 压缩建立统一质量门槛、失败分类与确定性降级链路。
- 固化 `hot|warm|cold|pruned` 状态迁移与 swap-back 检索排序，保证可解释与可回放。
- 为 file 冷存建立 `retention/quota/cleanup/compact` 全生命周期治理与容量护栏。
- 补齐 crash/restart/replay 场景下的幂等一致性断言与去重语义。
- 将 A69 配置、诊断、replay、quality gate 接入统一阻断闭环。
- 为 a62 `context-governed` 示例提供稳定依赖面，降低示例漂移成本。

**Non-Goals:**

- 不新增 `reference-first/isolate/edit-gate/recap` 等 context 语义能力族。
- 不改写 A67-CTX 已归档语义定义。
- 不在 A69 内承担 A64 性能优化主责。
- 不引入外部托管冷存或第二状态事实源。

## Decisions

### Decision 1: A69 只做生产治理，不新增平行语义

- 方案：所有改动限定在“质量、稳定性、恢复性、可观测性、门禁”维度。
- 原因：避免 context 语义再次分叉，保持 A67-CTX 作为语义唯一源。
- 取舍：语义创新空间受限，但合同稳定性更高。

### Decision 2: 语义压缩采用“质量门槛 + 失败分级 + 确定性 fallback”

- 方案：semantic compaction 必须输出质量评估与失败分类，在 `best_effort` 与 `fail_fast` 下保持确定性行为。
- 方案：rule-based 压缩对象边界显式化，含“最早工具调用结果”类历史项裁剪时必须满足证据保留条件。
- 原因：把“能压缩”与“值得压缩”分离，避免不可解释收益。
- 取舍：实现复杂度提升，但可回归性显著提高。

### Decision 3: 冷热分层与回填统一到规则化状态机

- 方案：统一 `hot|warm|cold|pruned` 迁移约束，并要求 swap-back 使用“相关性优先 + 新近性次级”排序。
- 原因：顺序读取在长会话下召回不稳定，难以保障关键上下文回填质量。
- 取舍：需要补充排序元数据与诊断字段，但能提升恢复确定性。

### Decision 4: file 冷存采用“可限额、可回收、可压实”治理

- 方案：默认 file backend 保留，但必须具备 retention/quota/cleanup/compact 与异常中断恢复语义。
- 原因：避免 `context-spill.jsonl` 无界增长与碎片化。
- 取舍：I/O 路径复杂度提高，但运行风险可控。

### Decision 5: 一致性治理坚持“单事实源 + 幂等恢复”

- 方案：spill/swap-back/replay 只依赖统一状态事实源；重启后不得出现重复回填、重复计数或状态撕裂。
- 原因：多状态源是恢复漂移和 replay 不可复现的主要风险来源。
- 取舍：对状态边界约束更严格，但回放稳定性更高。

### Decision 6: 配置与诊断字段坚持 additive + nullable + default

- 方案：A69 新字段必须满足向后兼容原则，保持 `env > file > default`、非法更新 fail-fast 与原子回滚。
- 原因：防止历史消费者因字段扩展被破坏。
- 取舍：字段设计需更克制，但兼容性风险更低。

### Decision 7: A69 replay 采用专属 fixture + 漂移分类

- 方案：新增 A69 fixture contract，并定义 compaction/tiering/swap-back/cold-store/recovery drift taxonomy。
- 原因：仅靠单测难覆盖跨重启与跨模式语义漂移。
- 取舍：维护 fixture 成本增加，但 CI 回归能力增强。

### Decision 8: A69 成为 a62 context-governed 子项的完成前置

- 方案：a62 非 context 示例可并行推进；`context-governed` 相关任务完成判定必须依赖 A69 门禁与回放收敛。
- 原因：示例层不应反向定义 runtime 语义，必须建立在稳定合同之上。
- 取舍：a62 局部任务节奏受 A69 影响，但整体交付风险更低。

## Risks / Trade-offs

- [Risk] A69 门禁接线增加 CI 耗时。  
  -> Mitigation: 使用影响面映射执行 required suites，保留 `fast/full` 分层但不跳过必选项。

- [Risk] 冷存治理参数过多导致配置误用。  
  -> Mitigation: 提供安全默认值、范围校验与热更新回滚，文档同步示例。

- [Risk] semantic 与 rule-based 组合导致诊断复杂。  
  -> Mitigation: 固化失败分类与 fallback 阶段字段，统一 replay drift taxonomy。

- [Risk] a62 依赖 A69 后出现交付节奏耦合。  
  -> Mitigation: 明确“非 context 并行、context 后置收口”策略，避免全量阻塞。

## Migration Plan

1. 建立 A69 基线清单与影响面映射（context/diagnostics/gates/docs）。
2. 先落地 S1/S2（语义压缩与冷热分层）并补齐单测与 replay fixture。
3. 再落地 S3/S4（冷存治理与恢复一致性）并补齐 crash/restart 回归。
4. 落地 S5/S6（配置诊断字段 + 强门禁接线），同步 docs/index。
5. 执行严格门禁与回放回归后，再放行 a62 context-governed 子项验收。

## Rollback Plan

- 功能回滚：关闭 A69 新增治理开关后，退回既有路径，不改变默认语义。
- 配置回滚：非法或不兼容热更新必须 fail-fast，并原子回滚到前一有效快照。
- 门禁回滚：仅允许临时降级非 required 子步骤，且必须记录恢复计划与截止时间。
