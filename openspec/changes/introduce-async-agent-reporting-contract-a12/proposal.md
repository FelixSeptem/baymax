## Why

当前多代理链路已具备“异步执行 + 轮询等待”能力，但缺少“提交后独立回报”的一等契约，导致调用方仍需阻塞等待或自建轮询回收逻辑。A12 需要在 `library-first` 前提下补齐异步回报闭环，支持非阻塞编排并保持可回归契约。

## What Changes

- 新增统一异步提交与回报契约：`SubmitAsync` + `ReportSink`。
- 将“回报通道”与 `WaitResult` 解耦：支持提交后立即返回，并由独立回报路径上报终态。
- 定义回报投递语义：至少一次投递、幂等去重、失败重试窗口与终止条件。
- 提供最小内置回报 sink（in-memory channel + callback sink），并保持可扩展 sink 接口。
- 扩展 runtime config/diagnostics/timeline 字段，覆盖异步回报状态、重试、去重与丢弃原因。
- 将异步回报契约测试并入 shared multi-agent gate，覆盖 recovery 下回放一致性。

## Capabilities

### New Capabilities
- `multi-agent-async-reporting`: 定义多代理异步提交与独立回报通道契约（提交、投递、幂等、重试、可观测性）。

### Modified Capabilities
- `a2a-minimal-interoperability`: 增加 A2A 异步提交后的独立回报语义与兼容约束。
- `multi-agent-composed-orchestration`: 增加 composed 流程对异步回报闭环的编排契约。
- `multi-agent-lib-first-composer`: 增加 composer 子任务异步调度后回报汇聚语义。
- `distributed-subagent-scheduler`: 增加 scheduler 下异步回报回填与去重语义。
- `runtime-config-and-diagnostics-api`: 增加异步回报配置域与 additive 诊断字段。
- `action-timeline-events`: 增加异步回报 reason taxonomy 与关联字段约束。
- `go-quality-gate`: 增加异步回报契约测试并纳入共享门禁。

## Impact

- 代码：
  - `a2a/*`
  - `orchestration/composer/*`
  - `orchestration/scheduler/*`
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `observability/event/*`
- 测试：
  - `integration/*` 新增异步回报矩阵（成功、失败重试、幂等去重、recovery 一致性）
  - `scripts/check-multi-agent-shared-contract.*` 门禁扩展
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 同步路径保持可用；
  - 新能力默认关闭或仅在显式启用时生效；
  - 字段扩展遵循 `additive + nullable + default`。
