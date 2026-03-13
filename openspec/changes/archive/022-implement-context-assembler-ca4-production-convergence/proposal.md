## Why

Context Assembler 已完成 CA1/CA2/CA3 功能闭环，但在生产可运维层面仍缺少 CA4 的策略收敛：阈值优先级、token 计数回退语义、可观测口径与性能门禁尚未统一到稳定发布标准。当前推进 CA4 可以在继续扩展 HITL/A2A 前先稳定上下文策略计算基线，降低后续联动风险。

## What Changes

- 收敛 CA4 阈值策略计算语义：明确全局阈值与 stage 阈值覆盖关系、冲突时的确定性选择、percent 与 absolute 双触发的优先级解释。
- 固化 token 计数与回退策略：
  - `sdk_preferred` 模式下优先 provider 计数；
  - provider 不可用或失败时固定回退本地估算；
  - 本地估算优先 `tiktoken`，失败再回退轻量字符估算；
  - 不因计数失败阻断主流程（fail-open for counting only）。
- 强化“阈值策略计算”契约测试：覆盖 Run/Stream 一致性、small delta 触发路径、stage override、fallback 行为。
- 纳入性能门禁：增加 CA4 相关 benchmark 与相对百分比阈值验收（含 P95）。
- 保持 spill backend 现状：`db/object` 继续仅保留接口与 TODO，不在本期实现。
- 同步 README 与 docs，确保 CA4 行为与实现一致。

## Capabilities

### New Capabilities
- `context-assembler-production-convergence`: 定义 CA4 生产收敛要求（阈值策略计算、token 计数回退、性能门禁与文档一致性）。

### Modified Capabilities
- `context-assembler-memory-pressure-control`: 增补 CA4 阶段对阈值计算优先级、Run/Stream 等价语义与恢复策略的规范要求。
- `runtime-config-and-diagnostics-api`: 明确 CA4 token 计数职责与回退语义，补充诊断字段语义一致性要求。
- `go-quality-gate`: 将 CA4 benchmark 相对百分比门禁纳入标准验证流程。

## Impact

- 代码范围：`context/assembler`、`core/runner`、`runtime/config`、`runtime/diagnostics`、`integration`（benchmark/contract tests）。
- 测试范围：新增 CA4 契约测试与 benchmark 回归检查，维持 `go test`/`race`/`lint`/`govulncheck` 门禁。
- 文档范围：`README.md`、`docs/context-assembler-phased-plan.md`、`docs/runtime-config-diagnostics.md`、`docs/development-roadmap.md` 及相关索引文档。
