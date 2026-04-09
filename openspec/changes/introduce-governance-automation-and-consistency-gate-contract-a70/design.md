## Context

仓库已经形成“OpenSpec 驱动 + roadmap 对齐 + gate 阻断”的治理基线，但仍依赖较多人工核对：

- roadmap 状态字段与 OpenSpec 实际状态偶发不一致；
- 提案级 example 影响声明存在“有规则、缺自动化”的执行空档。

A70 聚焦治理自动化，不扩展 runtime 能力边界。

## Goals / Non-Goals

**Goals:**
- 固化 roadmap 与 OpenSpec 状态的自动一致性校验并阻断漂移。
- 固化提案 `example impact assessment` 的统一声明格式与自动校验。
- 保证 shell/PowerShell 校验路径等价，失败语义一致。
- 将治理输出做成可审计分类，便于 PR/CI 快速定位问题。

**Non-Goals:**
- 不修改任何 runtime、context、model、mcp 语义。
- 不引入平台化治理控制面或外部服务依赖。
- 不替代各能力提案本身的 contract/replay/perf 验证职责。

## Decisions

### Decision 1: 状态一致性采用“单事实源 + 双向校验”

- 方案：`openspec list --json` 与 `archive/INDEX.md` 作为事实源，对 `docs/development-roadmap.md` 做一致性检查。
- 原因：避免多个来源各自维护状态，降低人工对账误差。
- 备选：仅人工在 PR 评审时核对。
- 取舍：新增脚本维护成本，但显著降低治理漂移。

### Decision 2: 提案 example 影响声明采用固定枚举

- 方案：后续提案必须声明 `新增示例`、`修改示例` 或 `无需示例变更（附理由）`。
- 原因：统一口径便于静态检查与审计。
- 备选：允许自由文本声明。
- 取舍：灵活性下降，但可执行性更强。

### Decision 3: 治理校验接入 quality/docs 双门禁

- 方案：状态一致性与声明校验同时接入 `check-quality-gate.*` 与 `check-docs-consistency.*`。
- 原因：一个偏代码门禁、一个偏文档门禁，双接线能减少漏检。
- 备选：只接入 docs consistency。
- 取舍：执行耗时略增，但覆盖更完整。

### Decision 4: 失败输出必须带稳定分类码

- 方案：校验失败输出 machine-readable 分类（例如 `roadmap-status-drift`、`missing-example-impact-declaration`、`invalid-example-impact-value`）。
- 原因：便于 CI 汇总和后续自动化统计。
- 备选：仅输出人类可读文本。
- 取舍：脚本实现复杂度略增，但诊断效率更高。

## Risks / Trade-offs

- [Risk] 治理脚本规则过严导致误阻断。  
  -> Mitigation: 首轮按最小必需字段实施，保留清晰 reason code 与修复指引。

- [Risk] roadmap 格式演进导致解析脚本脆弱。  
  -> Mitigation: 采用稳定锚点与容错解析，并在变更时同步更新测试样例。

- [Risk] 双门禁接入增加 CI 时长。  
  -> Mitigation: 脚本保持轻量静态校验，避免重复全量执行重型测试。

## Migration Plan

1. 建立 A70 校验规则与样例基线（正反例）。
2. 实现状态一致性与 example-impact 声明校验脚本（shell/PowerShell）。
3. 将校验接入 `check-quality-gate.*` 与 `check-docs-consistency.*`。
4. 同步更新 roadmap、contract index、AGENTS 映射说明。
5. 执行校验并在 CI 暴露 required-check 候选。

## Rollback Plan

- 若规则导致大面积误阻断，可临时降级为 warning 模式并在同一提案内修正规则；
- 回滚仅影响治理脚本与文档，不影响 runtime 行为。
