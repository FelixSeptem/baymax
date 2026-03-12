## Why

当前 runner 已具备 OpenAI/Anthropic/Gemini 的统一调用与流式语义，但在 provider 特性快速变化背景下，仍缺少基于官方 SDK 的动态能力发现与自动降级策略。该缺口已在 roadmap R3 M3 与 v1 limitation 中明确，若不补齐会导致请求在特性不匹配时直接失败，增加接入方维护成本与运行风险。

## What Changes

- 引入 provider 能力模型与动态探测接口，优先通过各官方 SDK 的可用能力元数据或探测 API 进行运行期发现，避免静态硬编码能力表。
- 在 model step 执行前增加能力判定流程；当当前 provider 不满足请求能力时，按配置的 provider 优先链路进行自动降级。
- 统一降级行为语义：仅支持 model-step 级 fallback，不支持 streaming 中途切换 provider；无可用候选时 fail-fast 终止。
- 扩展 runtime 诊断输出，提供能力判定结果与 fallback 路径摘要，保持 run/stream 外部语义一致。
- 同步更新 README 与 docs，确保实现状态、roadmap 与已知限制一致。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `llm-multi-provider-minimal`: 增加动态能力发现、请求能力判定、provider 级自动降级与 fail-fast 终止要求。
- `runtime-config-and-diagnostics-api`: 增加 fallback 相关诊断字段与可观测摘要要求。

## Impact

- 受影响代码：`core/runner`、`core/types`、`model/openai`、`model/anthropic`、`model/gemini`、`runtime/config`、`runtime/diagnostics`、`integration/`。
- 配置影响：新增或扩展 provider fallback 顺序与能力探测策略配置项（默认保持安全 fail-fast）。
- 测试影响：新增 capability/fallback 契约测试与跨 provider 回归覆盖（含 `go test -race ./...`）。
- 文档影响：`README.md`、`docs/development-roadmap.md`、`docs/v1-acceptance.md` 需要与实现收敛。
