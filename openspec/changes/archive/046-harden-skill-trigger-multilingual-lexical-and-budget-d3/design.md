## Context

当前 skill trigger scoring 默认策略为 `lexical_weighted_keywords`，但词法分词实现主要针对英文 token，中文与中英混合输入在默认路径下存在命中不足。虽然 D2 已引入 `lexical_plus_embedding` 增强，但 embedding 是可选能力，默认路径仍需具备稳定的多语言可用性。

同时，当前语义候选列表缺少明确预算约束，可能导致低价值候选进入 compile，带来提示词噪音与行为不稳定。项目已有 runtime config 热更新与诊断字段增量扩展机制，适合在保持兼容的前提下增量增强。

## Goals / Non-Goals

**Goals:**
- 为默认 lexical 路径增加 `mixed_cjk_en` 分词能力，覆盖中文和中英混合输入。
- 引入语义候选预算配置 `max_semantic_candidates`，默认 `top_k=3`。
- 为 skill 触发诊断新增 `tokenizer_mode`、`candidate_pruned_count`。
- 保持默认策略与已有外部 API 不变，继续强制 Run/Stream 语义等价门禁。
- 配置入口保持 JSON/YAML + env 映射，不新增 CLI 参数。

**Non-Goals:**
- 不引入 emoji/symbol 专项词法规则。
- 不修改 `lexical_plus_embedding` 的融合公式与 fallback 语义。
- 不在本案引入复杂 rerank 或在线学习策略。
- 不扩展新的 provider 绑定或额外模型依赖。

## Decisions

### 1) 分词模式采用 `mixed_cjk_en`，并保留原 lexical 框架
- 方案：新增可配置 `tokenizer_mode`，默认 `mixed_cjk_en`。在现有 weighted-keyword 评分框架中，仅替换 token 生成逻辑。
- 原因：最小化变更域，保留现有阈值、权重、tie-break 与回退策略。
- 备选：新增独立中文 scorer。拒绝原因：会增加策略分叉与配置复杂度。

### 2) 候选预算采用 `top_k`，默认 `max_semantic_candidates=3`
- 方案：在“阈值过滤 + 稳定排序”之后，对 semantic candidates 执行 top-k 截断；显式触发（explicit）不受该预算限制。
- 原因：行为可解释、结果稳定，能直接抑制长尾噪音候选。
- 备选：基于 score gap 的动态裁剪。拒绝原因：调参敏感且跨场景可预测性较弱。

### 3) 诊断字段采用 additive 扩展
- 方案：在 skill 触发相关事件中新增 `tokenizer_mode` 与 `candidate_pruned_count`，不改动既有字段语义。
- 原因：便于离线诊断和策略调优，同时保持兼容。
- 备选：只在 debug 日志输出。拒绝原因：无法稳定进入 diagnostics API 契约。

### 4) 配置治理沿用现有 fail-fast + rollback 机制
- 方案：为 `tokenizer_mode` 与 `max_semantic_candidates` 增加 startup/hot-reload 校验；非法配置拒绝激活并回滚到上一有效快照。
- 原因：与现有 runtime/config 语义一致，降低运行时风险。
- 备选：非法值回退默认值。拒绝原因：会隐藏配置错误，削弱可运维性。

### 5) Run/Stream 继续强制语义等价
- 方案：新增契约测试覆盖中文输入与预算裁剪路径，断言等价输入下 skill 选择集合与顺序语义一致。
- 原因：防止仅在流式路径发生分词/裁剪行为偏差。

## Risks / Trade-offs

- [Risk] `mixed_cjk_en` 分词规则过粗导致误召回  
  → Mitigation: 先保持 deterministic 简化规则，配合 `confidence_threshold + top_k` 双重收敛。

- [Risk] `top_k=3` 在部分长查询场景召回不足  
  → Mitigation: 提供可配置上限并输出 `candidate_pruned_count`，支持回放调优。

- [Risk] 新增配置引入热更新回归  
  → Mitigation: 复用 runtime config fail-fast 校验与 manager rollback 测试模式。

- [Risk] Run/Stream 行为差异  
  → Mitigation: 新增中文/混合输入 + budget 裁剪等价契约测试。

## Migration Plan

1. 增量引入配置项并设默认：`tokenizer_mode=mixed_cjk_en`、`max_semantic_candidates=3`。  
2. 在 loader 打通多语言分词与语义候选裁剪逻辑。  
3. 打通 diagnostics 字段映射与契约测试。  
4. 更新 runtime config / acceptance / roadmap 文档。

回滚策略：
- 将变更回退到旧版本实现，或在紧急情况下通过热更新临时提升 `max_semantic_candidates` 以放宽裁剪影响。

## Open Questions

- 当前范围内无阻断级 open questions。
