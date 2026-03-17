## Why

CA2 external retriever 已完成 E1/E2 的可运行与观测基线，但仍缺少两类关键能力：一是 SPI 层可扩展的 capability/hint 透传口，二是可复用的 provider 风格模板包。当前接入仍依赖大量手工 mapping，导致跨系统接入成本高、语义一致性难以稳定复用。

本提案聚焦 E3 的“低耦合增强”：在不改变 assembler 主流程和既有 stage policy 语义的前提下，补齐 capability hints 与模板包能力，为后续按需引入 provider 专用能力保留扩展位。

## What Changes

- 在 CA2 Stage2 external retriever SPI 与 runtime config 层新增 capability hints 扩展口：
  - 主流程保持 provider-agnostic，assembler 不感知 provider 特有实现细节。
  - hint 不匹配仅输出观测信号，不自动切换 provider 或改变路由决策。
- 引入 CA2 external template pack（首期仅 3 个 profile）：
  - `graphrag_like`
  - `ragflow_like`
  - `elasticsearch_like`
  - 模板解析顺序固定为“profile defaults -> explicit overrides”，并允许仅用显式配置单独运行。
- 保持 Stage2 错误语义分层模型不变（`transport|protocol|semantic`），仅允许增量扩展字段。
- 明确 Run/Stream 在 hint 解析、模板解析、Stage2 结果分类上的语义等价要求（允许实现层事件时序差异）。
- 增加验证门禁：
  - 契约测试：hint/template 解析、观测语义、Run/Stream 等价。
  - benchmark 回归：保留既有趋势基线，并新增 hint/template 解析开销 baseline。
- 补齐精简集成文档：每个模板提供 YAML 示例与字段说明，不新增 runnable example。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `context-assembler-stage-routing`: 增加 Stage2 capability hints 扩展口与模板解析语义约束，保持主流程与策略行为不变。
- `runtime-config-and-diagnostics-api`: 增加 hints/template 配置与诊断扩展字段契约，补齐校验、观测与兼容要求。

## Impact

- Affected code:
  - `context/provider`
  - `context/assembler`
  - `runtime/config`
  - `runtime/diagnostics`
  - `observability/event`
  - `integration`（contract tests + benchmark）
- Affected docs:
  - `docs/runtime-config-diagnostics.md`
  - `docs/ca2-external-retriever-evolution.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
- Compatibility:
  - 所有新增配置与诊断字段均为增量扩展，现有消费者可继续按旧字段运行。
  - 本期不引入自动策略动作，不改变既有 `fail_fast/best_effort` 行为。
