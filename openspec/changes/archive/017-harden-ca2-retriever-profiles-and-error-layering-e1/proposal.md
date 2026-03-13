## Why

CA2 Stage2 external retriever 已具备可运行主路径，但接入配置仍偏复杂、错误语义分层不足、诊断字段难以支撑快速排障与演进决策。当前需要在不引入供应商绑定 SDK 的前提下，完成 E1 收敛：通过 profile 模板和错误分层降低接入与运维成本，并保持现有 fail-fast/best-effort 语义稳定。

## What Changes

- 为 CA2 Stage2 external retriever 引入 profile 模板机制，首批支持 `http_generic`、`ragflow_like`、`graphrag_like`、`elasticsearch_like`，并保持可扩展。
- 在 Stage2 retrieval 路径引入错误分层（transport/protocol/semantic）与标准 reason code 映射。
- 扩展 runtime diagnostics：新增 `stage2_reason_code`、`stage2_error_layer`、`stage2_profile` 字段（向后兼容现有字段）。
- 提供 external retriever 配置预检查库接口（无 CLI）：warning 可继续，error 必须 fail-fast。
- 保持 `env > file > default`、`fail_fast/best_effort` 与现有 runner 主状态机语义不变。
- 同步更新文档并执行文档一致性检查，修正过时描述（含 v1 验收文档中的占位语义）。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `context-assembler-stage-routing`: Stage2 external retriever 增加 profile 模板语义与错误分层输出语义，且不改变 stage policy 行为。
- `runtime-config-and-diagnostics-api`: 增加 external retriever profile 与预检查 API 契约，扩展 Stage2 诊断字段并定义 warning/error 行为边界。

## Impact

- 受影响模块：`runtime/config`、`context/provider`、`context/assembler`、`runtime/diagnostics`、`observability/event`、相关测试与 docs。
- 不新增供应商 SDK 依赖，不引入 agentic routing 改造，不修改 runner/tool 对外契约。
- 需要补充配置模板、错误分类、诊断字段与文档一致性回归测试。
