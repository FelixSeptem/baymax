## Why

当前仓库模型层仅稳定支持 `model/openai`，与 roadmap 中 R3 的“多 provider 可插拔”目标存在缺口。  
在 runtime/config/diagnostics、MCP 可靠性与并发基线已经收敛后，优先落地多 provider 的最小非流式能力，可以在可控范围内降低后续 Anthropic/Gemini 接入成本，并为 M2（流式语义对齐）打基础。

## What Changes

- 新增 `model/anthropic`，优先使用官方 SDK，提供最小非流式调用实现。
- 新增 `model/gemini`，优先使用官方 SDK，提供最小非流式调用实现。
- 对齐三 provider（OpenAI/Anthropic/Gemini）在 `Run/Generate` 路径的统一结果语义（不包含 streaming）。
- 增加跨 provider 契约测试，覆盖最小成功路径与基础错误分类映射。
- 在实现与文档中增加 M2 TODO 占位：后续细化 provider 错误分类映射粒度与流式语义对齐。
- 同步更新已有文档（README + `docs/development-roadmap.md` + `docs/v1-acceptance.md`），不新增独立文档。

## Capabilities

### New Capabilities
- `llm-multi-provider-minimal`: 在现有 OpenAI 之外，新增 Anthropic/Gemini 的最小非流式 provider 适配能力。

### Modified Capabilities
- `openai-native-stream-mapping`: 明确其职责聚焦 OpenAI streaming；多 provider streaming 对齐属于后续 M2 提案范围。

## Impact

- 影响目录：
  - `model/anthropic`
  - `model/gemini`
  - `integration`（新增跨 provider 契约测试）
  - `README.md`、`docs/development-roadmap.md`、`docs/v1-acceptance.md`
- API 影响：
  - 不新增对外 streaming 占位 API。
  - 维持现有调用路径，新增 provider 实现可按相同接口接入。
- 风险控制：
  - 明确 M1 仅非流式，避免一次性引入流式/tool-call 多维变更。
