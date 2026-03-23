## Why

A36 已补齐 mailbox lifecycle worker 基线，但当前 worker 在异常恢复能力上仍有空缺：handler panic 与长时间 in-flight 卡死缺少契约化收敛路径。  
在项目进入 0.x 收敛阶段后，需要把 mailbox worker 从“可运行”升级为“可恢复”，避免单点异常导致消息长期滞留或语义漂移。

## What Changes

- 新增 mailbox worker lease/reclaim/recover 契约：
  - 引入 `inflight_timeout` 与 `heartbeat_interval` 治理参数。
  - 支持 stale in-flight reclaim（默认在 consume 路径启用）。
  - 支持 handler panic recover，并按 worker 错误策略映射到一致生命周期动作。
- 扩展 mailbox worker 配置域：
  - `mailbox.worker.inflight_timeout=30s`
  - `mailbox.worker.heartbeat_interval=5s`
  - `mailbox.worker.reclaim_on_consume=true`
  - `mailbox.worker.panic_policy=follow_handler_error_policy`
- 扩展 lifecycle reason taxonomy，纳入 `lease_expired`（并保持 additive 扩展规则）。
- 扩展 lifecycle diagnostics 与 aggregates，覆盖 reclaim/heartbeat/panic-recovered 观测字段。
- 将 worker recover/reclaim suites 纳入 shared multi-agent gate 与 quality gate 阻断路径。

## Capabilities

### New Capabilities

- `mailbox-worker-lease-reclaim-contract`: 定义 mailbox worker lease/heartbeat/reclaim/panic-recover 的治理语义与默认值。

### Modified Capabilities

- `mailbox-lifecycle-worker-contract`: 扩展 worker 生命周期能力到 panic/reclaim，并更新 canonical reason taxonomy。
- `multi-agent-mailbox-contract`: 扩展 mailbox lifecycle 契约到 stale in-flight reclaim 与 heartbeat 保活语义。
- `runtime-config-and-diagnostics-api`: 增加 `mailbox.worker` 新配置字段与 reclaim/recover 诊断字段语义。
- `go-quality-gate`: 增加 worker recover/reclaim 契约套件与 taxonomy 漂移阻断。

## Impact

- 代码：
  - `orchestration/mailbox/*`（worker panic recover、lease heartbeat、reclaim 路径）
  - `runtime/config/*`（新字段解析、校验、热更新回滚）
  - `runtime/diagnostics/*`（reclaim/recover 事件、聚合、查询）
  - `orchestration/composer/*`（worker 参数接线与 lifecycle 观测收口）
  - `integration/*`（crash/reclaim/replay/run-stream parity 契约测试）
  - `scripts/check-multi-agent-shared-contract.*`
  - `scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `orchestration/README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 全部为 additive 配置与诊断扩展；
  - 默认行为保持保守，仍遵循 fail-fast + rollback 与 contract-first 边界。
