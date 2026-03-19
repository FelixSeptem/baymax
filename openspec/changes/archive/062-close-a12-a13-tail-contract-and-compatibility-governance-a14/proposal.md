## Why

A12（异步回报）与 A13（延后调度）把多代理通信能力补到可用状态，但目前收口风险仍集中在“契约冻结不完整”：reason taxonomy、诊断兼容窗口、shared gate 与主干索引存在漂移可能。A14 作为收尾治理提案，目标是在不扩功能的前提下，把 A12/A13 的代码、契约测试与文档口径一次收敛。

## What Changes

- 固化 A12/A13 共享契约冻结面：`a2a.async_*` 与 `scheduler.delayed_*` reason taxonomy、必需关联字段与 gate 约束。
- 增加跨模式合同矩阵：同步/异步/延后三种通信方式在 `Run/Stream`、`qos/recovery` 组合下的语义一致性与回放幂等性。
- 固化 A12/A13 additive 字段兼容窗口：`additive + nullable + default`，并补 legacy parser 语义约束。
- 扩展 shared multi-agent gate 与 contract index，确保 reason 规则、矩阵用例、文档映射同一变更内收敛。
- 更新 roadmap/runtime diagnostics/mainline 索引口径，明确 A12/A13 到 A14 的收口顺序与 DoD。

## Capabilities

### New Capabilities
- (none)

### Modified Capabilities
- `multi-agent-tail-governance`: 从 A5/A6 收口扩展到 A12/A13 收口，增加统一冻结与矩阵治理要求。
- `action-timeline-events`: 增加 A12/A13 合并后的 canonical reason completeness 与关联字段收敛约束。
- `runtime-config-and-diagnostics-api`: 增加 async/delayed additive 字段兼容窗口与 parser 语义要求。
- `go-quality-gate`: 增加 async+delayed 跨模式合同矩阵、taxonomy 漂移阻断与索引一致性检查。

## Impact

- 代码：
  - `tool/contributioncheck/*`
  - `scripts/check-multi-agent-shared-contract.sh`
  - `scripts/check-multi-agent-shared-contract.ps1`
  - `integration/*`（跨模式矩阵合同套件）
- 测试：
  - shared multi-agent contract gate 扩展
  - `Run/Stream` 语义等价与 replay-idempotent 组合场景扩展
- 文档：
  - `docs/mainline-contract-test-index.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 不引入平台能力与新外部依赖；
  - 不改变 A12/A13 业务语义，仅收敛契约、门禁与文档口径；
  - 新增字段与检查保持 `additive + nullable + default`。
