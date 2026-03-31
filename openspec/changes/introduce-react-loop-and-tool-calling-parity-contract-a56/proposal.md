## Why

当前主线在 `Run` 路径已具备工具回灌闭环，但 `Stream` 仍处于 `stream_tool_dispatch_not_supported` 中间态，导致 ReAct（Reason-Act-Observe）在双路径语义上不等价。为支持业务侧顺滑搭建 ReAct agent，本提案需要一次性覆盖运行闭环、provider 归一、安全准入、可观测、回放与门禁合同，避免后续再拆分 ReAct 主题提案。

## What Changes

- 新增 ReAct 闭环 contract：统一 `model -> tool dispatch -> tool result feedback -> next iteration` 的运行语义与终止条件。
- 扩展 `Stream` 主路径，补齐工具分发与结果回灌能力，收敛到与 `Run` 语义等价。
- 新增 run-level 工具调用上限 contract（与 iteration 上限协同），防止 ReAct 循环失控。
- 扩展多 provider tool-calling 归一 contract（OpenAI/Anthropic/Gemini），冻结 tool-call request/response 与错误 taxonomy 映射。
- 扩展 readiness preflight + admission guard，新增 react 必需能力与降级/阻断映射，保证不满足 ReAct 前置条件时 fail-fast 且 side-effect-free。
- 扩展 sandbox 执行隔离合同在 ReAct 多轮工具路径上的一致性语义（allow/deny/fallback/taxonomy），避免工具循环下的安全语义漂移。
- 扩展 `runtime/config` 与 diagnostics additive 字段，新增 react loop 相关配置、计数、终止 reason 与 drift 可观测字段。
- 扩展 diagnostics replay：新增 `react.v1` fixture 与 drift 分类断言，覆盖 Run/Stream parity 与 mixed-fixture backward compatibility。
- 扩展 quality gate：新增 `check-react-contract.sh/.ps1` 并接入 `check-quality-gate.*`，暴露独立 required-check 候选。
- 同步模板/示例/文档索引，给出 ReAct 最小接入蓝图与迁移映射，避免业务侧重复拼装。
- 本提案不改 A54/A55 范围，仅在其之上一次性完成 ReAct 能力收敛。

## Capabilities

### New Capabilities
- `react-loop-and-tool-calling-parity-contract`: 冻结 ReAct 运行闭环、Run/Stream 语义等价、loop limit 与终止 taxonomy 合同。

### Modified Capabilities
- `llm-multi-provider-minimal`: 增加多 provider tool-calling 请求/响应归一与 Run/Stream tool loop 等价要求。
- `runtime-config-and-diagnostics-api`: 增加 `runtime.react.*` 配置域与 react additive 诊断字段。
- `runtime-readiness-preflight-contract`: 增加 react 前置能力可用性 finding 与 strict/non-strict 映射。
- `runtime-readiness-admission-guard-contract`: 增加 react 相关 admission 决策与 deny side-effect-free 约束。
- `sandbox-execution-isolation`: 增加 ReAct 多轮工具执行下 sandbox 决策一致性要求。
- `diagnostics-replay-tooling`: 增加 `react.v1` fixture 与 drift 分类断言。
- `go-quality-gate`: 增加 react contract gate 与独立 required-check 暴露。

## Impact

- 代码：
  - `core/runner`（Run/Stream ReAct loop 收敛、loop limit、终止语义）
  - `core/types`（react loop policy/metadata 契约字段）
  - `model/openai`、`model/anthropic`、`model/gemini`（tool-calling 归一映射）
  - `runtime/config`、`runtime/config/readiness`、`runtime/diagnostics`、`observability/event`（react 配置、admission/preflight、additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（`react.v1` fixtures + parity suites）
  - `scripts/check-quality-gate.*` + `check-react-contract.*`
  - `examples/*`（ReAct 最小接入与 Run/Stream 等价示例）
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性：
  - 保持外部 API 兼容，新增字段遵循 `additive + nullable + default`。
  - 通过 fail-fast + rollback + replay + gate 保持合同稳定收敛。
