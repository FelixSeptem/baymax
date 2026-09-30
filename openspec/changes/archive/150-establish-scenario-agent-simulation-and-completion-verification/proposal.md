## Why

现有仓库已经分别具备 fake model/tool、审批、取消、Stream、恢复、completion safe-point、evaluation corpus 和 diagnostics replay 的契约测试，但跨这些边界的多步骤场景仍由各 integration suite 以重复 setup 拼接，缺少统一的场景身份、运行结果和证据完整性语义。这样会让“运行已结束”“业务结果成立”“证据足以判定”和“允许发布”之间的区别依赖测试代码约定，难以稳定复现审批等待、工具副作用、Stream 截断和恢复组合故障。

现在启动是因为能力资产溯源审计已经归档，roadmap 的下一阶段明确要求审计场景化 Simulation/Verifier 的 harness 缺口；本提案可以先以离线、确定性、fixture/replay 优先的方式收口证据，不改变生产运行时语义。

## What Changes

- 新增版本化、reference-first 的 Scenario profile，描述受控用户、模型、工具、审批、取消、Stream 故障、恢复和预期证据边界。
- 新增 bounded Run Result 投影，分离运行状态、证据完整性、业务结果引用和发布准入四种结论；证据不足时输出 `indeterminate`，不得默认为成功或失败。
- 新增测试专用、非生产的可组合 harness/builder 设计，复用现有 `integration/fakes`、Run/Stream 入口、completion safe-point 和 diagnostics recorder，支持确定性 fake model/SSE、tool output、approval、cancel、truncation 与 recovery 编排。
- 新增 verifier 与 replay contract，覆盖正向、负向、边界、Run/Stream parity、幂等恢复、证据缺失和副作用引用越界场景。
- 新增版本化 fixture、诊断 replay 分类和 Shell/PowerShell parity gate；所有输出保持 bounded、reference-only、可重放且无 live provider/network 依赖。
- 更新主线 contract test index、harness 架构文档和 roadmap 状态映射，说明该能力只服务测试与审计，不构成生产 executor 或业务 Outcome owner。
- 不新增 runtime 配置、hosted state、registry、动态下载、第二套终态状态机或生产级 Simulation API。

## Capabilities

### New Capabilities

- `scenario-agent-simulation-and-completion-verification`: 定义版本化场景、受控事件编排、bounded Run Result、证据完整性、完成验证、`indeterminate` 分类、replay 和 Run/Stream 对等测试支持合同。

### Modified Capabilities

- 无。现有 evaluation、diagnostics replay、completion safe-point 和 RuntimeRecorder 要求保持不变；本 change 通过引用既有 owner 组合它们，不重定义同名语义。

## Impact

- 代码范围：`integration/fakes`、测试支持包、`runtime/evalcontract` 或其引用适配层、`tool/diagnosticsreplay` 的离线验证入口；仅允许测试/审计路径依赖新合同。
- 测试范围：新增 deterministic fixture/replay、负向 drift 分类、Run/Stream parity、审批/取消/截断/恢复组合案例和 gate 接线。
- 文档范围：`docs/mainline-contract-test-index.md`、`docs/runtime-harness-architecture.md`、`docs/development-roadmap.md`，必要时补充模块边界说明。
- 依赖边界：不改变 `runtime/*`、`context/*`、`model/<provider>` 的既有依赖约束，不调用 Provider SDK、Git、workspace mutation 或网络服务。
- 配置与诊断：不新增 runtime 配置键；若产生诊断投影，必须 additive、nullable、default，并通过 `observability/event.RuntimeRecorder` 单写入口。
- 回滚：删除新 capability 的 builder、fixture、replay、gate、文档和 spec delta 即可回滚，无持久化迁移；既有 integration tests 和业务 Outcome 语义保持可用。

## Example Impact Assessment

无需示例变更（附理由）

本 change 只增加离线测试支持、fixture、replay 和 verifier contract，不修改 `examples/agent-modes` 的 runtime path、配置语义、expected markers 或业务 Outcome，因此不需要新增或修改示例。若后续实现需要改变示例行为，必须先完成 `MATRIX.md` 与对应模式 README 的文档基线，并将评估改为 `修改示例`。

## Why now / Risks / Verification

- **Why now**：能力资产溯源方向已归档，roadmap 将场景化 Simulation/Verifier 列为下一项观察候选；现有分散 harness 已能证明单点边界，但缺少组合场景的统一证据结果。
- **主要风险**：测试 harness 可能意外演化为生产执行器；Scenario 与业务 Outcome 可能被混淆；Run/Stream 组合回放可能引入第二套事件或终态语义。
- **控制措施**：builder 仅存在于测试支持路径；Run Result 只保存 bounded 引用；verifier 只判定测试证据，不写入 runtime owner；所有组合场景复用现有 Run/Stream、completion、RuntimeRecorder 和 replay owner。
- **验证命令**：`openspec validate --all`、受影响 Go contract/replay 测试、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1` 及对应 Shell gate。
