## Context

CA2 external retriever 在 E1/E2 已具备统一 SPI、profile 默认值与基础观测能力，但仍存在两个工程缺口：
1. 缺少稳定的 capability hints 扩展口，难以在不侵入 assembler 主流程的情况下承载 provider 特有能力提示。
2. 缺少可复用的 template pack，外部接入仍依赖重复 mapping 配置，跨系统迁移成本高。

本次 E3 仅做“扩展面增强”，不做 provider 专用 adapter 落地，不改变 `fail_fast/best_effort` 或 runner 主状态机。

## Goals / Non-Goals

**Goals:**
- 在 CA2 Stage2 SPI + runtime config 层提供 capability hints 扩展口，保持 assembler 主流程不感知 provider 特有细节。
- 交付精简 template pack（`graphrag_like`、`ragflow_like`、`elasticsearch_like`）并定义确定性解析顺序。
- 保持 hints/template 失败或不匹配为观测信号，不触发自动策略动作。
- 在 diagnostics 增量扩展 hint/template 字段，保留既有错误分层语义（`transport|protocol|semantic`）并允许扩展枚举。
- 通过契约测试 + benchmark 基线保障 Run/Stream 语义等价与回归可观测。

**Non-Goals:**
- 不引入 provider 专用 adapter 执行逻辑。
- 不新增自动 provider 切换、自动路由调整或策略变更。
- 不新增 CLI 调试工具或 runnable examples。
- 不重构现有 Stage2 主状态机与失败策略模型。

## Decisions

### Decision 1: 扩展点落在 SPI 和 runtime config，不进入 assembler 主流程分支
- Choice: capability hints 作为 Stage2 SPI 的可选扩展字段，由 provider adapter 自主解释。
- Rationale: 避免 assembler 与 provider 特定语义耦合，保持 CA2 主流程稳定。
- Alternative considered: 在 assembler 路由层增加 provider-specific 分支逻辑。
- Rejected because: 扩散耦合面，后续 provider 增长时维护成本线性上升。

### Decision 2: template pack 仅收敛三种标准 profile
- Choice: E3 标准模板包仅包含 `graphrag_like`、`ragflow_like`、`elasticsearch_like`。
- Rationale: 聚焦真实接入高频面，降低首期维护与验证复杂度。
- Alternative considered: 同时维护更大模板集合（含 generic 变体）。
- Rejected because: 在无稳定使用数据前会放大治理与兼容风险。

### Decision 3: 模板解析采用“profile defaults -> explicit overrides”，并允许 explicit-only
- Choice: 先解析 profile 默认映射，再应用显式字段覆盖；当未选 profile 时允许显式配置独立运行。
- Rationale: 兼顾模板复用与细粒度控制，降低迁移门槛。
- Alternative considered: profile 与 explicit 二选一。
- Rejected because: 限制渐进迁移路径，无法覆盖“先模板后微调”的主流接入方式。

### Decision 4: hint mismatch 仅观测，不自动触发动作
- Choice: hints 不匹配仅记录诊断字段和事件信号，不改变 provider 选择与 stage policy。
- Rationale: 与 E2 观测优先策略一致，先稳定口径再考虑自动动作。
- Alternative considered: mismatch 时自动降级到默认 mapping 或自动切换 provider。
- Rejected because: 会引入隐式行为漂移，影响排障可预测性。

### Decision 5: 错误语义分层保持稳定，仅做增量字段扩展
- Choice: 保持 `transport|protocol|semantic` 基线语义不变，hint/template 相关信息以 additive 字段补充。
- Rationale: 保护现有消费方与告警规则，避免破坏兼容。
- Alternative considered: 重新定义 Stage2 错误分层模型。
- Rejected because: 超出 E3 范围且迁移成本高。

### Decision 6: Run/Stream 采用语义等价约束，不要求事件时序完全一致
- Choice: 对相同输入与配置，约束两条路径在模板选择、hint 结果和 Stage2 分类上语义等价。
- Rationale: 保障契约一致性，同时保留实现层并发差异空间。
- Alternative considered: 强制逐事件严格顺序一致。
- Rejected because: 对流式路径约束过强，收益低于实现成本。

## Risks / Trade-offs

- [Risk] hint/schema 扩展会增加配置理解成本。
  - Mitigation: 提供最小 YAML 样例和字段说明，保持字段命名与现有 CA2 风格一致。
- [Risk] 模板解析链增加运行时开销。
  - Mitigation: 通过基准测试新增解析 baseline，并与现有趋势 benchmark 一起回归。
- [Risk] hint mismatch 信号可能带来噪音。
  - Mitigation: 统一 reason code 与最小字段集，保持“仅观测”避免连锁动作。
- [Risk] Run/Stream 实现分叉导致语义漂移。
  - Mitigation: 增加等价契约测试覆盖成功、降级、mismatch 三条路径。

## Migration Plan

1. 在 runtime config 中引入 hint/template_pack 字段并补齐启动与热更新校验。
2. 在 Stage2 SPI 请求上下文新增可选 hint 扩展结构，provider 适配层按需消费。
3. 引入模板解析器并固定 precedence（profile defaults -> explicit overrides -> explicit-only）。
4. 在 diagnostics/event 增量添加 hint/template 字段，保持既有字段语义不变。
5. 补齐契约测试和 benchmark baseline，验证 Run/Stream 语义等价与解析开销。
6. 同步文档（runtime config、roadmap、acceptance）并补充三种模板 YAML 示例。

## Open Questions

- None for current E3 scope.
