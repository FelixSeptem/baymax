## Context

仓库已覆盖多 provider、CA1-CA4、HITL 与治理门禁，但 DX 侧仍存在两个实操缺口：
- 核心包（尤其 `core/runtime/context/skill`）对外 API 参考不够系统，接入成本依赖经验传播；
- 诊断数据已有沉淀能力，但缺少“可复盘、可回归”的最小 replay 视图，排障时需要手工拼接事件线索。

本提案聚焦 D1 级别的最小闭环：补齐 API 参考覆盖 + diagnostics JSON replay，保持库接口优先、CLI 可选辅助，不改 runtime 主执行语义。

## Goals / Non-Goals

**Goals:**
- 建立 `core/runtime/context/skill` 的 API 参考覆盖基线与示例入口。
- 提供 diagnostics replay 最小能力：JSON 输入、精简视图输出、稳定错误码。
- 补齐 replay 契约测试并纳入 CI 阻断门禁（required check 候选）。
- 保持中文优先文档口径并接受英文内容。

**Non-Goals:**
- 不引入可视化平台或重型调试产品。
- 不实现运行中数据直连回放（本期仅 JSON 输入）。
- 不修改 runner/model/context 主流程行为。

## Decisions

### Decision 1: Replay 输入仅支持 JSON（本期）
- 方案：回放入口统一读取 diagnostics 导出 JSON。
- 原因：实现边界清晰、可离线复现、适合契约测试固化。
- 备选：直接对接运行时 API 拉取；本期否决，避免耦合在线环境与权限链路。

### Decision 2: 输出采用精简视图
- 方案：仅输出 `phase/status/reason/timestamp` 与最小关联 ID（如 `run_id`、`sequence`）。
- 原因：优先满足“快速定位”场景，避免一次性引入复杂渲染语义。
- 备选：富视图/多层聚合；本期否决，复杂度与回归面过高。

### Decision 3: 库接口优先，CLI 仅最小入口
- 方案：复盘能力核心放在可复用库包；可选提供最小命令行入口用于本地调试。
- 原因：保持项目 `library-first` 定位，并利于测试与集成复用。
- 备选：仅 CLI；本期否决，复用性与可测性不足。

### Decision 4: 文档覆盖范围明确包含 skill
- 方案：D1 覆盖范围在原 `core/runtime/context` 基础上扩展到 `skill/*`。
- 原因：skill 触发和加载是接入方常见困惑点，需纳入统一文档治理。

### Decision 5: CI 以独立 replay gate job 形式阻断
- 方案：新增 `diagnostics-replay-gate`（命名可在实施时微调），作为 required check 候选。
- 原因：保持质量门禁可观察、可配置、与其他检查解耦。

## Risks / Trade-offs

- [Risk] 精简视图信息量不足以覆盖复杂排障场景
  -> Mitigation: 保留稳定扩展接口，后续增量引入 detail/verbose 模式。

- [Risk] JSON 合同变更可能导致 replay 解析漂移
  -> Mitigation: 增加固定样本契约测试，输出错误码与失败原因保持稳定。

- [Risk] API 文档覆盖要求提升维护负担
  -> Mitigation: 通过最小示例模板与 CI 检查控制新增负担，优先关键包而非全仓库。

## Migration Plan

1. 设计并实现 replay 核心解析与精简输出库接口（可选命令行包装）。
2. 增加 replay 契约测试样本（通过/失败路径）。
3. 更新 `README` 与相关 docs 的 API 参考入口和排障指引。
4. 将 replay 校验纳入 CI，设定 required check 候选。
5. 在一个完整 PR 流程中验证 gate 阻断与放行行为。

回滚策略：
- 若 replay gate 造成误阻断，可先临时降级该 job，不影响 runtime 主链路；
- 文档调整与回放工具可独立回滚，不影响核心执行能力。

## Open Questions

- 本期无阻塞性开放问题；job 最终命名以实施 PR 中的 CI 约束为准。
