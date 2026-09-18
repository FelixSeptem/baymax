## Why

Baymax 已能用 corpus、Badcase、experiment、review-only feedback 与 continuity comparison 描述和重放最终评测结果，但仍无法结构化定位轨迹第一次偏离的位置、责任 owner、当时允许或禁止的下一动作，以及结论所依据的有界证据。现在补齐这一缺口，可以作为已归档 Eval continuity contract 的收尾，并为后续 Provider、Budget 与 Memory application 质量审计提供统一、离线且可门禁的归因基础。

## What Changes

- 新增版本化、离线、reference-only 的首错归因与轨迹前缀决策边界 contract，表达 first-error step/kind、root-cause owner、primary/secondary cause、recoverability、confidence 和 bounded evidence refs。
- 定义轨迹前缀上的 acceptable actions、forbidden actions、required evidence 与 safety constraints，并要求确定性归一化、稳定摘要和有界集合。
- 将首错归因作为 Badcase/experiment 的 additive + nullable 关联；历史 corpus、Badcase、experiment 与 feedback payload 保持可读，既有聚合和 continuity semantics 不变。
- 将 evidence-linked recommendation 保持为 review-only：建议可以引用归因结果，但不得自动修改 prompt、Skill、tool、policy、memory、runtime 配置、代码、测试或 gate。
- 将 Memory retrieval/application 的命中、遗漏、误用和 scope error 作为既有 evaluation corpus 的场景，不新增 memory 事实源或独立评测控制面。
- 新增版本化 `eval_first_error_attribution.v1` fixture、离线 replay 分类、正向/负向/边界 contract test 和双平台 gate 接线；先以失败 fixture 证明结果级 Badcase 的缺口，再实现最小 contract。
- 不修改 runtime loop、provider/tool 执行、配置语义、终态、Run/Stream 决策路径或 `examples/agent-modes`。

## Capabilities

### New Capabilities

- `evaluation-first-error-attribution-and-trajectory-boundary`: 定义有界首错归因、轨迹前缀决策边界、证据引用、可恢复性与置信度的确定性离线评测 contract。

### Modified Capabilities

- `evaluation-corpus-badcase-and-experiment-contract`: 为 Badcase、experiment 和 review-only feedback 增加 additive + nullable 的首错归因关联，同时保持历史聚合、批准和非自动执行语义。
- `diagnostics-replay-tooling`: 增加 `eval_first_error_attribution.v1` fixture 的离线、确定性验证、canonical drift taxonomy 与历史 fixture 兼容性要求。

## Impact

- 主要影响 `runtime/evalcontract` 的新归因类型、归一化/比较函数和 contract tests，以及 `tool/diagnosticsreplay` 的 fixture evaluator、版本化 testdata 和 replay tests。
- 文档影响包括 `docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`，以及必要的 Eval/replay contract 映射说明。
- 复用现有 `Reference`、Badcase correlation、continuity reference、checkpoint/artifact/event identity 和 diagnostics replay；不读取或持久化 raw reasoning、完整 transcript、provider response、tool output body、memory body 或 workspace body。
- 不新增外部依赖、运行时配置、持久化迁移、Provider SDK 依赖、托管服务或平台控制面。

## Example Impact Assessment

无需示例变更（附理由）：本提案只增加离线 Eval contract、fixture、replay 与 gate，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或用户可见行为。
