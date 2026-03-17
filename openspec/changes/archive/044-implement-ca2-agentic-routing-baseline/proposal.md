## Why

当前 `context_assembler.ca2.routing_mode` 已支持配置 `agentic`，但运行时仍返回 not-ready，形成“可配置但不可用”的能力断点。需要在保持现有 CA2/CA3 与 Run/Stream 语义稳定的前提下，交付可落地的 agentic 路由基线能力，并提供失败回退路径。

## What Changes

- 将 CA2 `agentic` 路由从占位实现升级为可用实现，引入宿主 callback 决策扩展接口。
- 在 `routing_mode=agentic` 下接入 callback 决策；当 callback 超时/异常/非法结果时，按 `best_effort` 自动回退到 `rules` 路由。
- 强制 Run/Stream 在等价输入与配置下保持路由决策语义等价。
- 扩展 CA2 最小诊断字段：`stage2_router_mode`、`stage2_router_decision`、`stage2_router_reason`、`stage2_router_latency_ms`、`stage2_router_error`。
- 保持本次变更边界：不引入灰度开关，不新增内建 LLM 路由器，不改变 Stage2 provider 与 stage policy 的既有语义。

## Capabilities

### New Capabilities
- 无

### Modified Capabilities
- `context-assembler-stage-routing`: 将 agentic 路由从 TODO 占位升级为 callback 可用路径，并定义失败回退 `rules` 的契约。
- `runtime-config-and-diagnostics-api`: 扩展 CA2 agentic 路由配置与路由决策诊断字段，保持热更新与 fail-fast/rollback 语义一致。

## Impact

- 受影响代码：`context/assembler/*`、`core/runner/*`（上下文装配调用链）、`runtime/config/*`、`runtime/diagnostics/*`、`observability/event/*`。
- 受影响测试：新增/更新 CA2 agentic 路由契约测试（callback success/fallback/timeout/error，Run/Stream 等价）。
- 受影响文档：`docs/runtime-config-diagnostics.md`、`docs/development-roadmap.md`、`docs/v1-acceptance.md`。
- 兼容性：pre-1.x 增量能力扩展；默认仍保持 `rules` 路由，显式启用 `agentic` 后生效。
