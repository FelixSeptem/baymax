## 1. Shared Contract Freeze

- [x] 1.1 在 `docs/multi-agent-identifier-model.md` 补齐统一状态语义层与子域映射表（明确 `a2a.submitted -> pending`）。
- [x] 1.2 在同一文档补齐 reason code 命名空间规范（仅 `team.*|workflow.*|a2a.*`）。
- [x] 1.3 统一并固定 A2A 远端标识字段命名为 `peer_id`，消除同义字段表述。

## 2. Spec Delta Alignment

- [x] 2.1 在 `action-timeline-events` 增加共享契约要求：状态映射、reason 前缀、`peer_id` 命名。
- [x] 2.2 在 `runtime-config-and-diagnostics-api` 增加共享契约要求：多代理配置命名空间不重叠、诊断字段命名一致。
- [x] 2.3 在 `runtime-module-boundaries` 增加阻断级前置要求：未通过共享契约门禁不得进入 Teams/Workflow/A2A 实施。

## 3. Blocking Gate

- [x] 3.1 新增多代理共享契约一致性检查脚本（至少覆盖状态映射、reason 前缀、`peer_id` 命名）。
- [x] 3.2 将门禁脚本登记到主干契约索引与文档，作为 required-check 候选。
- [x] 3.3 增加脚本测试用例（正例/反例）并验证在本地可复现执行。

## 4. Cross-Change Consistency

- [x] 4.1 对 `teams-runtime-baseline`、`workflow-dsl-baseline`、`a2a-minimal-interoperability` 执行 spec 文案对齐，确保共享契约一致。
- [x] 4.2 运行 `openspec validate` 校验上述变更与本变更的一致性。
- [x] 4.3 输出一致性检查结论（冲突项清单为 0）后，放行后续功能实现提案。
