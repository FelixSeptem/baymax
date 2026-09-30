## Why

归档 146 已经能够测量已准入工具集合的 schema pressure 与 synthetic selection quality，但它刻意不回答“压力是否稳定到足以进入运行时按需投影设计”。当前宿主只能看到一次性快照，缺少窗口化、可重放的 readiness 证据；同时主线 Roadmap/README 仍把已归档的场景模拟提案标为进行中，导致状态门禁失败。现在需要先建立一个离线、证据驱动的 readiness contract，避免未经稳定触发信号就改动 Tool/MCP/Provider 运行时语义。

## What Changes

- 新增版本化 `tool_schema_projection_readiness.v1` 能力，消费仅包含已准入工具和 canonical schema facts 的 bounded reference-only 快照。
- 增加稳定窗口判定：区分一次性峰值与跨样本重复出现的工具数量、schema token/bytes、预算占用和选择质量信号。
- 增加 projection opportunity 的 advisory 评估，要求候选集合始终是既有准入集合的子集，并验证必要 schema/能力语义不会丢失。
- 输出独立的 pressure、opportunity、selection quality、admission integrity、semantic completeness、Run/Stream parity 与 replay determinism verdict，以及 `not_ready`、`ready_for_runtime_design`、`blocked` 结论。
- 复用 `tool/schemaaudit`、`tool/diagnosticsreplay`、`tool/contributioncheck` 和既有 Skill/MCP/manifest/allowlist/sandbox owner；新增 fixture、replay、contract test 与双平台 gate。
- 修正 Roadmap、README 和相关一致性文档中已归档场景模拟提案的状态与主线基线。
- 明确不新增 runtime selector、动态 schema projection、ModelRequest/provider adapter 改动、registry/marketplace/dynamic download、credential store、RuntimeRecorder 字段或第二套准入/终态状态机。

## Capabilities

### New Capabilities

- `tool-schema-projection-readiness`: 对已准入工具集合进行 bounded、provider-neutral、离线可回放的按需 schema projection readiness 评估；仅产生进入后续运行时设计的证据，不改变当前执行语义。

### Modified Capabilities

- 无。现有 `tool-schema-pressure-selection-audit` 的 pressure/selection contract 不变；本变更只消费其 canonical facts 并增加独立 readiness 能力。

## Impact

- 代码与测试：`tool/schemaaudit`、`tool/diagnosticsreplay`、`tool/contributioncheck`，以及新的 fixture 和 contract gate。
- 文档：`docs/development-roadmap.md`、`README.md`、`docs/mainline-contract-test-index.md`，同步归档状态和 readiness contract 映射。
- 兼容性：新增版本化、nullable/default 兼容的离线结果，不修改运行时配置、诊断写入、Tool/MCP 生命周期、provider 请求或准入行为。
- 风险与回滚：错误阈值或样本判定可能造成 readiness 误报；可通过 fixture 调整、版本化 evaluator 回滚或移除新增 gate 回滚，且无需数据迁移。
- Example Impact Assessment：`无需示例变更（附理由）`。本变更只增加离线审计与治理证据，不改变 `examples/agent-modes` 的运行时行为、配置语义或用户可见流程；真正接入 runtime projection 时必须另起提案重新评估示例影响。
