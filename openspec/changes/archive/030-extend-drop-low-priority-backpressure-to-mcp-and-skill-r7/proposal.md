## Why

当前 `drop_low_priority` 背压仅覆盖 `tool/local` 路径，导致同一配置在 `mcp` 与 `skill` 路径下语义不一致，外部使用者难以建立稳定预期。需要将该模式扩展到 `local + mcp + skill`，并保持 Run/Stream、timeline、diagnostics 的统一行为。

## What Changes

- 将 `concurrency.backpressure=drop_low_priority` 从 local-only 扩展为覆盖 `local + mcp + skill` 三路径。
- 沿用现有优先级规则（`priority_by_tool`、`priority_by_keyword`、`droppable_priorities`），本期不新增判定维度。
- 统一终止语义：任一路径在同一轮“全量 drop”时立即 fail-fast。
- 统一可观测语义：保持 timeline reason `backpressure.drop_low_priority`，并在 diagnostics 增加按来源分桶计数（`local/mcp/skill`）。
- 增补契约测试与 benchmark，覆盖三路径 Run/Stream 语义一致性与 p95 指标。
- 默认策略保持不变：`backpressure=block`。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `runtime-backpressure-drop-low-priority`: 扩展适用域至 `mcp/skill`，并新增全量 drop fail-fast 的跨路径一致性要求。
- `runtime-concurrency-control`: 收敛三路径背压行为与默认策略边界（默认仍为 `block`）。
- `runtime-config-and-diagnostics-api`: 增加 drop 计数分桶字段并对齐配置/诊断口径。
- `action-timeline-events`: 明确 `backpressure.drop_low_priority` 在三路径下的一致发射约束。

## Impact

- Affected code:
  - `tool/local/*`
  - `mcp/*` 调度与执行路径
  - `skill/*` 调度路径
  - `core/runner/*`
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `integration/*` benchmark + contract tests
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/v1-acceptance.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- Compatibility:
  - 默认 `backpressure=block` 保持不变。
  - 仅当显式配置 `drop_low_priority` 时生效扩展语义。
