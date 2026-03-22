## Why

当前仓库已经具备完整的 `mailbox.*` 配置域与 mailbox 契约测试，但主执行链路仍大量按调用临时创建 `in-memory` bridge，导致配置能力与运行时行为未完全对齐。  
在 A33/A34 之后，下一步应把 mailbox 从“契约能力”收敛为“运行时默认可接线能力”，避免配置漂移与观测盲区。

## What Changes

- 在编排主链路引入共享 mailbox runtime wiring（优先 Composer 管理路径），替代每次调用临时 `NewInMemoryMailboxBridge()` 的模式。
- 将 `runtime/config` 的 `mailbox.*` 字段接入 mailbox 初始化与刷新逻辑：
  - `mailbox.enabled=false` 时使用共享 memory mailbox；
  - `mailbox.enabled=true` 时按 `backend/path/retry/ttl/query` 初始化；
  - `backend=file` 初始化失败时回退 `memory` 并记录 fallback 诊断。
- 为 mailbox publish/query 路径接入运行时诊断写入，确保 `QueryMailbox` / `MailboxAggregates` 反映真实编排链路数据，而非仅测试写入。
- 扩展 shared multi-agent gate 与 quality gate，新增 mailbox runtime wiring 契约阻断（配置接线、fallback、Run/Stream 等价、memory/file 语义一致）。
- 同步更新 README、roadmap、runtime config/diagnostics 与 orchestration 文档，删除 mailbox “配置存在但主链路未接线”的中间态描述。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `multi-agent-mailbox-contract`: 增加“共享 mailbox runtime 接线与配置驱动初始化”要求，约束主链路不得退回临时 per-call bridge 中间态。
- `runtime-config-and-diagnostics-api`: 将 `mailbox.*` 从“可解析字段”升级为“运行时有效接线 + 可观测诊断”的契约要求。
- `go-quality-gate`: 纳入 mailbox runtime wiring 契约阻断，覆盖配置生效、fallback 行为、diagnostics/query traceability。

## Impact

- 代码：
  - `orchestration/composer/*`
  - `orchestration/collab/*`
  - `orchestration/scheduler/*`
  - `orchestration/invoke/*`
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `integration/*`
  - `scripts/check-multi-agent-shared-contract.*`
  - `scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/development-roadmap.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `orchestration/README.md`
- 兼容性：
  - API 以 additive 为主，不新增平台化依赖；
  - 行为上将从“临时 per-call mailbox”收敛为“共享 runtime mailbox”，属于主线语义收口。
