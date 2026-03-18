## Why

当前仓库已具备 Teams/Workflow/A2A 的单模块能力，但仍缺少跨进程调度协调底座，`v1` 也明确标注“无分布式编排与跨进程协调”。A6 需要补齐可持久化、可接管、可回放的 subagent 调度基线，把 multi-agent 从“可组合”推进到“可运营”。

## What Changes

- 新增分布式 subagent 调度基线模块，提供任务入队、租约领取、心跳续租、租约过期接管与结果回传。
- 引入调度幂等契约（submit/claim/result）与去重语义，保证重试与 replay 下不重复执行关键副作用。
- 引入父子 run 协调治理基线（最大深度、最大并发子任务、时间预算继承/耗尽终止），避免失控 fanout。
- 将 A2A 互联路径与调度基线打通：A2A 负责 peer 协作语义，调度器负责跨进程任务状态与 lease 协调。
- 扩展 runtime config、timeline reason、run diagnostics 字段，覆盖 queue/lease/child-run 关键可观测信号。
- 增加跨进程调度契约测试与故障恢复回归（worker crash、lease expire、重复提交、重复回放）。

## Capabilities

### New Capabilities
- `distributed-subagent-scheduler`: 定义分布式 subagent 调度语义（queue + lease + heartbeat + takeover + idempotency + parent-child run guardrails）。

### Modified Capabilities
- `a2a-minimal-interoperability`: 将 A2A 从最小互联扩展为可被调度基线消费的远端协作执行域。
- `runtime-config-and-diagnostics-api`: 增加 scheduler/subagent 治理配置与 additive 诊断字段契约。
- `action-timeline-events`: 增加 scheduler/subagent 调度原因码与跨进程关联字段契约。
- `runtime-module-boundaries`: 增加 scheduler 模块职责边界，明确与 A2A/MCP/runner 的依赖方向。

## Impact

- 影响代码：`orchestration/scheduler/*`（新）、`a2a/*`、`runtime/config/*`、`runtime/diagnostics/*`、`observability/event/*`、`integration/*`。
- 影响测试：新增/更新 lease 接管、幂等去重、父子 run 预算耗尽、Run/Stream 等价、A2A+scheduler 组合回归。
- 影响文档：`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/v1-acceptance.md`。
- 兼容性：以 additive 为主；默认保持现有本地执行路径可用，scheduler 功能通过配置启用。
