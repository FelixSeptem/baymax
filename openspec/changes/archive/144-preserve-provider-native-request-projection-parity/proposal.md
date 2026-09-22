## Why

归档 142（`establish-provider-request-projection-and-cache-observability-contract`）已经以真实 SDK 边界 fixture、离线 replay 和 gate 固定了一个跨三家 supported provider 的可复现缺口：当 `ModelRequest.Input` 非空时，OpenAI、Anthropic 与 Gemini 的 Generate/Stream 路径经 `toolcontract.CanonicalInput` 压平为单段 user 文本，丢弃或降级 `Messages` 中的 system/user/assistant 角色语义，并把 `ToolResult` 写成文本信封；但 Anthropic/Gemini 的 `CountTokens` 已走更结构化的 message 映射。由此产生「计费投影与实际请求不一致」、Skill system fragment / task-aware tail recap / assistant history 未以原生角色到达 SDK，以及 tool-result 关联未以原生 request part 传递的确定性 drift。

现在应在归档 142 的证据基线之上完成最小运行时修复：使每个 `model/<provider>` 适配器按各自官方 SDK 的原生 request shape 保留 canonical request facts，并保证 Run 与 Stream 使用同一投影语义。该 change 不再审计是否存在 gap，而是将已声明的 role、tool-result-native 与稳定顺序 gap 显式迁移为已满足的契约期望；若修复无法在三家 Provider 间保持有界、可回放和对等语义，gate 必须阻断。

## What Changes

- 在 `model/openai`、`model/anthropic` 与 `model/gemini` 内实现 provider-owned 的原生请求构造：保留 source-owned `ModelRequest.Messages` 的 system/user/assistant 角色与稳定顺序，并把具备 `call_id`/`tool_name` 的 canonical `ToolResult` 映射为各 SDK 可表达的原生 tool-result / function-response 形状。
- 定义单一、无 SDK 依赖的 canonical request-facts 归一化与校验边界，供三家 adapter 各自的 Generate、Stream 与 CountTokens 复用；它只描述来源、顺序、关联和有界 payload 规则，不成为新的 provider wire protocol，也不承载 provider SDK 类型。
- 保持 invalid tool feedback 的既有 `feedback_invalid` fail-fast 语义；在原生 tool-result 映射不可安全表达时，返回稳定 request-shape 分类并且不发送半成品 provider 请求，不退回到静默文本信封。
- 更新 `provider_request_projection.v1` fixture、replay、conformance 与双平台 gate：将归档 142 已证明、现由本 change 修复的 `declared_gap` 显式迁移为无 gap 的 native role / tool-result 预期；继续冻结 canonical digest、稳定顺序、Run/Stream parity、隐私与有界性。
- 增加跨 adapter 的正向、负向、边界与 Run/Stream / CountTokens 一致性测试，并用真实 SDK 请求参数级捕获证明实现实际到达 SDK 边界的形状。
- 同步 roadmap：将已归档的 Eval 首错归因候选标为归档基线，并将 Provider 后续方向明确为「142 证据基线后的原生请求投影修复」，而非重复建设审计契约。
- **不包含** prompt-cache usage 字段或诊断 schema 演进、远程 model catalog、本地模型路由、共享 provider wire/gateway、credential store、`context/*` 或 runtime 配置改动、scheduler/ReAct/tail recap 策略改动、raw prompt/reasoning/transcript 持久化。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `provider-request-projection-and-cache-observability`: 将请求投影从“可审计且允许以显式 declared gap 钉住”提升为 supported adapter 必须保留原生角色、tool-result 关联与稳定顺序的运行时语义；保留 cache usage 仅可用性基线，不在本 change 增加 cache schema。
- `llm-multi-provider-minimal`: 明确 OpenAI、Anthropic 与 Gemini 的 Generate、Stream、CountTokens 对等使用 canonical request facts，并将 canonical tool-result feedback 映射为 provider-native request shape，而不是文本信封降级。
- `cross-provider-handoff-and-stream-edge-conformance`: 将既有请求侧 conformance 从“审计时不得改变运行时行为”演进为“按 provider-owned 原生 request shape 修复后必须有界、可回放、无原始 payload 且 Run/Stream 对等”。

## Example Impact Assessment

无需示例变更（附理由）

本 change 修改 provider adapter 的 SDK 请求构造，但不增加 agent-mode 配置键、不改变 examples 的 runtime path 或 expected markers；第一阶段验证使用离线 fixture、SDK 边界捕获和 conformance gate。若实现表明既有 agent-mode 示例的可观察 tool-result、usage 或输出 marker 发生变化，必须暂停代码任务，先更新 `examples/agent-modes/MATRIX.md` 与受影响模式 README，再重新声明为“修改示例”。

## Impact

- 主要代码 owner：`model/toolcontract`（SDK-neutral canonical request facts 与 fail-fast 校验）、`model/openai`、`model/anthropic`、`model/gemini`（各自 SDK 原生参数构造）。Provider 官方 SDK 依赖继续只位于对应 `model/<provider>` 包。
- 测试/证据：`model/conformance`、各 adapter 的 request projection tests、`tool/diagnosticsreplay`、`tool/contributioncheck`、`provider_request_projection.v1` fixture 与 shell/PowerShell gate。
- 文档：`README.md`、`model/README.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/runtime-module-boundaries.md` 及必要的 replay contract 映射。
- 风险：不同 SDK 对 system prompt、assistant history 与 tool-result 的承载能力不完全相同；本 change 必须在 Design 阶段逐 adapter 定义可表达形状及 fail-fast 边界，禁止为了“统一”而引入非原生文本降级或新的共享 SDK abstraction。
- 回滚：恢复每个 adapter 的此前请求构造和归档 142 的 `declared_gap` fixture 期望，并移除本 change 新增的 native-shape helpers/tests/gate assertions；无配置迁移、无持久化迁移、无诊断 schema 回滚。
