## Why

当前 Action Timeline 已完成标准化与聚合观测（H1/H1.5），但仍缺少对高风险动作的“执行前确认”闸门能力。推进 H2 可以在不引入 H3 pause/resume 主状态机的前提下，先建立可审计、可配置、Run/Stream 语义一致的 HITL 控制基线。

## What Changes

- 新增 Action Gate（H2）能力：在工具动作执行前引入可配置的确认闸门。
- 新增 Gate 判定与回调接口（library-first）：由业务侧注入确认逻辑，框架不绑定 CLI/前端实现。
- 默认策略设为 `require_confirm`：未配置回调时按确认语义处理，不自动放行。
- 首期风险判定仅基于 `tool name + keyword`，不包含参数 schema 规则。
- 统一 Run/Stream 语义：在 `allow/deny/timeout` 场景下保持一致行为与错误口径。
- 默认超时策略为 `deny`。
- 新增最小 diagnostics 字段与 timeline reason code，补齐契约测试与并发安全验证。

## Capabilities

### New Capabilities
- `action-gate-hitl`: 定义 H2 外部编排式 HITL 的 Action Gate 要求（确认策略、超时语义、Run/Stream 一致性、观测字段）。

### Modified Capabilities
- `action-timeline-events`: 新增 gate 相关 reason code 与阶段语义，保持既有 timeline 事件兼容。
- `runtime-config-and-diagnostics-api`: 增补 Action Gate 配置与最小运行诊断字段，明确默认策略与超时行为。

## Impact

- 代码范围：`core/runner`、`core/types`、`runtime/config`、`runtime/diagnostics`、`observability/event`。
- 测试范围：新增 Run/Stream gate 契约测试（allow/deny/timeout），并维持 `go test -race ./...` 基线。
- 文档范围：`README.md`、`docs/development-roadmap.md`、`docs/runtime-config-diagnostics.md`、`docs/v1-acceptance.md`。
- 兼容性：不引入 H3 pause/resume 主状态机改造；不新增 CLI 依赖；保持 library-first。
