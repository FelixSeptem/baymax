# model 组件说明

## 功能域

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
