## Why

Context Assembler CA2 当前仅支持 `file` provider，`rag/db` 仍为 not-ready，占位状态已成为 R3 Knowledge 能力落地的主要阻塞点。现在需要在不绑定特定供应商 SDK 的前提下，先把可扩展的外部检索接入层做实，形成稳定的 Stage2 检索主路径。

## What Changes

- 在 CA2 Stage2 引入统一 Retriever SPI，规范检索请求/响应与错误语义。
- 新增通用 `http` retriever adapter，支持通过 JSON 映射配置接入外部检索服务。
- 将 `rag`、`db`、`elasticsearch` provider 从 not-ready 占位升级为可运行实现（基于统一 SPI）。
- 扩展配置模型，支持 provider 级 endpoint、鉴权头、JSON 字段映射与超时参数（仍遵循 `env > file > default`）。
- 保持 `fail_fast / best_effort` 语义不变，并将 Stage2 命中与降级原因写入 diagnostics。
- 补齐单元测试与最小集成测试（mock HTTP retriever），不依赖真实外部服务。
- 同步更新 README 与 docs，确保实现与文档一致。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `context-assembler-stage-routing`: 扩展 Stage2 provider 能力为 `file/http/rag/db/elasticsearch`，并要求通过统一 Retriever SPI 执行检索。
- `runtime-config-and-diagnostics-api`: 扩展 CA2 provider 配置项与 diagnostics 字段（`stage2_hit_count`、`stage2_source`、`stage2_reason`）契约。

## Impact

- 受影响模块：`context/assembler`、`context/provider`、`runtime/config`、`runtime/diagnostics`、相关 tests 与 docs。
- 不变更 runner 主状态机与 model/tool 现有对外契约。
- 外部依赖策略：不新增供应商绑定 SDK，优先通用 HTTP + JSON 映射接入路径。
