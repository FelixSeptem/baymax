## Why

Harness² 提供了有价值的候选提出、同任务探测与组合比较方法，但它本身不是 Baymax runtime 或执行控制面。Baymax 已有可复现 Badcase 与实验比较契约，缺少把候选资产范围、固定实验条件、baseline 保留、隔离评分和成本证据作为一个可审查评测流程的要求；补齐它可以验证哪些改进确实来自行为证据，而不把 Harness² 引入生产执行路径。

## What Changes

- 为现有评测契约增加有界候选对比流程：比较不可变 baseline、同预算重复执行择优和 Propose–Probe–Compose 候选，并始终允许选择 baseline。
- 固定并绑定实验身份：Badcase/corpus 与 rubric 版本、模型、权限、数据、预算、网络策略、候选资产 identity/version/digest、write scope、结果证据和成本账本。
- 将选择证据与最终评分隔离；评分不可用或证据不足时输出 `unavailable` / `insufficient-evidence`，不得折算成零或自动采用候选。
- 把候选修改限制在显式列出的 prompt/Skill 等资产及大小/静态校验边界内；实验产物与经验仅供 review，不自动发布、写回长期经验库或修改运行行为。
- 把实验评测限定为只读、离线优先的既有评测 owner 扩展；发现新 owner gap 时另行提出增量 OpenSpec。

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `evaluation-corpus-badcase-and-experiment-contract`: 增加 baseline-preserving 的候选对比、固定且可追溯的实验身份、资产/网络/评分隔离、成本与证据要求，以及 review-only 结果边界。

## Impact

- 评测契约 owner：`runtime/evalcontract`；离线 fixture/replay owner：`tool/diagnosticsreplay`；门禁与证据索引：contract test scripts 和 `docs/mainline-contract-test-index.md`。
- 修改文档：`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`；同步评测/replay 文档与 fixture、门禁映射。
- 不改变 runtime 执行、scheduler、mailbox、checkpoint、native event history、provider、公开执行 API 或生产资源所有权；不引入 Harness² 依赖或新执行 owner。

## Example Impact Assessment

无需示例变更（附理由）：该变更新增离线评测契约，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或公开运行时行为。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/evaluation-contract.md`
- 维护 owner：evaluation contract maintainers、documentation and contract-test index maintainers
- 验证证据：`openspec validate evaluate-bounded-harness-candidate-comparisons --strict`、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | evaluation contract maintainers | 不增加 runtime owner、执行循环或模块依赖 |
| components | 修改文档 | `docs/development-roadmap.md` | roadmap owner | 研究结论与 active proposal 范围已同步 |
| configuration | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 记录离线实验配置绑定；不增加生产配置键、默认值或热更新语义 |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md` | contract-test index maintainers | 增加评测契约与 replay/gate 验证映射 |
| best practices | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 明确评测隔离、证据状态与 review-only 边界 |
| diagnostics | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 记录 replay-only 证据与 review-only 输出；不改变 RuntimeRecorder 写入 |
| examples | 修改文档 | `docs/evaluation-contract.md` | examples maintainers | 明确无示例变更及不应推断运行时恢复的边界 |
| CLI/integration | 无需文档变更（附理由） | — | integration maintainers | 不新增 CLI 或生产集成入口 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | Harness² 校准和 proposal 路由已记录 |
| release status | 修改文档 | `README.md` | release documentation maintainers | 同步 active milestone snapshot 与 OpenSpec 状态 |
