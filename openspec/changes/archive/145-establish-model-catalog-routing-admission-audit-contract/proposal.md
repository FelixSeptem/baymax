## Why

Baymax 已具备 host-injected provider/model catalog、能力协商、credential evidence、fallback admission、readiness projection 和原子 reload，但当前 roadmap 仍无法证明“多个候选模型之间的 deterministic selection”是否是实际缺口。若直接增加本地路由或 discovery，容易冻结未经宿主需求验证的优先级语义，并越过 remote catalog、credential store 和全局 router 的边界。

本 change 以可回放的 catalog/routing admission 审计为先：统一候选集、能力、credential、fallback、readiness 与 Run/Stream parity 的事实表达；只有 fixture/replay 明确证明现有 exact-identity admission 无法表达宿主需求时，才在 `model/catalog` 增加最小、纯函数、宿主提供候选集的 resolver。

## What Changes

- 建立版本化、有界、provider-neutral 的 model catalog routing/admission audit fixture，覆盖候选集归一化、确定性顺序、能力匹配、credential evidence、fallback、readiness 和 Run/Stream parity。
- 复用现有 `model/catalog`、`adapter/capability`、`runtime/config`、readiness/admission 与 `RuntimeRecorder` 所有权，禁止创建平行 catalog、credential、readiness 或 terminal 状态机。
- 增加稳定的 audit/replay drift 分类，区分 catalog descriptor、candidate selection、capability、credential、fallback、readiness、catalog generation 和 Run/Stream parity 缺口。
- 审计 fixture 已证明多个 independently admissible 候选无法由 exact-identity admission 表达明确选择需求，因此增加 opt-in 的 host-supplied candidate resolver；resolver 不执行发现、刷新、credential probe 或 provider 调用。
- 为 resolver（若被触发）定义 deterministic tie-break、fail-fast 冲突和 blocked/degraded admission 语义，并与现有 fallback/strict readiness policy 对齐。
- 扩展 diagnostics replay、contribution boundary、shell/PowerShell gate 以及 model/runtime 文档，验证有界性、隐私、历史 fixture 兼容和 gate 对等。

## Example Impact Assessment

无需示例变更（附理由）：本 change 首阶段只增加 offline audit、fixture、replay 和 gate；不改变 `examples/agent-modes` 的 runtime path、配置键或 expected markers。若条件化 resolver 后续改变示例可观察选择或输出，必须先更新 `MATRIX.md` 和受影响模式 README，并将声明改为“修改示例”。

## Capabilities

### New Capabilities

- `model-catalog-routing-admission-audit`: 定义 host-supplied model catalog 候选集的归一化、审计、replay 和条件化 deterministic selection 边界。

### Modified Capabilities

无。现有 `provider-model-capability-and-credential-preflight`、`runtime-readiness-preflight-contract` 和 `adapter-capability-negotiation-and-fallback` 的要求保持不变；新 capability 只定义跨 owner 的审计、回放和条件化 resolver 事实，并复用这些既有合同。

## Impact

- 可能受影响的代码：`model/catalog`、`adapter/capability` 的 provider-neutral 事实接口、`runtime/config` catalog snapshot 接缝、`runtime/config` readiness/admission projection、`runtime/diagnostics` additive fields、`tool/diagnosticsreplay`、`tool/contributioncheck`。
- 可能新增的离线资源：版本化 catalog routing fixture、replay tests、shell/PowerShell provider/model catalog gate。
- 不新增 provider、remote discovery、background refresh、credential store、credential probe、全局 router、共享 provider wire protocol 或 `context/*` SDK 依赖。
- 不新增 runtime 配置键，除非审计阶段证明现有 host-supplied candidate input 无法表达明确需求，并在 design/spec 中先记录影响与回滚点。
- 代码、测试、文档和 spec delta 必须在同一 feature branch 内完成；归档前执行完整质量门禁，归档后再合并回最新 `master`。
