## Why

A29 正在补齐 Task Board 只读查询，但多代理协作的消息协调面仍分散在 `A11(A2A sync submit+wait)`、`A12(async report sink)`、`A13(not_before delayed dispatch)` 三套 API 中，导致语义重复、门禁分散、扩展成本高。

项目尚未对外使用，当前是一次合适窗口：在 `library-first` 边界内统一 mailbox 契约并收敛接口，不再为旧 API 保留兼容负担。

## What Changes

- 新增 mailbox 统一消息契约（envelope + lifecycle）：`command/event/result`、`ack/nack/retry/ttl/dlq/idempotency-key`。
- 新增 mailbox 只读查询能力：过滤、排序、分页、opaque cursor，支持排障与编排可观测消费。
- 在 `orchestration/mailbox` 提供统一 API：发布、消费、确认、重试、查询、快照恢复。
- 新增 `mailbox.*` 配置域并纳入 `runtime/config`（`env > file > default` + fail-fast 校验）。
- 新增 mailbox 诊断字段与查询入口，补齐与 scheduler/task/run 的关联键。
- 将 shared multi-agent gate 接入 mailbox contract suites，作为阻断项。
- **BREAKING**：A11/A12/A13 旧调用面（分散的 sync/async/delayed API）进入 deprecate 路径，主线迁移为 mailbox 统一契约，不再承诺旧 API 兼容。

## Capabilities

### New Capabilities
- `multi-agent-mailbox-contract`: 定义统一 mailbox envelope、交付语义、重试/过期/DLQ、查询与恢复契约。

### Modified Capabilities
- `multi-agent-sync-invocation-contract`: 同步调用收敛到 mailbox command->result 路径，旧 submit+wait API 可废弃。
- `multi-agent-async-reporting`: 异步回报收敛到 mailbox result 交付语义，旧 report-sink 直连 API 可废弃。
- `multi-agent-delayed-dispatch`: 延后执行语义收敛到 mailbox envelope 的 `not_before/expire_at`。
- `runtime-config-and-diagnostics-api`: 新增 `mailbox.*` 配置与 mailbox 诊断查询入口。
- `go-quality-gate`: 新增 mailbox contract suites 阻断要求。

## Impact

- 代码：
  - `orchestration/mailbox/*`（新包）
  - `orchestration/invoke/*`、`orchestration/collab/*`、`orchestration/scheduler/*`（接缝迁移与旧 API deprecate）
  - `runtime/config/*`、`runtime/diagnostics/*`（配置与诊断扩展）
  - `scripts/check-multi-agent-shared-contract.*`（门禁接入）
- 测试：
  - mailbox 单测与 integration contract suites（幂等、重试、TTL、DLQ、query、memory/file 一致性）
  - A11/A12/A13 对应旧路径测试迁移到 mailbox 主路径
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `orchestration/README.md`、`a2a/README.md`（语义更新与 deprecate 说明）
