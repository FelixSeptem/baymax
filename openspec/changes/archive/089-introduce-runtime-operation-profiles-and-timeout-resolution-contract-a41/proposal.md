## Why

当前多代理主链路已经具备 `scheduler.async_await`、`mailbox.worker`、`teams/workflow/a2a` 等分域超时契约，但缺少统一的跨域预算解析规则。调用方在同一请求内同时触发多层超时配置时，难以稳定判断最终生效值与来源，容易出现父子预算失配、终态收敛漂移与排障困难。

## What Changes

- 新增库级 `operation profile` 契约：在 `interactive|background|batch|legacy` 语义下提供统一的 timeout baseline 与解析入口。
- 定义跨域 timeout 解析优先级：`profile baseline -> domain override -> request override`，并固定父子任务预算收敛规则（子任务生效预算不得超过父任务剩余预算）。
- 为冲突配置与非法组合增加 fail-fast 校验，并与 readiness 语义对齐（可观测降级/阻断）。
- 扩展 diagnostics additive 字段，显式记录 `effective_operation_profile`、`timeout_resolution_source`、`timeout_resolution_trace`，支持 QueryRuns/TaskBoard 排障。
- 将跨域 timeout 解析与父子预算收敛套件接入 shared contract gate 与 quality gate（含 Run/Stream 等价、memory/file parity、replay idempotency）。

## Capabilities

### New Capabilities
- `runtime-operation-profiles-and-timeout-resolution-contract`: 定义 operation profile、跨域 timeout 优先级解析、父子预算收敛与可观测语义。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 `runtime.operation_profiles.*` 配置域与 timeout 解析相关诊断字段契约。
- `distributed-subagent-scheduler`: 增加父子任务 timeout 预算收敛与冲突 fail-fast 约束。
- `multi-agent-lib-first-composer`: 增加 operation profile 透传与生效预算摘要输出约束。
- `go-quality-gate`: 增加 cross-domain timeout resolution contract suites 的阻断映射。

## Impact

- 代码：
  - `runtime/config/*`（operation profiles 配置加载/校验/热更新回滚）
  - `orchestration/scheduler/*`（预算解析、父子收敛、冲突判定）
  - `orchestration/composer/*`（profile 传递与摘要透传）
  - `runtime/diagnostics/*`（additive 字段、query 输出）
  - `integration/*`（timeout resolution 合同测试矩阵）
  - `scripts/check-multi-agent-shared-contract.*`
  - `scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
- 兼容性：
  - 默认 `profile=legacy`，保持历史默认行为；
  - 新增字段遵循 `additive + nullable + default`；
  - 不引入平台化控制面与外部 MQ 依赖。
