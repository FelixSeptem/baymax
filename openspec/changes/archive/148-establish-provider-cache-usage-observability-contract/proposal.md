## Why

归档 142 已建立请求侧投影的 cache 可用性占位，归档 144 已修复 supported provider 的原生请求投影，但三家官方 SDK 现在已经提供不同形态的 cache usage 来源，Baymax 仍只能把 cache 标记为 unavailable。缺少一个 provider-neutral、可回放且兼容历史 payload 的 usage 投影，会使 prompt cache 的实际成本、命中收益和 Run/Stream 差异无法被验证；本提案只补齐观测契约，不引入缓存策略。

## What Changes

- 扩展既有 `provider-request-projection-and-cache-observability` capability，建立 additive、nullable、default 的 cache usage 投影字段。
- 在 OpenAI、Anthropic、Gemini adapter 的 Run 与 Stream 路径捕获官方 SDK 可表达的 cache usage 来源，并归一化为 bounded provider-neutral facts。
- 将归一化结果放入独立的 `ModelResponse.CacheUsage`；Stream 仅在终态事件 `Meta["cache_usage"]` 暴露权威快照，不改变既有 `TokenUsage` 或 RuntimeRecorder schema。
- 区分 cache read、cache write/create 与合计可用性；无法可靠表达或来源缺失时保持 `available=false`，不从普通 input token 推导 cache 数值。
- 更新 `provider_request_projection.v1` 及必要的 usage fixture/replay，覆盖正向、缺失、未知字段、负值/不一致和 Run/Stream parity 场景。
- 增加离线 replay、contract gate、canonical digest 与文档映射，保证历史 fixture 缺少新字段仍可读取，新增字段不会改变旧 digest 语义。
- **不包含** prompt cache 创建/刷新/失效策略、stable-prefix 重排、缓存 key、价格模型、自动路由、远程 catalog、credential store、运行时配置或新的 RuntimeRecorder 事实源。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `provider-request-projection-and-cache-observability`: 为既有请求投影补充真实 provider cache usage 的 nullable/default 观测、归一化、回放与 Run/Stream 对等要求。

## Impact

- 主要代码 owner：`model/conformance`、`model/openai`、`model/anthropic`、`model/gemini`、`tool/diagnosticsreplay`。
- 主要证据：provider SDK 边界 fixture、usage normalization、replay taxonomy、contract gate 与 adapter unit tests；不调用 live provider。
- 兼容性：现有 `TokenUsage` 的 input/output/total 字段保持不变；新 cache 字段仅 additive + nullable + default，历史 JSON 缺失时按 unavailable 处理，并通过独立响应载体提供给调用方。
- 文档影响：`model/README.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md` 及相关 spec/replay 映射。

## Example Impact Assessment

无需示例变更（附理由）：本提案只新增离线 usage fixture、replay、adapter 观测和 contract gate，不改变 `examples/agent-modes` 的配置、runtime path 或 expected markers。
