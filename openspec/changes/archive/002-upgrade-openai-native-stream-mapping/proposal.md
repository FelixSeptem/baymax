## Why

当前 `model/openai` 的流式路径仍是 compatibility-first 实现，已成为 v1 已知限制，并且位于 R1 的高优先级稳定化项。现在升级到 OpenAI 官方 SDK 的原生流式事件映射，可以在不扩大范围的前提下提升流式语义稳定性、可观测链路一致性与回归可测性。

## What Changes

- 在 `model/openai` 中使用 OpenAI Go 官方 SDK 的 Responses 原生 streaming 事件，替换现有兼容流式路径。
- 明确并固化 SDK 事件到 `types.ModelEvent` 的映射规则，允许新增事件枚举类型以覆盖原生流式语义。
- 流式执行路径采用 fail-fast 语义：一旦出现错误立即终止并返回分类错误。
- 仅暴露完整 tool call，不暴露参数增量片段。
- 增加 adapter 层与 integration 层测试（包含 golden 事件序列）以防止顺序、错误分类与终态回归。
- 接入 `golangci-lint`，并新增建议配置文件，纳入开发与 CI 检查流程。
- 更新 `docs/` 下相关文档与本 change artifacts，确保行为约束、限制与验收标准一致。

## Capabilities

### New Capabilities
- `openai-native-stream-mapping`: 基于 OpenAI 官方 SDK 的原生流式事件映射与 fail-fast 终止语义。
- `go-quality-gate`: 通过 `golangci-lint` 配置与流程化检查建立 Go 代码质量门禁基线。

### Modified Capabilities
- None.

## Impact

- Affected code:
  - `model/openai/client.go`
  - `core/types/types.go`（新增/扩展流式事件类型定义）
  - `core/runner/runner.go`（stream 错误终止与事件发射对齐）
  - `integration/*`、`model/openai/*_test.go`（回归测试与 golden）
  - `.golangci.yml` 与 CI 配置文件（若仓库已有 CI）
- External dependencies:
  - 继续使用 `github.com/openai/openai-go` 官方 SDK（不新增 LLM SDK 类型依赖）
  - 新增/启用 `golangci-lint` 工具链依赖
- Behavioral/API impact:
  - `ModelEvent.Type` 允许新增枚举值（向后兼容扩展）
  - 流式错误处理改为立即终止，降低行为歧义
