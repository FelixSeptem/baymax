## Context

CA1/CA2 已提供 staged assembly 与外部检索能力，但尚未建立系统化 memory pressure control。当前在长上下文、多轮检索、工具结果堆积场景下，缺少统一的分级响应与恢复机制，且对关键内容保护策略不完整。CA3 目标是在保持 fail-fast 与并发安全基线的前提下，落地可配置的压力治理与单进程恢复闭环。

约束：
- 保持 library-first，不新增 CLI。
- 不改 runner 主状态机（不引入 HITL pause/resume）。
- 继续复用 `runtime/diagnostics`，不新增独立诊断 API。
- spill/swap 仅文件实现，其他后端保留接口。

## Goals / Non-Goals

**Goals:**
- 建立分级压力响应策略（安全/舒适/警告/危险/紧急）并支持阈值配置。
- 实现百分比 + 绝对 token 双触发机制。
- 实现 batch squash/prune（规则驱动最小集），支持 `critical`/`immutable` 保护标记。
- 实现文件型 spill/swap + `origin_ref` 回填追溯。
- 保证单进程 cancel/retry/replay 状态一致。
- 输出最小 CA3 观测字段并加入契约测试。
- 保证 Run/Stream 在分级决策结果上语义一致。

**Non-Goals:**
- 不实现学习型评分器与复杂语义重排。
- 不实现 DB/对象存储型 spill backend。
- 不实现跨进程/跨实例恢复。
- 不新增 examples（仅保留 roadmap TODO）。

## Decisions

### Decision 1: 阈值采用双触发（百分比 + 绝对 token）
- 方案：满足任一阈值即触发对应分级策略；默认 token 阈值按主流大模型 context window 设偏大并可配置。
- 理由：兼容不同模型窗口与不同部署资源约束。
- 备选：仅百分比阈值。
  - 放弃原因：对超大窗口模型不敏感。

### Decision 2: 分级策略按保护模式渐进升级
- 方案：
  - 安全区：正常加载
  - 舒适区：限制新加载预算
  - 警告区：触发 squash
  - 危险区：触发 prune
  - 紧急区：spill/swap + 默认拒绝低优先级新加载
- 理由：策略可解释、可观测、可测试。
- 备选：单阈值直接 prune/spill。
  - 放弃原因：抖动大，体验不稳定。

### Decision 3: 规则驱动 squash/prune + 保护标记
- 方案：先用规则（关键词/访问频率/最近使用）评分；`critical`/`immutable` 强保护。
- 理由：实现复杂度可控，后续可替换评分器。
- 备选：学习型评分器直接上线。
  - 放弃原因：测试与可解释性成本过高。

### Decision 4: spill/swap 仅文件后端，接口先行
- 方案：本期文件后端落地，DB/对象存储提供接口但不实现。
- 理由：收敛范围、减少外部依赖。
- 备选：并行实现多后端。
  - 放弃原因：实现面过大，验证成本高。

### Decision 5: 观测字段继续落在 run diagnostics
- 方案：扩展 `runtime/diagnostics.RunRecord`，不新增 API。
- 理由：与当前 single-writer/idempotency 架构一致。
- 备选：独立 CA3 diagnostics API。
  - 放弃原因：接口面膨胀，维护成本高。

## Risks / Trade-offs

- [风险] 阈值策略过激导致过早降级  
  - Mitigation: 提供可配置阈值与保守默认值，压测校准。

- [风险] prune 误删业务关键上下文  
  - Mitigation: `critical`/`immutable` 强保护 + 审计记录。

- [风险] spill/swap 增加 IO 开销  
  - Mitigation: 批处理写入与按需回填，增加观测字段跟踪成本。

- [风险] Run/Stream 决策不一致  
  - Mitigation: 增加契约测试，按分级结果等价做强验收。

## Migration Plan

1. 扩展 runtime 配置 schema 与默认值（分级阈值、token 阈值、策略参数）。
2. 在 context assembler 接入分级决策与 batch squash/prune。
3. 接入文件型 spill/swap 与 `origin_ref` 回填。
4. 扩展 diagnostics 记录与 recorder 字段映射。
5. 增加 Run/Stream 等价测试、幂等/恢复测试与 race 测试。
6. 更新 README/docs 并跑文档一致性检查。

回滚策略：
- 若 CA3 策略影响稳定性，可回退到 CA2 路径并关闭 CA3 配置分支（保持现有 CA2 行为不变）。

## Open Questions

- 默认绝对 token 阈值按“模型窗口比例”还是“固定上限”表达（本提案阶段先支持固定值，比例化作为后续 TODO）。
