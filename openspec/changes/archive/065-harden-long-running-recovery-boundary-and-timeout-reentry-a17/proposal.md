## Why

当前恢复能力已覆盖 `memory|file`、`fail_fast` 与 replay-idempotent，但在长任务链路上仍缺少严格的“恢复边界”契约：`resume` 边界、`in_flight` 处理和 timeout 后重入策略尚未统一定义。A17 的目标是在不平台化的前提下收敛这些边界，避免协作原语与恢复链路叠加后出现语义漂移。

## What Changes

- 新增长任务恢复边界契约，定义 `next_attempt_only`、`no_rewind` 与 timeout 重入规则。
- 固化 timeout 重入策略为 `single_reentry_then_fail`，并限制每 task 最大重入次数为 `1`。
- 固化恢复边界启用语义：`recovery.enabled=true` 时自动启用边界策略。
- 延续冲突策略 `fail_fast`，并把边界冲突纳入同一失败分类与门禁口径。
- 在 scheduler/composer/workflow 恢复路径统一接入边界判定，不改写既有核心状态机。
- 扩展 timeline/diagnostics additive 字段，覆盖恢复边界命中、重入次数、拒绝原因。
- 不新增顶级 reason namespace，复用 `recovery.*` 与 `scheduler.*` canonical reasons。
- 扩展 shared contract gate 与 integration matrix，覆盖 crash/restart/replay/timeout 组合场景。

## Capabilities

### New Capabilities
- `long-running-recovery-boundary`: 定义长任务恢复边界、in-flight 语义与 timeout 重入治理契约。

### Modified Capabilities
- `multi-agent-session-recovery`: 增加恢复边界与重入策略强约束。
- `distributed-subagent-scheduler`: 增加恢复边界下 claim/commit/replay 行为约束。
- `multi-agent-lib-first-composer`: 增加 composer 恢复接入边界策略与语义等价约束。
- `runtime-config-and-diagnostics-api`: 增加恢复边界配置与 additive 诊断字段契约。
- `action-timeline-events`: 增加恢复边界 timeline reason 与关联字段约束。
- `go-quality-gate`: 增加恢复边界合同矩阵与 shared gate 阻断规则。

## Impact

- 代码：
  - `orchestration/composer/*`
  - `orchestration/scheduler/*`
  - `orchestration/workflow/*`
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `observability/event/*`
  - `tool/contributioncheck/*`
  - `scripts/check-multi-agent-shared-contract.*`
- 测试：
  - `integration/*` 新增恢复边界矩阵（crash/restart/replay/timeout）
  - Run/Stream 语义等价 + replay-idempotent + timeout 重入边界
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - 新增字段遵循 `additive + nullable + default`；
  - 不引入平台化控制面；
  - 默认恢复关闭时行为不变。
