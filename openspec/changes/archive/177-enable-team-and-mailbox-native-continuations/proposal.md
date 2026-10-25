## Why

Baymax 已为 direct、child 与 workflow 提供了按 owner 证据开放的 durable continuation，但 team parallel/vote 和 mailbox host-plan 仍只能执行或保留 lifecycle history，不能在新进程中恢复同一逻辑 graph/message。现有 `HostPlanOutcome`、mailbox history、team task records 或重新 `Run` 都不是 continuation material；继续以它们模拟恢复会重复 effect、丢失 lease/attempt 因果关系，或错误宣传未实现的 capability。

现在需要在已归档的 source commit/projection recovery 基线之上，为 team 与 mailbox owner 建立真实、可验证、可拒绝的 pause/checkpoint/restore 路径，并将每个 form × pause-reason 的事实写回 manifest。

## What Changes

- 为 team parallel/vote source owner 增加 bounded continuation envelope、pause safe point、原子 claim、同机单写 fresh-process restore、duplicate projection 和 terminal/recovery evidence。
- 为 mailbox host-plan continuation 增加由既有 mailbox owner 保存的 message/delivery/lease/attempt continuation material；恢复复用原 mailbox 与原 logical message，不创建 queue、worker 或 scheduler。
- 支持 `input_required`、`approval_required`、`budget_exhausted` 三种 pause reason，但仅在具体 owner 的 source evidence、pending state、budget/deadline、effect references 和 restart fixture 全部完成时开放对应 capability cell。
- 将 lease loss、ack/nack/requeue/retry/DLQ、cancel、timeout 与 terminal arbitration 作为 mailbox continuation 的 source evidence；不把 lifecycle history 当 checkpoint。
- 更新 host capability manifest、planbridge admission 与公共文档，使 planner 在 activation 前获知 team/mailbox continuation 的支持范围、deployment scope 和 typed unsupported/recovery-required 边界。
- 添加 source-owner、fresh-process、duplicate/conflict、lease-loss/retry/DLQ、Run/Stream parity、read-only replay 和 PowerShell/shell contract gate 证据。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `composite-continuation-restore`: 将 team parallel/vote 与 mailbox host-plan continuation 从显式 unsupported 更新为 owner-evidenced checkpoint/restore，同时保留 resumed planbridge 的独立拒绝边界。
- `host-runtime-capability-manifest`: 为 team 与 mailbox form 增加基于 source owner evidence 的 checkpoint、restore、pause、mailbox continuation、deployment 与原因声明。
- `multi-agent-mailbox-contract`: 为既有 mailbox owner 增加 host-plan continuation material、restore claim、lease-aware recovery 与 terminal projection 的公开 contract。
- `teams-collaboration-runtime`: 为 team parallel/vote 增加 source-owned pause/checkpoint/restore、completed-member preservation 和 aggregate terminal arbitration contract。

## Impact

- 代码：`orchestration/teams`、`orchestration/mailbox`、`orchestration/planbridge`、`core/types` 及现有 checkpoint/continuation helpers。
- API：新增 team/mailbox continuation DTO、owner store/options、resume API 和 capability source projections；已有 Run/Stream 与 mailbox lifecycle API 保持兼容。
- 契约：公开支持面从“默认拒绝”变为“仅显式 owner 与同机单写 evidence 下可准入”；没有 source evidence 的 form/reason 继续 typed `unsupported_capability` 或 `recovery-required`。
- 依赖：不引入外部队列、数据库、scheduler、worker 或分布式 store；多写者与跨机器部署仍属于后续 `bound-durable-store-deployment-scope` 候选。

## Example Impact Assessment

修改示例

理由：更新现有 host execution plan 和 native execution events 示例的语义锚点，展示 owner-configured team/mailbox continuation、拒绝边界与恢复证据；不新增业务 payload、自动重跑或第二 mailbox owner。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `orchestration/teams/README.md`, `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md`, `examples/host-execution-plan/README.md`, `examples/native-execution-events/README.md`
- 维护 owner：Runtime contracts owner、team owner 与 mailbox owner maintainers
- 验证证据：owner unit/integration tests、fresh-process fixtures、Run/Stream parity、dedicated PowerShell/shell contract gates、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、质量门禁与 OpenSpec validation

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime contracts owner | source-owner boundary review |
| components | 修改文档 | `orchestration/teams/README.md`、`orchestration/mailbox/README.md`、`orchestration/planbridge/README.md` | team/mailbox maintainers | package continuation suites |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | explicit owner/store configuration review |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | strict OpenSpec validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | contract gate index review |
| examples | 修改文档 | `examples/host-execution-plan/README.md`、`examples/native-execution-events/README.md` | examples maintainer | expected-marker review |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | admission boundary review |
| best practices | 修改文档 | `README.md` | documentation maintainer | rollback and deployment-scope review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap consistency gate |
