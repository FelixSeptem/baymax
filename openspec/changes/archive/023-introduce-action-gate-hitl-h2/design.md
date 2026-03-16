## Context

Action Timeline H1/H1.5 已提供标准化阶段事件与 phase 聚合诊断，但当前运行时缺少“执行前确认”的控制平面，无法对高风险动作建立统一闸门。当前 roadmap 将 H2 定义为“Action Gate（外部编排式 HITL）”，要求在不引入 H3 pause/resume 主状态机的前提下，先完成确认策略、审计语义和 Run/Stream 一致性闭环。

约束：
- 保持 library-first，不依赖 CLI。
- 不改变现有 tool-call complete-only 与 streaming 契约。
- 并发安全是基线，需保持 `go test -race ./...` 通过。

## Goals / Non-Goals

**Goals:**
- 引入 Action Gate 判定与确认回调接口，支持 `require_confirm` 默认策略。
- 风险判定首期仅基于 `tool name + keyword`，保证实现收敛与可解释。
- 在 Run/Stream 中统一 gate 行为（allow/deny/timeout）与错误语义。
- 新增最小诊断字段与 timeline reason code，支持审计与回放分析。
- 保持文档与实现一致（README/runtime-config-diagnostics/v1-acceptance/roadmap）。

**Non-Goals:**
- 不引入 H3 pause/resume 状态机与跨请求挂起恢复。
- 不实现 CLI/前端确认交互。
- 不引入参数 schema 级风险判定规则。
- 不扩展 A2A/MCP 协议层来承载确认交互。

## Decisions

### 1) Action Gate 采用“判定器 + 回调器”双接口
- 决策：
  - `GateMatcher`: 根据 tool 名称和关键词产出 gate 决策（allow/require_confirm/deny）。
  - `GateResolver`: 当决策为 `require_confirm` 时向业务侧请求确认结果。
- 理由：
  - 将“风险识别”与“确认来源”解耦，适配不同集成方式（HTTP/UI/人工系统）。
  - 保持 runner 核心简洁，避免绑定单一交互协议。
- 备选：将判定与确认塞入单接口。
  - 放弃原因：可测试性与扩展性差，难以单独验证风险规则与确认超时语义。

### 2) 默认策略为 `require_confirm`，未配置 resolver 视为拒绝
- 决策：
  - 默认 gate 策略是 require_confirm。
  - 当触发 confirm 且未配置 resolver 时，按 deny 处理并 fail-fast。
- 理由：
  - 符合“高风险动作默认不自动放行”的安全基线。
  - 避免 silent bypass。
- 备选：未配置 resolver 时 warn 并放行。
  - 放弃原因：会弱化 H2 的控制价值，且与用户确认口径不一致。

### 3) 超时语义统一为 deny
- 决策：resolver 超时直接 deny，错误分类进入既有 error taxonomy（context/tool policy 路径）。
- 理由：
  - 与 require_confirm 语义一致，避免超时导致隐式放行。
  - Run/Stream 可保持统一处理路径。
- 备选：超时继续执行。
  - 放弃原因：审计语义不完整且存在风险暴露。

### 4) 可观测性采用最小字段集 + reason code
- 决策：
  - 诊断新增：`gate_checks`、`gate_denied_count`、`gate_timeout_count`。
  - timeline 增加 reason code：`gate.require_confirm`、`gate.denied`、`gate.timeout`。
- 理由：
  - 保持字段最小化，先覆盖治理闭环。
  - 为后续 H3/H4 扩展预留稳定口径。
- 备选：一次性引入完整 gate 审计对象。
  - 放弃原因：本期 scope 过大，且与 H3 边界耦合过深。

## Risks / Trade-offs

- [Risk] 关键词误判导致误拦截
  → Mitigation: 首期提供可配置关键词与工具白名单，测试覆盖误判边界。

- [Risk] resolver 实现质量参差，造成高延迟或假拒绝
  → Mitigation: 统一 timeout 配置 + deny 默认，并在 diagnostics 暴露 timeout 计数。

- [Risk] Run/Stream 语义分叉
  → Mitigation: 增加共享契约测试矩阵，强制校验等价结果与 reason code。

- [Risk] 过早引入复杂 HITL 状态机冲击稳定性
  → Mitigation: 明确非目标，不触发 H3 pause/resume 改造。

## Migration Plan

1. 在 `core/types` 增加 gate 判定/确认接口与最小事件模型。
2. 在 `runtime/config` 增加 gate 配置项（默认 `require_confirm`，timeout 默认 deny）。
3. 在 `core/runner` 的 tool 执行前接入 gate 决策链路，统一 Run/Stream 行为。
4. 在 `observability/event` 与 `runtime/diagnostics` 接入最小字段与 reason code。
5. 补齐契约测试（allow/deny/timeout + Run/Stream 等价 + race）。
6. 同步 README 与 docs，保证文档实现一致。

## Open Questions

- H2 结束后是否直接推进 H3 pause/resume，还是先进行一轮线上观测评估再进入 H3。
- Gate rule 是否需要在下一期纳入参数 schema 规则（当前明确不做）。
