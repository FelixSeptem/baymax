## Why

当前 roadmap 与 examples 计划已定义 R3 高阶并发示例（05-07），但仓库实际仅有 01-04，存在文档与可运行示例不一致。现在补齐最小可运行示例，可以在不改动核心 runtime 行为的前提下，快速完善并发/异步模式的学习与验证路径。

## What Changes

- 新增 R3 高阶示例：
  - `examples/05-parallel-tools-fanout`
  - `examples/06-async-job-progress`
  - `examples/07-multi-agent-async-channel`
  - `examples/08-multi-agent-network-bridge`（从 07 拆分出的网络通信最小演示，采用 JSON-RPC 2.0 消息协议）
- 所有新增示例统一接入 `runtime/config.Manager`，输出结构化事件与可观测字段。
- 为 06/07/08 增加并发与异步通信场景下的结构化 event 输出样例（stdout JSON）。
- 更新 README，新增按 PocketFlow Pattern 的示例导航索引表，并同步运行说明。
- 更新 `docs/examples-expansion-plan.md` 与 `docs/development-roadmap.md`，标注 R3 示例批次扩容与拆分后的目录映射。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `tutorial-examples-expansion`: 扩展 R3 高阶示例集合（05-08），并增加 runtime manager 接入与结构化 event 输出要求。

## Impact

- 受影响目录：`examples/05-*`、`examples/06-*`、`examples/07-*`、`examples/08-*`、`README.md`、`docs/examples-expansion-plan.md`、`docs/development-roadmap.md`、`docs/v1-acceptance.md`。
- 不引入核心行为变更：`core/runner`、`runtime/*`、`model/*` 契约保持不变。
- 测试策略：本期以“可编译、可运行”为验收，不新增高成本集成测试矩阵。
