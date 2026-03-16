## Context

F1 已交付 CA3 semantic compaction 的最小可用路径，但尚未形成“质量可判定、模板可约束、回退可追踪”的生产级控制面。F2 的目标是补齐这些控制能力，并保持既有 `fail_fast/best_effort` 语义边界与 Run/Stream 一致性。

当前仓库实现与文档已体现以下 F2 事实：
- semantic 路径有规则评分与阈值门控；
- 语义模板支持占位符白名单校验；
- embedding 仅保留 SPI hook；
- 诊断字段包含质量分、质量原因与回退原因；
- roadmap 已标记本变更完成。

> 注：本设计文档为归档异常后的恢复重建版本（Recovered），用于补全归档可追溯性。

## Goals / Non-Goals

**Goals:**
- 为 semantic compaction 建立确定性质量 gate（规则评分 + 阈值）。
- 为 semantic prompt 建立 runtime 模板白名单控制与 fail-fast 校验。
- 保持 `best_effort` 回退和 `fail_fast` 终止语义。
- 输出可运维诊断字段，支持质量与故障归因。
- 保持 Run/Stream 行为等价。

**Non-Goals:**
- 不在本期绑定 provider 级 embedding adapter。
- 不引入 vector store 或 retrieval 生命周期能力。
- 不改动 CA2 路由协议与外部 retriever 契约。

## Decisions

### Decision 1: 质量门控使用规则评分加权而非黑盒评分
- 方案：`coverage/compression/validity` 加权求分，与阈值比较决定是否通过。
- 理由：可解释、可调参、可回放，且与当前语义压缩实现耦合低。

### Decision 2: 质量失败复用既有 stage policy 语义
- `best_effort`：回退 `truncate` 并记录 `fallback_reason`。
- `fail_fast`：直接返回错误终止。
- 理由：不引入新的失败策略，降低行为认知成本。

### Decision 3: 模板配置采用“prompt + 占位符白名单”双重约束
- 方案：模板必须非空；占位符必须平衡且属于白名单。
- 理由：降低注入和错误模板带来的语义偏移风险。

### Decision 4: embedding 先保留 SPI hook，不绑定 adapter
- 方案：配置层暴露 `enabled/selector` 与校验；运行时未绑定 adapter 时保持规则评分路径并记录原因。
- 理由：为后续 E3 平滑扩展预留接口，同时不扩大本期风险面。

## Risks / Trade-offs

- [Risk] 规则评分权重配置不当导致误判
  - Mitigation: 启动/热更新参数校验 + benchmark 回归门禁。
- [Risk] 模板配置错误导致 semantic 输出异常
  - Mitigation: 启动/热更新 fail-fast 校验（空模板、占位符非法、白名单不匹配）。
- [Risk] embedding hook 开启但 adapter 未绑定产生认知偏差
  - Mitigation: 明确诊断 reason（如 `embedding_hook_not_bound`）和文档说明。

## Migration Plan

1. 扩展 CA3 compaction config：quality/template/embedding 子结构与默认值。
2. 在 semantic compactor 中接入质量评分、阈值判断和 reason 聚合。
3. 接入质量失败回退语义与 fallback reason。
4. 扩展 runner/diagnostics/event 的字段映射。
5. 增补 Run/Stream 契约测试与 benchmark 基线。
6. 同步 README 与 runtime diagnostics 文档。

## Open Questions

- E3 首个 provider adapter 采用哪条路径及默认模型（后续变更处理）。
- 质量阈值是否需要按 stage 或负载等级分层（后续可选优化）。
