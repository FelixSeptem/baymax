## Why

当前存在 3 个并行未完成变更：
- `teams-runtime-baseline`
- `workflow-dsl-baseline`
- `a2a-minimal-interoperability`

三者同时修改 `action-timeline-events`、`runtime-config-and-diagnostics-api`、`runtime-module-boundaries`，且都引入 `task_id/agent_id/...` 与 reason/status 语义。若不先冻结共享契约，会在实现阶段产生字段漂移、reason code 冲突和回放口径不一致。

本提案作为 R4 前置阻断项：先冻结共享契约与门禁，再进入 Teams/Workflow/A2A 代码实现。

## What Changes

- 冻结多代理共享标识、状态映射、reason code 命名空间契约。
- 明确统一策略：采用“统一超集语义 + 子域映射”。
  - 例如 A2A `submitted` 在统一语义层映射为 `pending`。
- 强制 reason code 前缀规范：`team.*`、`workflow.*`、`a2a.*`。
- 统一 A2A 远端标识字段命名为 `peer_id`。
- 引入阻断级契约门禁（contract consistency gate），用于拦截 Teams/Workflow/A2A 变更中的共享契约漂移。
- 范围限定：本提案仅做 spec/doc/gate，不引入 runtime 功能代码。

## Capabilities

### Modified Capabilities
- `action-timeline-events`: 增加多代理 reason 前缀、状态映射、`peer_id` 字段命名规范。
- `runtime-config-and-diagnostics-api`: 增加多代理共享字段命名/配置命名空间一致性规范。
- `runtime-module-boundaries`: 增加共享契约阻断门禁要求与前置依赖约束。

## Impact

- 影响范围：
  - OpenSpec 规格：共享契约冻结与门禁要求。
  - 文档：`docs/multi-agent-identifier-model.md` 作为单一事实源补齐映射表。
  - 门禁：新增契约一致性检查脚本与主干索引登记（后续实施）。
- 不影响范围：
  - 不新增 Teams/Workflow/A2A runtime 行为。
  - 不变更现有 Run/Stream 执行路径。
