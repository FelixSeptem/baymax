# Orchestration

## Purpose

`orchestration` 与 `a2a` 组合 workflow、teams、scheduler、composer 和跨 agent mailbox。

## Owned facts/state

节点关系、调度策略、重试/backoff、mailbox 和组合终态由 orchestration owner 管理。

## Inputs/outputs

输入为已定义的 agent/tool capabilities 与 workflow graph；输出为子运行结果、事件和聚合终态。

## Host execution plan bridge

`orchestration/planbridge` 提供 opt-in `host_execution_plan.v1`：版本化 descriptor + 独立 native binding → bounded Compile/全计划 preflight → 一个既有 owner 的 Run/Stream。它不建立通用 graph executor；composer.Run/Stream 仍只委托 runner。详情见 [package README](../../orchestration/planbridge/README.md) 与 [离线示例](../../examples/host-execution-plan/README.md)。

首版目标为 direct、teams parallel/vote、flat workflow 和隔离独占 scheduler 的 local child；shared-queue dispatch、mailbox delivery、resumed/native resume、跨 owner durable continuation 与组合 dynamic pause 明确不支持。direct input_required 保留 pause，原生 budget/retry/aggregation/checkpoint/terminal 继续由各 source owner 管理；native task terminal 不代表业务 commit。

宿主提供 mode choice、authority/tool digests、业务授权、write scope、revision、effect key 和 commit gate。plan digest 只证明规范化描述一致；actual activation 仍需 live admission/parent budget clamp。真实事件沿用 `host/eventadapter`，RuntimeRecorder 单写；不新增全局顺序、事件游标或恢复状态机。交付证据与限制见 [design](../../openspec/changes/establish-host-execution-plan-bridge-contract/design.md)、[tasks](../../openspec/changes/establish-host-execution-plan-bridge-contract/tasks.md) 与 [示例文档基线](../../examples/host-execution-plan/README.md)。

## Dependencies and forbidden dependencies

可依赖 runner 抽象、context、observability；不得嵌入具体 Provider SDK 或 transport 实现。

## Lifecycle / failure / rollback

graph admission → schedule → execute → reconcile。子运行失败需带 owner 和 retryability；聚合失败可回滚到 checkpoint。

## Observability

记录 node、attempt、mailbox 和聚合决策，沿用 RuntimeRecorder 单写入口。

## Verification / source links

`go test ./orchestration/... ./a2a/... -count=1`；[orchestration README](../../orchestration/README.md)。
