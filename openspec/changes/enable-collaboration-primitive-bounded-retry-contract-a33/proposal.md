## Why

当前代码中 `composer.collab.retry.enabled` 已暴露配置位，但运行时仍强制禁止开启，导致协作原语在短暂传输抖动下只能直接失败，缺少受控自愈能力。  
在 A32 进入实施阶段后，下一步应优先收敛这类“有配置无能力”的主链路缺口，并保持 lib-first 与既有门禁语义一致。

## What Changes

- 将协作原语重试从“硬禁用”升级为“默认关闭、可显式开启的有界重试”。
- 新增 `composer.collab.retry.*` 治理字段：`max_attempts`、`backoff_initial`、`backoff_max`、`multiplier`、`jitter_ratio`、`retry_on`，并冻结推荐默认值。
- 固化重试分类：默认仅对传输层失败重试（`retry_on=transport_only`），协议/语义失败不重试。
- 固化重试范围：覆盖 `delegation sync` 与 `async submit` 阶段；不覆盖 `async accepted` 后的回报/对账收敛阶段。
- 固化重试所有权：scheduler 管理路径避免 primitive 层二次重试，防止双重重试叠加。
- 扩展协作原语重试诊断字段（additive）并要求 replay 幂等。
- shared multi-agent gate 纳入 collaboration retry contract suites 作为阻断项（Run/Stream 等价、replay 稳定、策略漂移阻断）。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `multi-agent-collaboration-primitives`: 将“retry 默认禁用”扩展为“默认禁用 + 显式开启时有界重试”，并新增范围与所有权语义。
- `runtime-config-and-diagnostics-api`: 扩展 `composer.collab.retry.*` 配置域与协作重试诊断聚合字段。
- `go-quality-gate`: 在 shared multi-agent gate 中纳入 collaboration retry contract suites 阻断项。

## Impact

- 代码：
  - `orchestration/collab/*`（重试执行策略、错误分类与范围边界）
  - `orchestration/invoke/*`、`orchestration/teams/*`、`orchestration/workflow/*`（sync/async submit 路径对齐）
  - `runtime/config/*`（重试配置解析、默认值、校验、热更新回滚）
  - `runtime/diagnostics/*`、`observability/event/*`（协作重试聚合字段）
  - `integration/*`（collab retry 契约与 shared gate 入口）
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 保持 `0.x` 兼容窗口：新增字段遵循 `additive + nullable + default`。
  - 默认行为保持不变（`composer.collab.retry.enabled=false`）。

