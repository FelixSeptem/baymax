## Context

当前代码路径已经支持异步子代理提交与回报收敛，但生命周期语义仍主要依附在 `running + lease` 模型上：
- async submit 被接受后，调用方获得 `AsyncAccepted`，后续终态依赖 report sink 回报；
- scheduler 任务状态缺少显式“等待回报”态，查询和诊断只能间接推断；
- 回报超时与晚到回报在主干契约层缺少统一可验证语义。

A30 正在统一 mailbox 协调面；A31 作为后续收口，专注于“异步子代理生命周期硬化”，不重复 A30 的 envelope 与 mailbox 基础能力。

## Goals / Non-Goals

**Goals:**
- 在调度域定义显式异步等待态 `awaiting_report` 及状态迁移。
- 固化 async-await 超时、晚到回报、重复回报与重放的确定性语义。
- 扩展 Task Board 查询与运行时诊断，提供可回归的一致口径。
- 保持 Run/Stream 语义等价与 memory/file 后端语义等价。
- 将上述语义纳入 shared quality gate 的阻断 contract suites。

**Non-Goals:**
- 不引入平台化控制面（UI/RBAC/多租户）。
- 不引入外部 MQ 或新的分布式存储依赖。
- 不改变 `runtime/*` 与 `mcp/*` 既有边界约束。
- 不做 exactly-once 承诺，保持 at-least-once + idempotent convergence。

## Decisions

### 1) 新增显式状态 `awaiting_report`
- 方案：在 scheduler 任务状态中新增 `awaiting_report`，用于表示“已异步接受、等待终态回报”。
- 原因：将“执行中”和“等待回报”解耦，避免运行中 lease 语义和回报语义混淆。
- 备选：继续复用 `running`。拒绝原因：查询和治理口径不清晰，易导致超时/重试误判。

### 2) async accepted 后切换为“等待回报跟踪”而非继续 lease 驱动
- 方案：异步接受后不再依赖 worker heartbeat 维持活跃执行语义，改由 `report_timeout` 控制等待窗口。
- 原因：异步任务真实执行发生在远端，继续沿用本地 lease 机制会引入伪超时和错误回收。
- 备选：保留 lease + heartbeat。拒绝原因：需要额外虚拟心跳机制，复杂且不稳定。

### 3) 超时终态采用确定性规则
- 方案：达到 `report_timeout` 后，默认进入 `failed`；若 DLQ 开启且达到策略条件，进入 `dead_letter`。
- 原因：与现有 scheduler 终态分类兼容，避免新增终态类型扩大迁移成本。
- 备选：新增 `timed_out` 终态。拒绝原因：会扩大现有状态枚举与兼容面，不利于本轮收口。

### 4) 晚到回报统一 `drop_and_record`
- 方案：任务已终态后收到晚到 report，统一丢弃业务写入，仅记录诊断与 timeline。
- 原因：避免终态翻转，保持最终一致性与幂等语义稳定。
- 备选：允许晚到回报覆盖终态。拒绝原因：会破坏确定性和回放稳定性。

### 5) 配置域新增 `scheduler.async_await.*`
- 方案：新增配置项并纳入 `env > file > default` 与 fail-fast 校验：
  - `scheduler.async_await.report_timeout`（默认 `15m`）
  - `scheduler.async_await.late_report_policy`（默认 `drop_and_record`）
  - `scheduler.async_await.timeout_terminal`（默认 `failed`）
- 原因：保证治理策略可配置且与 runtime/config 一致。
- 备选：硬编码策略。拒绝原因：不利于环境差异化与契约测试矩阵。

### 6) 诊断字段采用 additive 方式扩展
- 方案：新增聚合字段并遵循兼容窗口：`additive + nullable + default`。
  - `async_await_total`
  - `async_timeout_total`
  - `async_late_report_total`
  - `async_report_dedup_total`
- 原因：兼顾可观测性与现有消费者兼容。
- 备选：替换现有 async 指标。拒绝原因：会破坏历史指标语义与回放对比。

### 7) Gate 以契约测试阻断为准
- 方案：在 shared multi-agent gate 增加 async-await lifecycle suites，覆盖：
  - async accepted -> awaiting_report；
  - timeout terminalization；
  - late report drop_and_record；
  - dedup/replay idempotency；
  - Run/Stream 等价；
  - memory/file parity。
- 原因：确保 lifecycle 语义不会在后续演进中漂移。
- 备选：仅新增单测。拒绝原因：缺少跨模块与门禁级保障。

## Risks / Trade-offs

- [Risk] 新增状态会影响现有状态枚举消费方  
  → Mitigation: 以 additive 方式扩展，文档与查询契约同步，旧消费方可按未知状态降级处理。

- [Risk] A30 与 A31 并行期可能产生语义重叠  
  → Mitigation: A31 明确依赖 A30 完成后接入，A31 只处理 lifecycle，不重复 mailbox 基础契约。

- [Risk] timeout 与 retry/DLQ 策略耦合复杂  
  → Mitigation: 固化默认值与 deterministic 迁移规则，并以 contract tests 覆盖边界矩阵。

## Migration Plan

1. 以 A30 为前置，确认 mailbox 主路径稳定并完成旧接口收口。
2. 在 scheduler 状态机中引入 `awaiting_report` 并实现超时收敛路径。
3. 在 async 报告桥接中对齐 `late_report_policy=drop_and_record`。
4. 扩展 Task Board 查询 state 枚举及游标稳定语义覆盖。
5. 扩展 runtime/config 与 diagnostics 字段，保持 fail-fast 与兼容窗口约束。
6. 接入 shared multi-agent gate，并更新 mainline contract index 和文档映射。

回滚策略：
- 若 lifecycle 语义不稳定，可先回滚 gate 切换与新状态暴露；
- 保留旧 async 路径作为短期兜底，但不新增新契约承诺。

## Open Questions

无阻塞项，按推荐值冻结：
- `scheduler.async_await.report_timeout=15m`
- `scheduler.async_await.late_report_policy=drop_and_record`
- `scheduler.async_await.timeout_terminal=failed`

