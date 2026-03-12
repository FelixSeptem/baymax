## Why

`bootstrap-multi-llm-providers-m1` 已完成 OpenAI/Anthropic/Gemini 的最小非流式能力，但当前多 provider 在流式语义和错误分类上仍未对齐。  
这导致调用方在跨 provider 切换时需要处理不同事件粒度和错误语义，增加接入与维护成本。

为完成 R3 的多 provider 可插拔目标，需要在 M2 中收敛：
- 三 provider 的 streaming 语义
- 细粒度错误分类映射
- 文档中的阶段状态与边界描述

## What Changes

- 为 `model/anthropic` 与 `model/gemini` 增加 streaming 实现（官方 SDK 优先）。
- 对齐 OpenAI/Anthropic/Gemini 的流式事件语义，必要时新增 `ModelEvent.Type` 枚举。
- 继续保持“仅发完整 tool call，不发参数增量片段”的外部语义。
- 细化 provider 错误分类映射（auth / rate-limit / timeout / request / server / unknown）。
- 增加跨 provider streaming 契约测试（事件顺序、fail-fast、终态一致性）。
- 更新现有文档（README、roadmap、v1-acceptance 等）以反映 M2 状态。

## Capabilities

### New Capabilities
- `multi-provider-streaming-error-taxonomy`: 多 provider 流式事件语义与错误分类对齐能力。

### Modified Capabilities
- `llm-multi-provider-minimal`: 从“非流式最小能力”扩展为“流式+非流式语义一致”的多 provider 能力基线。
- `openai-native-stream-mapping`: 对齐到跨 provider streaming 统一语义集合。

## Impact

- 影响目录：
  - `model/openai`
  - `model/anthropic`
  - `model/gemini`
  - `core/types`（事件枚举扩展）
  - `integration`（跨 provider streaming 契约测试）
  - `README.md`、`docs/development-roadmap.md`、`docs/v1-acceptance.md`
- API 影响：
  - 允许新增 `ModelEvent.Type` 枚举值（向后兼容扩展）。
  - 不改变“tool-call 仅完整态对外可见”的既有契约。
