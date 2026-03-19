## Why

当前 workflow 已具备确定性 DAG、重试/超时、A2A 远程步骤与 checkpoint/resume，但复杂流程仍缺少图级复用能力，导致步骤与条件大量重复定义。A15 通过引入子图复用与条件模板，在不改变现有执行语义的前提下提升 DSL 表达力与可维护性。

## What Changes

- 新增 Workflow 图复用能力：`subgraphs` + `use_subgraph`。
- 新增条件模板能力：`condition_templates` + `template_vars`（仅作用于 `condition`）。
- 引入“编译展开”阶段：先展开子图与模板，再进入现有扁平 DAG 规划与执行。
- 固化展开规则：递归深度上限为 `3`；展开后步骤 ID 采用 `<subgraph_alias>/<step_id>`。
- 固化覆盖策略：允许覆盖 `retry` 与 `timeout`，禁止覆盖 `kind`。
- 新能力默认关闭，仅在显式启用特性开关时生效。
- 非法组合全部 fail-fast：模板缺失、变量缺失、子图循环引用、ID 冲突、越界深度。
- 增加 contract tests、quality gate 与文档索引映射，确保代码/测试/文档同步。

## Capabilities

### New Capabilities
- `workflow-graph-composability`: 定义 workflow 子图复用与条件模板的编译、校验、执行与兼容契约。

### Modified Capabilities
- `workflow-deterministic-dsl`: 扩展 DSL 编译与确定性规划契约，纳入子图展开规则。
- `multi-agent-composed-orchestration`: 增加 composer + workflow 在子图场景下的组合语义约束。
- `runtime-config-and-diagnostics-api`: 增加 graph composability feature flag 与 additive 诊断字段契约。
- `go-quality-gate`: 增加 graph composability 合同测试与阻断规则。

## Impact

- 代码：
  - `orchestration/workflow/*`
  - `orchestration/composer/*`（workflow 集成路径）
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `tool/contributioncheck/*`
  - `scripts/check-multi-agent-shared-contract.*`
- 测试：
  - `integration/*workflow*` graph composability 合同矩阵
  - Run/Stream 等价、resume 一致性、非法输入 fail-fast 覆盖
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 默认行为不变（feature flag 默认关闭）；
  - 新增字段与输出遵循 `additive + nullable + default`。
