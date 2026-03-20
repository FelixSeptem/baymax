## Why

A30 正在统一 mailbox 协调面，但异步子代理在 `async accepted` 到最终回报之间仍缺少显式生命周期语义。当前任务主要停留在 `running` 语义，导致查询、超时治理、晚到回报处理和诊断口径不够确定，容易产生跨模块理解偏差。

## What Changes

- 新增异步子代理生命周期契约，定义 `awaiting_report` 状态及其状态迁移规则。
- 固化 async-await 路径的超时规则：达到 `report_timeout` 后进入确定性终态（默认失败，启用 DLQ 时按策略进入 `dead_letter`）。
- 固化晚到回报（late report）处理：不改变已收敛业务终态，统一执行 `drop_and_record` 诊断记录。
- 固化重复回报与重放行为：保持幂等收敛，不膨胀逻辑终态与聚合计数。
- 扩展任务查询契约：Task Board 可按 `awaiting_report` 过滤并与现有分页/游标语义保持一致。
- 扩展运行时配置与诊断字段，增加 async-await 生命周期治理与可观测指标。
- 将 async-await 生命周期 contract suites 接入 shared multi-agent gate 作为阻断项。

## Capabilities

### New Capabilities
- `multi-agent-async-await-lifecycle-contract`: 定义异步子代理从 accepted 到 report commit 的状态机、超时、晚到回报与幂等收敛语义。

### Modified Capabilities
- `distributed-subagent-scheduler`: 扩展 scheduler 任务状态与终态收敛规则以覆盖 async-await 生命周期。
- `multi-agent-async-reporting`: 将异步回报契约对齐到 awaiting-report 生命周期与 late-report 策略。
- `multi-agent-task-board-query-contract`: 扩展任务状态过滤枚举与查询语义，覆盖 `awaiting_report` 状态。
- `runtime-config-and-diagnostics-api`: 新增 async-await 配置域与诊断聚合字段契约。
- `go-quality-gate`: 将 async-await 生命周期 contract suites 纳入共享阻断门禁。

## Impact

- 代码：
  - `orchestration/scheduler/*`（状态机、超时处理、查询过滤）
  - `orchestration/composer/*`（async accepted 进入 awaiting-report 的桥接语义）
  - `orchestration/invoke/*`、`orchestration/collab/*`（异步调用契约对齐）
  - `runtime/config/*`、`runtime/diagnostics/*`（配置与诊断扩展）
  - `observability/event/*`（生命周期事件与统计口径）
  - `integration/*`（跨后端 parity 与 Run/Stream 等价契约）
- 测试：
  - 新增 async-await 生命周期单测 + integration contract suites。
  - 新增 late report / timeout / dedup / replay 回归用例。
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`

