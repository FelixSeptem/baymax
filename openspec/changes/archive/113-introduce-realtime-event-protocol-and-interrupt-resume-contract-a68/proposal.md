## Why

当前 A67 正在实施，ReAct 计划治理路径已进入收敛阶段；下一顺位的缺口是 realtime 双向事件协议与 interrupt/resume 合同。现有 Run/Stream 已具备基础流式输出，但对实时事件信封、序列推进、去重幂等、恢复游标和中断恢复边界缺少统一 contract，导致跨入口实现与回放验证容易漂移。按 roadmap 顺序起草 A68，可一次性收口 realtime 同域需求，避免后续拆分平行提案。

## What Changes

- 新增 A68 主合同：realtime event protocol + interrupt/resume contract。
- 新增 realtime 事件合同：
  - canonical event envelope（`event_id/session_id/run_id/seq/type/ts/payload`）；
  - canonical taxonomy（request/delta/interrupt/resume/ack/error/complete）；
  - 顺序与幂等语义（单调 `seq`、去重键、ack 语义）。
- 新增 interrupt/resume 合同：
  - interrupt 冻结规则、resume 游标恢复规则；
  - 重放幂等（重复 interrupt/resume 不膨胀 counters）。
- 新增配置域：
  - `runtime.realtime.protocol.*`
  - `runtime.realtime.interrupt_resume.*`
- 新增 QueryRuns additive 字段（最小集）：
  - `realtime_protocol_version`
  - `realtime_event_seq_max`
  - `realtime_interrupt_total`
  - `realtime_resume_total`
  - `realtime_resume_source`
  - `realtime_idempotency_dedup_total`
  - `realtime_last_error_code`
- 新增 replay fixture：`realtime_event_protocol.v1`，并冻结 drift taxonomy：
  - `realtime_event_order_drift`
  - `realtime_interrupt_semantic_drift`
  - `realtime_resume_semantic_drift`
  - `realtime_idempotency_drift`
  - `realtime_sequence_gap_drift`
- 新增 gate：`check-realtime-protocol-contract.sh/.ps1`，并接入 `check-quality-gate.*`。
- 新增边界断言：`realtime_control_plane_absent`（禁止平台化实时网关/托管连接控制面）。
- 一次性收口约束：A68 同域需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在 A68 增量吸收，不再新增平行 realtime 提案。

## Capabilities

### New Capabilities
- `realtime-event-protocol-and-interrupt-resume-contract`: realtime 双向事件协议与 interrupt/resume 合同。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 `runtime.realtime.protocol.*` / `runtime.realtime.interrupt_resume.*` 与 A68 additive 字段。
- `react-loop-and-tool-calling-parity-contract`: 在 realtime interrupt/resume 场景保持 Run/Stream 语义等价。
- `diagnostics-replay-tooling`: 增加 `realtime_event_protocol.v1` fixture 与 A68 drift 分类断言。
- `go-quality-gate`: 增加 realtime contract gate 与 `realtime_control_plane_absent` 边界断言。

## Impact

- 代码：
  - `core/runner`（interrupt/resume 状态推进与事件发射边界）
  - `runtime/config`（A68 配置解析、校验、热更新回滚）
  - `runtime/diagnostics`、`observability/event`（A68 additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（A68 fixtures + drift tests）
  - `scripts/check-realtime-protocol-contract.*` + `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
- 兼容性与边界：
  - 对外 API 不引入 breaking 变更；新增字段遵循 `additive + nullable + default`。
  - A68 必须复用 A58/A67 解释链字段与 A56 loop 终止 taxonomy，不新增平行 loop 或平行决策语义。
  - 保持 `library-first`：不引入托管实时网关、托管连接路由或平台化实时控制面。
