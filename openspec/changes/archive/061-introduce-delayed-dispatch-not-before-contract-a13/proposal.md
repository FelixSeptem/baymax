## Why

当前调度链路只有 retry backoff 语义下的 `next_eligible_at`，缺少业务级“延后可执行”能力，无法以统一契约表达“任务在某个时间点之后再开始领取”。A13 需要补齐 `not_before` 调度语义，完成三类 agent 间通信方式闭环（同步、异步、定时延后）。

## What Changes

- 在 scheduler task 模型中新增业务级延后字段（推荐 `not_before`）。
- 统一 claim 可领取判定：未到 `not_before` 的任务不可领取，到期后按既有 FIFO/priority 规则参与调度。
- 明确 `not_before` 与 retry backoff 的边界：`not_before` 控制首次可领取时机，重试仍由现有 backoff 策略治理。
- 在 memory/file backend 中持久化与恢复 `not_before`，保证恢复后不提前执行。
- 在 composer child dispatch 路径提供延后调度接入（library-first）。
- 增加 delayed dispatch 的 timeline reason、run diagnostics additive 字段与契约测试。

## Capabilities

### New Capabilities
- `multi-agent-delayed-dispatch`: 定义 scheduler/composer 的业务级延后执行契约（`not_before`、可领取判定、恢复一致性）。

### Modified Capabilities
- `distributed-subagent-scheduler`: 增加 delayed claim 语义与恢复一致性要求。
- `multi-agent-lib-first-composer`: 增加 child dispatch 延后执行契约。
- `runtime-config-and-diagnostics-api`: 增加 delayed dispatch 相关配置/诊断字段契约。
- `action-timeline-events`: 增加 delayed dispatch reason taxonomy 与关联字段要求。
- `go-quality-gate`: 增加 delayed dispatch 合同测试并纳入共享门禁。

## Impact

- 代码：
  - `orchestration/scheduler/*`
  - `orchestration/composer/*`
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `observability/event/*`
- 测试：
  - `integration/*`（准点可领取、提前不可领取、恢复后不漂移、Run/Stream 等价）
  - shared multi-agent contract gate 扩展
- 文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- 兼容性：
  - `not_before` 为空时行为保持现状；
  - 新字段与新统计保持 `additive + nullable + default`。
