## Why

Context Assembler CA1 已建立 prefix 一致性、append-only 与 fail-fast 基线，但仍缺少“按需加载 + 阶段路由 + 末尾复述”能力，无法支撑后续 RAG/Memory 接入和复杂上下文编排。现在推进 CA2，可以在不破坏 Run/Stream 语义的前提下补齐 P3/P4/P5，并为后续 agentic 判定和存储后端扩展预留稳定接口。

## What Changes

- 在 `context/assembler` 引入 CA2 双阶段装配：Stage1（session/hot context）与 Stage2（retrieval provider）。
- 新增规则化 stage routing（先 Stage1，满足则跳过 Stage2；不满足再触发 Stage2），并保留 agentic 决策扩展钩子（TODO）。
- 新增 tail recap 输出（最小字段：`status`、`decisions`、`todo`、`risks`），作为末尾稳定块追加。
- 在 `runtime/config` 增加 CA2 可配置策略（stage 开关、失败策略、超时、routing 阈值、provider 选择）。
- 增加 retrieval provider 接口层：本期仅实现本地文件 provider；RAG/DB provider 只暴露接口与占位错误。
- 扩展 diagnostics 枚举与字段，覆盖 stage 命中、跳过原因、latency、recap 状态与失败分类。
- 同步更新 README 与 docs，明确 CA2 完成边界与 examples TODO（本期不新增示例实现）。

## Capabilities

### New Capabilities
- `context-assembler-stage-routing`: 定义 CA2 的双阶段装配、路由策略与 tail recap 语义。

### Modified Capabilities
- `context-assembler-baseline`: 从 CA1 扩展为支持 CA2 分阶段装配与 provider 接口。
- `runtime-config-and-diagnostics-api`: 扩展 CA2 配置项与 stage/recap 诊断字段约束。

## Impact

- 受影响模块：`context/assembler`、`context/guard`、`runtime/config`、`runtime/diagnostics`、`core/runner`。
- 新增模块：`context/provider`（或等效命名）用于 retrieval provider 接口与本地文件实现。
- 质量门禁保持不变：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- 文档影响：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/development-roadmap.md`、`docs/context-assembler-phased-plan.md`、`docs/v1-acceptance.md`。
