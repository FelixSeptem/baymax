# Orchestration

## Purpose

`orchestration` 与 `a2a` 组合 workflow、teams、scheduler、composer 和跨 agent mailbox。

## Owned facts/state

节点关系、调度策略、重试/backoff、mailbox 和组合终态由 orchestration owner 管理。

## Inputs/outputs

输入为已定义的 agent/tool capabilities 与 workflow graph；输出为子运行结果、事件和聚合终态。

## Dependencies and forbidden dependencies

可依赖 runner 抽象、context、observability；不得嵌入具体 Provider SDK 或 transport 实现。

## Lifecycle / failure / rollback

graph admission → schedule → execute → reconcile。子运行失败需带 owner 和 retryability；聚合失败可回滚到 checkpoint。

## Observability

记录 node、attempt、mailbox 和聚合决策，沿用 RuntimeRecorder 单写入口。

## Verification / source links

`go test ./orchestration/... ./a2a/... -count=1`；[orchestration README](../../orchestration/README.md)。
