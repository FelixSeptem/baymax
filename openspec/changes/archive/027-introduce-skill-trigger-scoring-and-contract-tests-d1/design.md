## Context

当前 `skill/loader` 已存在基于描述/trigger 的轻量词法匹配，但实现形态偏“内嵌算法 + 固定阈值”，缺少稳定配置入口、同分决策契约与低置信度安全默认。现阶段项目已完成多 provider、context assembler、HITL 与并发基线收敛，下一步需要提升 skill 触发链路的可解释性与可回归性，避免在生态扩展中出现触发漂移。

约束：
- 保持 `library-first`，不新增 CLI。
- 不新增对外公开 API；扩展点仅限 loader 内部/包内复用。
- Run/Stream 主语义不变；仅影响 skill 选择结果的确定性。
- embedding 暂不实现，但需预留接口与 TODO，避免后续破坏性改造。

## Goals / Non-Goals

**Goals:**
- 定义并落地 skill 触发评分契约：默认“关键词加权 + 阈值命中”。
- 定义稳定同分规则：默认 `highest-priority`。
- 默认开启低置信度抑制（低分不触发）。
- 将评分相关参数接入 runtime YAML，并遵循 `env > file > default` 与 fail-fast 校验。
- 增加契约测试：评分命中、同分决策、低置信度抑制、配置覆盖优先级、Run/Stream 语义稳定。

**Non-Goals:**
- 不在本期引入 embedding 检索/向量召回实现。
- 不新增或变更对外公开接口（包括 CLI 与外部 SDK surface）。
- 不改写 skill discover/compile 的主流程结构与诊断基础模型。

## Decisions

1. 评分策略采用可插拔接口 + 默认词法实现
- 决策：在 `skill/loader` 内部抽象 scorer 接口，默认实现为关键词加权评分器。
- 理由：先收敛稳定行为与测试基线；后续 embedding 仅需替换 scorer，不影响调用方。
- 备选：直接引入 embedding。放弃原因：依赖/成本/模型漂移较大，且当前需求可由词法策略满足。

2. 同分规则固定 `highest-priority`
- 决策：候选技能分值相同时，按配置优先级选择最高者。
- 理由：相较 first-registered，更适合运营侧显式控制风险和行为。
- 备选：first-registered。放弃原因：行为依赖注册顺序，跨环境不稳定。

3. 低置信度抑制默认开启
- 决策：评分未达阈值即不触发 skill，默认开启并可配置。
- 理由：优先保证安全与确定性，减少弱语义误触发。

4. 配置进入 runtime YAML
- 决策：新增 `skill.trigger_scoring.*`（策略、阈值、同分规则、开关、权重等）并走现有 Manager 管线。
- 理由：保持统一配置入口与热更新能力，避免 skill 自建配置旁路。

5. 仅内部接口，不新增公开 API
- 决策：scorer 接口和策略注册能力仅在 loader 包内/内部复用范围开放。
- 理由：降低 v1 表面面积，避免过早锁死外部契约。

6. 合同测试先行
- 决策：新增 loader 合同测试矩阵，覆盖默认策略、配置覆盖、边界与回归。
- 理由：触发逻辑对用户体验敏感，需避免隐式行为漂移。

## Risks / Trade-offs

- [风险] 词法评分对同义表达覆盖有限 → [缓解] 保留 scorer 扩展点与 embedding TODO，文档明确适用边界。
- [风险] 新配置项过多导致维护负担上升 → [缓解] 提供最小必需字段与严格校验，保持默认即可用。
- [风险] 优先级策略可能掩盖高相关但低优先级技能 → [缓解] 通过合同测试与示例文档说明 tie-break 语义，并允许后续策略扩展。
- [风险] 热更新下评分行为突变 → [缓解] 复用 runtime 原子切换与回滚机制，非法配置 fail-fast 拒绝生效。
