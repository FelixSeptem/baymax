# model 组件说明

## 功能域

`model` 提供多 Provider 模型适配，当前包含：

- `openai`
- `anthropic`
- `gemini`
- `providererror`（错误归类工具）

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

## 关键入口

- `openai/client.go`
- `anthropic/client.go`
- `gemini/client.go`
- `providererror/classified.go`

## 边界与依赖

- Provider 协议细节必须收敛在 `model/<provider>`，不得泄漏到 `core/*` 或 `context/*`。
- 上层仅依赖 `core/types` 契约接口，不依赖具体 SDK 类型。
- 新增 provider 时应复用同一事件和错误语义，避免跨 provider 行为漂移。
