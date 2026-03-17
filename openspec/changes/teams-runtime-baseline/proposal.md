## Why

当前仓库已经具备多代理示例（`07/08`），但缺少可复用的 Teams 一等抽象。业务侧需要在每个应用里重复实现角色分工、任务分发、结果收敛与冲突处理，导致行为不稳定、观测口径不一致，也阻塞后续 Workflow 与 A2A 能力接入。

现在启动 Teams runtime baseline，可以在不破坏现有 runner 契约的前提下，把“单次 run”扩展到“多代理协作 run”的可维护形态，并沉淀统一字段与诊断规范。

## What Changes

- 新增 Teams 运行时基线能力：角色模型（leader/worker/coordinator）与协作策略（serial/parallel/vote）。
- 定义 Teams 任务生命周期与收敛语义，统一失败、超时、取消传播行为。
- 为 Teams 增加统一标识与观测字段（`team_id`、`agent_id`、`task_id`、`role`、`strategy`）并约束映射关系。
- 扩展 runtime 配置与 diagnostics 契约，确保 `env > file > default`、fail-fast 校验、single-writer 观测口径延续。
- 约束模块边界：Teams 编排不直接挤入 `core/runner` 主状态机，实现通过独立编排模块接入。

## Capabilities

### New Capabilities
- `teams-collaboration-runtime`: 提供多代理角色建模、协作策略执行与任务状态收敛的运行时基线。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 Teams 配置与 run 级诊断字段契约。
- `action-timeline-events`: 增加 Teams 协作路径相关的 timeline 元数据与 reason 语义。
- `runtime-module-boundaries`: 增加 Teams 编排模块与 `core/runner` 的边界约束。

## Impact

- 影响代码：
  - `core/*`（请求/响应与元数据传播点）
  - 新增编排模块（建议 `orchestration/teams`，最终路径以实施稿为准）
  - `runtime/config`、`runtime/diagnostics`、`observability/event`
- 影响测试：
  - Teams 策略行为测试（serial/parallel/vote）
  - Run/Stream 等价契约测试
  - 诊断幂等与回放测试
- 影响文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/runtime-module-boundaries.md`
  - `docs/diagnostics-replay.md`
  - 新增统一字段说明文档（`docs/multi-agent-identifier-model.md`）
