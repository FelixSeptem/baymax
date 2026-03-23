## Why

在 A35 将 mailbox 主链路接线后，当前仓库仍缺少可复用的 mailbox lifecycle worker 原语，`consume/ack/nack/requeue` 语义主要停留在底层存储测试，缺少统一运行时消费闭环与诊断可见性。  
为保证 mailbox 契约从“可用能力”收敛到“可默认治理能力”，需要引入库级 worker 契约并冻结生命周期 reason taxonomy。

## What Changes

- 新增 mailbox lifecycle worker 契约（library-first）：
  - 提供可选 worker loop 原语（默认关闭）。
  - 支持 `consume -> handler -> ack/nack/requeue` 标准执行闭环。
  - 默认 `handler_error_policy=requeue`。
  - 默认 `poll_interval=100ms`。
- 将 `mailbox.worker.*` 配置纳入 runtime config 与热更新校验：
  - `mailbox.worker.enabled=false`（默认关闭）
  - `mailbox.worker.poll_interval=100ms`
  - `mailbox.worker.handler_error_policy=requeue`
- 扩展 mailbox 诊断写入，覆盖 lifecycle 关键节点（consume/ack/nack/requeue/dead_letter/expired）。
- 冻结 mailbox lifecycle reason taxonomy，并将 taxonomy 漂移纳入阻断门禁。
- 在 shared multi-agent gate 与 quality gate 纳入 mailbox lifecycle worker contract suites。

## Capabilities

### New Capabilities

- `mailbox-lifecycle-worker-contract`: 定义 mailbox worker 执行闭环、默认策略与 reason taxonomy 语义。

### Modified Capabilities

- `multi-agent-mailbox-contract`: 将 mailbox lifecycle 从“存储能力”扩展为“worker 可执行闭环”契约。
- `runtime-config-and-diagnostics-api`: 扩展 `mailbox.worker.*` 配置域与 mailbox lifecycle 诊断字段语义。
- `go-quality-gate`: 新增 mailbox lifecycle worker suites 与 reason taxonomy 漂移阻断。

## Impact

- 代码：
  - `orchestration/mailbox/*`（worker loop + lifecycle hook）
  - `runtime/config/*`（mailbox.worker 配置、校验、热更新回滚）
  - `runtime/diagnostics/*`（mailbox lifecycle 记录与聚合）
  - `orchestration/composer/*`（可选 worker 托管接线）
  - `integration/*`（lifecycle worker 契约测试）
  - `scripts/check-multi-agent-shared-contract.*`
  - `scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `orchestration/README.md`
- 兼容性：
  - 新增字段与接口遵循 `additive + nullable + default`；
  - 默认关闭策略确保未启用场景行为保持稳定。
