## Context

D3 已经把 skill lexical 触发能力扩展到 `mixed_cjk_en` 并增加固定 top-k 预算（`max_semantic_candidates`），但固定预算无法同时覆盖“高置信头部”和“多候选接近”两种分布。当前需要在不破坏 D1/D2/D3 既有语义（默认 lexical、best-effort、Run/Stream 等价）的前提下，增加可解释且可调参的自适应预算机制。

## Goals / Non-Goals

**Goals:**
- 引入预算模式 `fixed|adaptive` 并将默认模式设为 `adaptive`。
- 在 adaptive 模式下支持 `min_k`、`max_k`、`min_score_margin` 三个核心参数，并保持确定性输出。
- 保持 explicit 命中旁路预算裁剪语义不变。
- 扩展最小诊断字段用于预算决策追踪。
- 配置继续通过 JSON/YAML + env 映射生效，支持 hot-reload fail-fast 与 rollback。

**Non-Goals:**
- 不修改 lexical/embedding 的打分公式与 fallback 语义。
- 不引入学习型在线策略或复杂 rerank pipeline。
- 不增加 CLI 入口或新的 provider SDK 依赖。

## Decisions

### 1) 预算模式双轨：`fixed` 与 `adaptive`，默认 `adaptive`
- 方案：保留 fixed 兼容路径，同时将默认预算模式切换为 adaptive。
- 原因：兼顾兼容性与收益，固定预算仍可用于稳定回放与场景对齐。
- 备选：仅保留 adaptive。拒绝原因：调试与回归基线不利。

### 2) 自适应预算采用“分差驱动”的确定性规则
- 方案：在“阈值过滤 + 稳定排序”后执行预算决策：
  - 在 `[min_k, max_k]` 范围内选取候选；
  - 使用 `min_score_margin` 判断是否需要扩展候选；
  - `top1-top2` 用于快速判断头部置信度并写入诊断字段。
- 原因：可解释、易调参、结果稳定。
- 备选：基于 query 长度或启发式权重。拒绝原因：跨语料稳定性较差，难以形成统一契约。

### 3) 观测字段 additive 扩展
- 方案：新增 `budget_mode`、`selected_semantic_count`、`score_margin_top1_top2`、`budget_decision_reason`，保留既有 `tokenizer_mode` 与 `candidate_pruned_count`。
- 原因：便于定位预算行为，不破坏历史消费者。

### 4) 配置治理沿用 fail-fast + rollback
- 方案：对 mode、k 范围、margin 范围做强校验；非法热更新拒绝并回滚。
- 原因：与现有 runtime 配置治理一致，避免隐式降级。

## Risks / Trade-offs

- [Risk] 默认切到 adaptive 可能引入触发分布变化  
  → Mitigation: 保留 fixed 模式可回退；通过诊断字段支持回放分析。

- [Risk] margin 参数设置不当导致过度收缩或过度扩展  
  → Mitigation: 默认 `min_score_margin=0.08`，并提供 `min_k/max_k` 保护边界。

- [Risk] 热更新期间配置错误影响线上稳定性  
  → Mitigation: 启动/热更新统一 fail-fast，非法配置不激活并回滚旧快照。

- [Risk] Run/Stream 预算行为出现分歧  
  → Mitigation: 强制契约测试覆盖 adaptive/fixed 路径与边界分支。

## Migration Plan

1. 增量发布预算模式与参数配置（默认 adaptive，仍支持 fixed）。  
2. 发布预算决策与诊断字段扩展。  
3. 补齐 Run/Stream 等价与配置回滚回归测试。  
4. 更新 runtime config / acceptance / roadmap 文档口径。

回滚策略：
- 将 `skill.trigger_scoring.budget.mode` 切回 `fixed`；
- 维持 `max_semantic_candidates` 作为固定预算上限路径。

## Open Questions

- 当前范围内无阻断级 open questions。
