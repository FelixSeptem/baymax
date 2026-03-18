## Why

当前多代理链路已具备组合编排与调度基线，但跨会话崩溃恢复与确定性重放仍缺少统一契约，导致重启后的状态收敛与副作用幂等依赖调用方自行保证。A9 需要把恢复能力收敛为 `library-first` 的标准运行时语义，并纳入主干阻断门禁。

## What Changes

- 新增 composer 级会话恢复能力，定义统一 `RecoveryStore` 抽象并提供 `memory|file` 两种后端。
- 定义恢复最小单元：run envelope、workflow checkpoint、scheduler task/attempt 状态、A2A in-flight 状态与 replay cursor。
- 提供恢复入口（resume/recover）并固化重放幂等语义，避免重复提交/重复终态放大聚合计数。
- 约束默认恢复策略为 `disabled`，显式启用后生效。
- 约束恢复冲突处理为 `fail_fast`（快照与实时状态不一致时立即中止并输出标准错误）。
- 增加恢复路径 timeline 与 run 摘要 additive 字段（兼容窗口维持 `additive + nullable + default`）。
- 将恢复契约测试并入现有 `check-multi-agent-shared-contract.*` 阻断门禁。

## Capabilities

### New Capabilities
- `multi-agent-session-recovery`: 定义多代理组合链路的跨会话恢复、冲突处理与确定性重放契约。

### Modified Capabilities
- `multi-agent-composed-orchestration`: 增加组合链路的 resume/recover 入口与跨会话一致性要求。
- `distributed-subagent-scheduler`: 增加 scheduler attempt 恢复、重放幂等与恢复冲突 fail-fast 语义。
- `workflow-deterministic-dsl`: 增加 workflow checkpoint 与组合恢复协同契约。
- `a2a-minimal-interoperability`: 增加 A2A in-flight 状态在恢复路径中的收敛契约。
- `runtime-config-and-diagnostics-api`: 增加 recovery 配置域、默认关闭策略与恢复观测字段要求。
- `action-timeline-events`: 增加 recovery/replay 相关 reason 与关联字段约束。
- `runtime-module-boundaries`: 明确 recovery 组件与 diagnostics single-writer 的依赖与职责边界。
- `go-quality-gate`: 增加跨会话恢复与确定性重放契约测试到共享阻断门禁。

## Impact

- 代码：`orchestration/*`、`a2a/*`、`runtime/config/*`、`runtime/diagnostics/*`、`observability/event/*`、`integration/*`、`tool/contributioncheck/*`、`scripts/check-multi-agent-shared-contract.*`。
- 测试：新增恢复/重放契约测试（重启恢复、重复重放、冲突 fail-fast、Run/Stream 语义一致）。
- 文档：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/mainline-contract-test-index.md`、`docs/v1-acceptance.md`、`docs/development-roadmap.md`。
- 兼容性：新增字段与能力均采用 additive + nullable + default，不破坏现有消费者。
