# Development Roadmap

更新时间：2026-03-18

## 定位

Baymax 的主线定位是 `library-first + contract-first`：
- 优先交付可嵌入 Go runtime 能力，而不是平台化控制面。
- 所有行为变化以 OpenSpec + 契约测试驱动。
- 文档、代码、测试必须同一节奏更新。

## 现状快照（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更实时口径：`openspec list --json`
- 已归档口径：`openspec/changes/archive/INDEX.md`

截至 2026-03-18：
- 已归档并稳定：R1-R3 主线能力、A4-A9。
- 已完成（未归档）：`introduce-scheduler-qos-fairness-and-deadletter-governance-a10`（25/25）。

## 当前代码能力（简版）

已稳定主干：
- Runner Run/Stream 统一语义（含并发与背压基线）。
- 多 Provider（OpenAI/Anthropic/Gemini）与能力探测/降级。
- Context Assembler CA1-CA4。
- 安全治理 S1-S4。
- 多代理基础编排：`teams` / `workflow` / `a2a` / `scheduler` / `composer`。

已落代码但默认关闭：
- Composer 恢复域 `recovery.*`（A9）：`memory|file` 后端、冲突策略 `fail_fast`、恢复摘要字段。

进行中能力：
- A10 调度治理：QoS/公平性/DLQ/退避策略。

## 活跃变更

### A9（已归档）

目标：收敛跨会话恢复与确定性重放语义（已完成并归档）。

### A10（已完成，待归档）

目标：补齐 scheduler 治理能力，不改变 lib-first 定位。

已确认设计约束：
- 默认调度：`fifo`。
- 优先级来源：task 字段。
- 公平窗口：`3`。
- DLQ 默认：关闭。
- 退避策略：指数退避 + 抖动。

完成定义（DoD）：
- 配置、实现、诊断、timeline reason、契约测试、文档六项同步完成。
- Run/Stream 等价语义不回退。
- `check-multi-agent-shared-contract` 与 `contributioncheck` 通过。

## 下一阶段提案池（Agent 间通信三种方式全覆盖）

### A11（同步执行并等待结果：语义收敛与库接口固化）

对应能力：`1. 同步执行并等待结果`

现状（已具备）：
- A2A 已支持 `Submit -> WaitResult`。
- workflow/teams/composer 的 remote 路径可同步等待子任务终态。

建议变更：
- 固化统一同步调用接口（library-first）：统一 `timeout/cancel/error-layer` 语义，避免 teams/workflow/composer 各自拼装。
- 明确同步路径契约：终态判断、重复查询幂等、Run/Stream 等价输出字段。
- 新增“同步调用最小 API”示例，作为主线推荐路径。

DoD：
- 同步调用 API、错误分层、超时/取消语义文档化且可回归。
- 契约测试覆盖 `teams + workflow + composer + scheduler a2a adapter` 的同步等待一致性。

### A12（异步执行后汇报：从轮询补齐到主动回报）

对应能力：`2. 异步执行后汇报`

现状（部分具备）：
- 支持异步执行（服务端后台执行）。
- 支持轮询获取结果与 `WaitResult` 内部 callback 重试。
- 尚未形成“提交后独立主动回报通道”作为一等能力。

建议变更：
- 引入统一异步提交与回报契约：`SubmitAsync` + `ReportSink`（至少支持内存 channel + callback sink）。
- 回报通道与 `WaitResult` 解耦，支持“提交后不阻塞主流程，完成后主动汇报”。
- 增加回报投递幂等键与重试窗口，保证“至少一次 + 可去重”语义。

DoD：
- 异步回报路径具备配置、诊断字段与 timeline reason。
- 合同测试覆盖：成功回报、回报失败重试、重复回报去重、进程重启后回报一致性（结合 recovery）。

### A13（定时延后执行：调度时间语义）

对应能力：`3. 定时延后执行`

现状（未具备）：
- 当前仅有 retry backoff 驱动的 `next_eligible_at`，不等价于业务定时调度。
- 无 `not_before/execute_at` 公开任务字段与契约。

建议变更：
- 在 scheduler task 引入显式时间字段（推荐 `not_before`），用于业务级延后执行。
- claim 语义明确区分：`queued-now` 与 `queued-not-before`。
- 持久化后端（memory/file）统一保存/恢复时间语义，避免恢复后提前执行。

DoD：
- 配置与文档明确时钟基准、精度、边界行为（过去时间、空值、极端值）。
- 契约测试覆盖：准点可领取、提前不可领取、恢复后定时语义不漂移、Run/Stream 统计口径一致。

### 推荐推进顺序（通信能力主线）

1. A11：先把已存在的同步能力做接口与契约收敛。
2. A12：补齐异步主动回报，形成非阻塞协作闭环。
3. A13：补齐业务级延后调度，完成三种通信方式全覆盖。

## Lib-First Multi-Agent 差距收敛清单（2026-03-18）

P0（必须优先）：
- [x] A9 恢复能力主干落地。
- [x] A9 文档与契约收口后归档。
- [x] A10 调度治理落地并验收。
- [ ] recovery + qos 端到端合同测试矩阵补齐。
- [ ] 新增 reason taxonomy 与 run summary 解析语义统一（`additive + nullable + default`）。

P1（紧随其后）：
- [ ] workflow 图能力增强（subgraph/复用节点/条件模板）。
- [ ] multi-agent 协作原语增强（handoff/delegation/aggregation）。
- [ ] 长任务恢复边界收敛（resume 边界、in-flight 不回溯、超时重入策略）。
- [ ] 按 run/team/workflow/task 的统一检索 API（库接口优先）。
- [ ] 多代理主链路性能基线纳入 CI（吞吐/延迟/重试放大/recovery 时间）。

P2（可选，DX/生态）：
- [ ] 最小 replay CLI 工具链。
- [ ] 全链路示例（team + workflow + a2a + scheduler + recovery）。
- [ ] 外部适配样板与迁移映射文档。

推荐顺序：
1. A10 归档。
2. P1 编排能力与协作原语。
3. P1 观测与性能回归门禁。
4. P2 DX/生态补充。

## 非近期范围（为保持简洁，明确延后）

以下能力不进入当前主线：
- 多租户控制面、RBAC、审计平台化。
- Web 控制台与运营平台。
- 全局调度控制面与跨租户负载均衡。

说明：相关方向保留在长期 `R4 platformization` 议题，不影响当前 lib-first 路线。

## 执行规则

- 每次只推进一个 OpenSpec change，避免并行大改导致口径漂移。
- 提案到实现必须遵循顺序：`proposal/design/spec/tasks -> code -> tests -> docs`。
- 合并前最少验证：
  - `go test ./...`
  - `go test -race ./...`
  - `pwsh -File scripts/check-docs-consistency.ps1`
  - `pwsh -File scripts/check-multi-agent-shared-contract.ps1`
