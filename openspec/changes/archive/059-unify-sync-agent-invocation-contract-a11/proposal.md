## Why

当前同步远程调用语义分散在 `workflow/teams/composer/scheduler` 多条路径上，虽然都基于 `a2a Submit + WaitResult`，但在超时、取消、错误分层、重试提示等细节上仍存在实现层重复与漂移风险。A11 需要将同步调用契约收敛为可复用库能力，降低后续 A12/A13 扩展成本并提升回归可控性。

## What Changes

- 新增统一同步调用抽象（建议落点 `orchestration/invoke`），封装 `Submit -> WaitResult -> terminal normalize`。
- 固化同步调用默认行为：`poll_interval` 默认值、`context` 取消优先级、终态返回约束、错误分层归一规则。
- 将 `workflow StepKindA2A`、`teams TaskTargetRemote`、`composer ChildTargetA2A`、`scheduler ExecuteClaimWithA2A` 接入统一同步调用抽象。
- 保持 callback 为兼容性可选钩子，不在 A11 引入独立异步主动回报通道。
- 增补跨模块契约测试矩阵，阻断同步语义漂移（超时/取消/错误层/Run-Stream 等价）。
- 同步更新 README 与多代理契约文档，确保代码与文档语义一致。

## Capabilities

### New Capabilities
- `multi-agent-sync-invocation-contract`: 定义跨 workflow/teams/composer/scheduler 的统一同步调用语义与错误归一契约。

### Modified Capabilities
- `a2a-minimal-interoperability`: 增加 `WaitResult` 与共享同步契约的语义一致性要求（终态、取消、超时、错误层）。
- `multi-agent-composed-orchestration`: 增加 composed remote 路径统一同步调用契约约束。
- `multi-agent-lib-first-composer`: 增加 composer A2A child-run 同步调用契约与终态收敛要求。
- `distributed-subagent-scheduler`: 增加 scheduler A2A 适配路径与共享同步契约对齐要求。
- `go-quality-gate`: 增加同步调用契约回归测试并纳入 shared multi-agent gate。

## Impact

- 代码：
  - `orchestration/invoke/*`（新增）
  - `orchestration/workflow/*`
  - `orchestration/teams/*`
  - `orchestration/composer/*`
  - `orchestration/scheduler/*`
- 测试：
  - `integration/*` 组合编排契约矩阵补充
  - `orchestration/*` 相关单测更新
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/runtime-module-boundaries.md`（仅当包边界说明需补充时）
- 兼容性：
  - 不新增破坏性配置项；
  - 语义按 `additive + nullable + default` 兼容窗口收敛。
