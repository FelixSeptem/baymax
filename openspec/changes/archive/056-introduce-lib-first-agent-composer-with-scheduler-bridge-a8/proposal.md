## Why

A5/A6/A7 已完成多代理编排与调度基线，但宿主接入仍需手工拼装 `runner + workflow + teams + a2a + scheduler`，导致配置消费、可观测口径和回归语义容易漂移。现在需要一个 `library-first` 的组合入口，把现有能力收敛为可直接复用、可验证的统一契约。

## What Changes

- 新增 `orchestration/composer` 组合层（新包），提供多代理运行时的一体化接入入口。
- 将 `teams.*`、`workflow.*`、`a2a.*`、`scheduler.*`、`subagent.*` 配置快照统一映射到组合执行路径。
- 引入 scheduler-managed subagent bridge，支持 `local child-run` 与 `a2a child-run` 双路径。
- 约束 scheduler 初始化失败时按策略降级到 `memory` backend，并记录标准化事件与诊断信号。
- 固化组合层 `run.finished` additive 摘要字段注入契约，保持 `RuntimeRecorder` 单写入口不变。
- 固化 Run/Stream 组合路径语义一致（终态类别与聚合字段等价），并补回归门禁。
- 更新示例为 composer 主路径接入范式，确保文档、代码、契约测试一致。

## Capabilities

### New Capabilities
- `multi-agent-lib-first-composer`: 定义 `orchestration/composer` 组合入口、接缝 API、调度桥接与语义约束。

### Modified Capabilities
- `multi-agent-composed-orchestration`: 由“模块可组合”升级为“组合层一体化接入”契约。
- `distributed-subagent-scheduler`: 增加组合层桥接、后端降级策略与子任务执行收口语义。
- `runtime-config-and-diagnostics-api`: 增加 composer 配置消费与 run 摘要注入契约。
- `action-timeline-events`: 增加 composer 管理路径的 reason/correlation 契约要求。
- `runtime-module-boundaries`: 明确 composer 与 runner/orchestration/runtime 的依赖与职责边界。
- `go-quality-gate`: 将 composer 组合契约测试纳入现有阻断门禁路径。

## Impact

- 代码：`orchestration/composer/*`（新）、`orchestration/scheduler/*`、`orchestration/teams/*`、`orchestration/workflow/*`、`a2a/*`、`runtime/config/*`、`observability/event/*`、`integration/*`、`examples/*`。
- 测试：新增 composer 集成契约（Run/Stream 等价、scheduler takeover、idempotency、fallback-to-memory、config reload 边界）。
- 文档：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/mainline-contract-test-index.md`、`docs/v1-acceptance.md`、`docs/development-roadmap.md`。
- 兼容性：保持 additive + nullable + default 兼容窗口，不移除既有字段或既有模块 API。
