## Why

A15 将 workflow 表达能力推进到“图可复用”，但多代理协作语义（handoff/delegation/aggregation）仍分散在 `teams/workflow/composer` 层内实现，缺少统一的一等契约。A16 目标是在 `library-first` 前提下补齐统一协作原语，收敛执行语义、可观测口径与门禁治理，降低后续扩展成本。

## What Changes

- 新增统一协作原语能力包：`orchestration/collab`，首期提供 `handoff`、`delegation`、`aggregation`。
- 固化聚合策略首版范围：`all_settled`、`first_success`，默认 `all_settled`。
- 固化失败策略：默认 `fail_fast`。
- 固化重试策略：协作原语层默认不自带重试，沿用现有 scheduler/retry 治理链路。
- 统一接入 composer/workflow/teams 协作路径，保持现有主行为兼容。
- feature flag 默认关闭，显式启用后生效。
- timeline reason 不新增顶级命名空间，沿用既有 canonical namespace（`team.*` / `workflow.*` / `a2a.*` / `scheduler.*`）。
- 扩展 runtime diagnostics additive 字段与 shared contract gate，确保语义/文档/测试一致。

## Capabilities

### New Capabilities
- `multi-agent-collaboration-primitives`: 定义 handoff/delegation/aggregation 的统一库级协作原语契约。

### Modified Capabilities
- `teams-collaboration-runtime`: 引入协作原语接入与语义对齐要求。
- `workflow-deterministic-dsl`: 引入 workflow 场景下 delegation/handoff/aggregation 的确定性编排约束。
- `multi-agent-lib-first-composer`: 增加 composer 对统一协作原语的消费契约。
- `action-timeline-events`: 增加协作原语 reason 与关联字段约束（复用既有命名空间）。
- `runtime-config-and-diagnostics-api`: 增加协作原语 feature flag、策略字段与 additive 诊断字段契约。
- `go-quality-gate`: 增加协作原语合同矩阵并纳入 shared gate 阻断。

## Impact

- 代码：
  - `orchestration/collab/*`（new）
  - `orchestration/teams/*`
  - `orchestration/workflow/*`
  - `orchestration/composer/*`
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `observability/event/*`
  - `tool/contributioncheck/*`
  - `scripts/check-multi-agent-shared-contract.*`
- 测试：
  - `integration/*` 协作原语合同矩阵（sync/async/delayed + Run/Stream）
  - replay-idempotent 与 recovery 组合回归
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 默认关闭（feature flag）；
  - 新增字段遵循 `additive + nullable + default`；
  - 不引入平台化控制面能力。
