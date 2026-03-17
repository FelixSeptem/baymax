## Context

当前 `skill/loader` 的语义触发路径仅使用 lexical weighted-keyword scorer。该实现简单稳定，但在同义改写、上下文表达差异较大时，对技能匹配的召回与精度存在天花板。代码中已存在 scorer 抽象与 TODO 扩展位，具备以增量方式引入 embedding scorer 的条件。

同时，项目已有 CA3 embedding 适配与 runtime 配置校验体系，可以复用现有设计经验：默认行为不变、扩展能力可选开启、异常路径 best-effort 回退、诊断字段增量扩展。

## Goals / Non-Goals

**Goals:**
- 在 skill trigger scoring 中引入可选 `lexical_plus_embedding` 策略。
- 使用线性加权融合 lexical 分数与 embedding 分数。
- embedding 不可用时回退 lexical，不中断 skill 触发流程（best-effort）。
- 通过 runtime JSON/YAML 配置提供 embedding 参数并纳入 startup/hot-reload 校验。
- 增加最小可观测字段，支持诊断和离线调优。
- 强制验证 Run/Stream 在等价输入与配置下的 skill 触发语义等价。

**Non-Goals:**
- 不改变默认策略（仍为 `lexical_weighted_keywords`）。
- 不引入新的 CLI 参数或独立命令入口。
- 不引入复杂多阶段重排或学习型在线策略。
- 不在本变更内扩展 skill-level 多模型路由治理。

## Decisions

### 1) 策略扩展为 `lexical_plus_embedding`，默认保持 lexical-only
- 方案：在现有 `skill.trigger_scoring.strategy` 上增加新枚举，默认值不变。
- 原因：保证向后兼容与风险可控，已有业务配置无需变更即可保持原行为。
- 备选：直接切换默认到 embedding 增强。拒绝原因：会引入行为漂移与回归风险。

### 2) 融合方式采用线性加权
- 方案：`final_score = lexical_weight * lexical_score + embedding_weight * embedding_score`。
- 原因：可解释、可调参、实现简单，适合当前 pre-1.x 快速迭代阶段。
- 备选：仅重排或门控阈值。拒绝原因：可观测性与调参体验不如线性模型直接。

### 3) embedding scorer 采用扩展接口 + 宿主注入
- 方案：在 `skill/loader` 暴露 embedding scorer 绑定入口；未绑定时策略自动回退 lexical。
- 原因：保持 library-first 与 provider 解耦，不把 skill loader 绑定到某个 SDK。
- 备选：内建 provider SDK 调用。拒绝原因：耦合度高，测试和维护成本上升。

### 4) 失败语义固定为 best-effort lexical fallback
- 方案：未注册 scorer、超时、调用错误、无效分数均回退 lexical，并输出标准化 fallback reason。
- 原因：优先保证可用性与稳定性，不因增强路径影响主流程。
- 备选：fail-fast。拒绝原因：与当前用户确认的策略冲突，且会放大可用性风险。

### 5) Run/Stream 强制语义等价
- 方案：在等价输入、等价配置、等价 scorer 行为下，断言触发技能集合和排序语义一致。
- 原因：现有契约已强调 Run/Stream 等价，避免模式切换出现隐式差异。
- 备选：仅校验 Run。拒绝原因：无法覆盖流式路径回归风险。

### 6) 运行时配置仅支持 JSON/YAML 路径
- 方案：新增 `skill.trigger_scoring.embedding.*` 配置，不新增 CLI 参数。
- 原因：符合当前配置体系和用户要求，减少接口面扩张。

## Risks / Trade-offs

- [Risk] embedding scorer 接入质量不一致导致触发波动  
  → Mitigation: 默认 lexical-only；embedding 可选开启；提供 fallback reason 与分数字段用于回放调优。

- [Risk] 线性权重设置不当导致精度下降  
  → Mitigation: 保留 lexical 默认路径，提供权重配置与契约测试基线。

- [Risk] 新增配置带来热更新错误风险  
  → Mitigation: startup/hot-reload 统一 fail-fast 校验，非法更新回滚旧快照。

- [Risk] Run/Stream 行为偏差  
  → Mitigation: 增加等价契约测试，覆盖成功与 fallback 场景。

## Migration Plan

1. 增量发布配置与策略枚举，默认仍为 lexical-only。  
2. 发布 embedding scorer 扩展接口与 best-effort fallback 语义。  
3. 发布诊断字段与 Run/Stream 等价契约测试。  
4. 文档更新（runtime config、acceptance、roadmap）。

回滚策略：
- 将 `skill.trigger_scoring.strategy` 切回 `lexical_weighted_keywords`；
- 或移除 embedding scorer 绑定，系统自动回退 lexical。

## Open Questions

- 当前里程碑无阻断级 open questions。
