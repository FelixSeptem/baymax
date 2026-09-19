# model 组件说明

## 功能域

OpenAI、Anthropic 与 Gemini adapter 各自拥有 provider-native tool-call/result、thinking、usage 和 stream edge 转换。跨 Provider conformance 只输出有界 canonical digest/correlation；首个语义 stream event 后不得 fallback。离线证据见 `tool/diagnosticsreplay/testdata/provider_handoff_stream_edge.v1.json`，门禁见 `scripts/check-provider-handoff-stream-edge-contract.sh/.ps1`。该合同不新增配置、credential store、远程 catalog 或共享 wire protocol。

`model` 提供多 Provider 模型适配，当前包含：

- `openai`
- `anthropic`
- `gemini`
- `providererror`（错误归类工具）
- `toolcontract`（工具结果输入合同构建）

Canonical 架构入口：`docs/runtime-harness-architecture.md`

## 架构设计

每个 provider 子包都实现统一契约：

- `types.ModelClient`（`Generate` / `Stream`）
- `types.ModelCapabilityDiscovery`（能力探测）
- 条件支持 `CountTokens`（按 SDK 能力实现）

适配器负责：

- SDK 请求/响应映射
- 流式事件标准化为 `types.ModelEvent`
- 工具调用事件标准化为 `types.ToolCall`
- provider 错误归类与 `Retryable` 语义对齐
- 工具结果回灌输入的 canonical envelope 构建

## 请求侧投影契约（provider_request_projection.v1）

Runtime → Provider 方向与响应侧是同族合同：`model/conformance` 拥有 provider-neutral 的 `source`/`observed` 双投影、canonical digest 与稳定分类词表，`tool/diagnosticsreplay` 负责离线只读 replay，三个 adapter 在包内测试中通过既有缝隙（`newResponse`/`newStream`、`Config.GenerateFn`/`StreamFn`）捕获真实 SDK 请求形状。

- 版本化 fixture：`tool/diagnosticsreplay/testdata/model_request_projection.v1.json`，覆盖 OpenAI / Anthropic / Gemini × run / stream，以及「无 tool result」分支。
- 生成入口：`BAYMAX_REGEN_REQUEST_PROJECTION_FIXTURE=1 go test ./tool/diagnosticsreplay -run TestGenerateProviderRequestProjectionFixture`。正常测试会校验提交物与契约构建器逐字节一致，提交物不得被静默改写。
- `declared_gap` 是**待修复项的证据锚点**，不是容忍语义：缺口未申报、或已申报缺口不再复现，都会以稳定码失败并要求显式更新契约。
- cache 用量按 `additive + nullable + default` 演进：当前 adapter 没有 cache 会计来源，投影固定 `available=false` 且不得伪造 read/write 值；新增 provider 字段不得破坏历史 fixture。
- 分类词表（与 `model/conformance`、`tool/diagnosticsreplay` 与门禁保持一致）：
  `provider_request_schema_drift`、`provider_request_role_projection_drift`、`provider_request_tool_result_native_drift`、`provider_request_part_ordering_drift`、`provider_request_stable_prefix_drift`、`provider_request_tool_order_drift`、`provider_request_capability_projection_drift`、`provider_request_run_stream_parity_drift`、`provider_cache_usage_projection_drift`、`provider_request_overflow_drift`、`provider_request_contract_drift`。
- 已知不对称（基线事实，不是本契约的容忍项）：`CountTokens` 路径会原生投影 `Messages` 的 system/assistant 角色，而 `Generate`/`Stream` 把整个请求压平为单段文本，因此 token 会计与实发请求可能不对应。这是后续增量 change 的触发证据。
- 门禁：`scripts/check-provider-request-projection-contract.sh` / `.ps1`。

## 关键入口

- `openai/client.go`
- `anthropic/client.go`
- `gemini/client.go`
- `providererror/classified.go`
- `toolcontract/input.go`

子模块文档：

- `providererror/README.md`
- `toolcontract/README.md`

## 边界与依赖

- Provider 协议细节必须收敛在 `model/<provider>`，不得泄漏到 `core/*` 或 `context/*`。
- 上层仅依赖 `core/types` 契约接口，不依赖具体 SDK 类型。
- `toolcontract` 只负责输入合同，不承载 provider 传输与 SDK 调用。
- 新增 provider 时应复用同一事件和错误语义，避免跨 provider 行为漂移。

## 配置与默认值

- Provider 选择、模型名与凭证来自运行时配置与环境变量，不在 `model/*` 中硬编码。
- 未显式声明能力时，适配器应回退为保守能力集（如 token counting unsupported）。
- 错误归类默认走 `providererror` 标准路径。

## 可观测性与验证

- 关键验证：`go test ./model/openai ./model/anthropic ./model/gemini ./model/providererror ./model/toolcontract -count=1`。
- 主链路验证通过 `core/runner` 与 integration 契约测试覆盖 run/stream 等价。
- provider 错误与降级语义需在诊断事件中保持可追踪原因码。

## 扩展点与常见误用

- 扩展点：新增 provider 子包并实现 `types.ModelClient` + capability discovery。
- 常见误用：把 SDK 原始类型直接暴露到 `core/*`，造成边界泄漏。
- 常见误用：run 与 stream 返回不同语义终态，破坏契约一致性。
