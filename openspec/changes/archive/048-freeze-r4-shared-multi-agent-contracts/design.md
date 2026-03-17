## Context

R4 的 Teams/Workflow/A2A 三个变更都在定义各自的 lifecycle、timeline reason、diagnostics 字段。若先实现再收敛，会把兼容成本转移到运行时代码和历史观测数据。当前应先在 spec 层建立共享契约，再允许实现进入。

## Goals / Non-Goals

**Goals:**
- 冻结共享契约并形成阻断级门禁。
- 统一 lifecycle 语义层与子域映射规则。
- 统一 reason code 命名空间与字段命名（含 `peer_id`）。
- 统一配置命名空间和诊断字段最小约束，避免跨变更冲突。

**Non-Goals:**
- 不实现 Teams/Workflow/A2A 运行时模块。
- 不改变现有 action timeline 基础状态枚举。
- 不引入新的跨进程协议行为。

## Decisions

### 1) 共享状态采用“统一语义层 + 子域映射”
- 方案：统一语义层继续使用现有 timeline 规范（`pending|running|succeeded|failed|skipped|canceled`）。
- 子域状态允许保留 raw 语义，但必须可映射到统一语义层。
- 明确映射：A2A `submitted -> pending`。
- 原因：避免破坏既有 timeline 枚举，同时满足 A2A 语义表达。

### 2) reason code 强制前缀分域
- 方案：仅允许 `team.*`、`workflow.*`、`a2a.*` 三类前缀进入多代理新增 reason code。
- 原因：防止跨域冲突，便于趋势聚合与统计分桶。

### 3) A2A 远端标识字段统一为 `peer_id`
- 方案：多代理相关 specs/docs/gate 统一使用 `peer_id`，不再引入别名字段。
- 原因：降低消费者适配分叉与后续迁移成本。

### 4) 共享契约门禁设为阻断级
- 方案：新增 contract consistency gate，作为 Teams/Workflow/A2A 进入实现前置条件。
- 原因：把冲突发现前置到 spec/CI，而不是后置到实现与回归阶段。

### 5) 本变更限定为 spec/doc/gate
- 方案：本变更不触达 runtime 功能逻辑，仅冻结口径与治理机制。
- 原因：收敛变更面，快速解除并行 spec 冲突风险。

## Risks / Trade-offs

- [Risk] 阻断门禁可能拉长短期交付节奏
  - Mitigation: 收敛范围仅限共享契约，不引入运行时代码。

- [Risk] 子域状态映射表达不充分
  - Mitigation: 保留 raw 状态字段在子域内部使用，但统一对外聚合语义。

- [Risk] 现有草案中已有命名差异
  - Mitigation: 先做 spec 对齐，再进入实现，避免重复返工。

## Migration Plan

1. 冻结共享契约（status mapping / reason namespace / field naming）。
2. 更新三条未完成变更的相关 spec 文案，确保一致性。
3. 增加阻断级门禁脚本并纳入主干契约索引。
4. 通过 `openspec validate` 后，放行 Teams/Workflow/A2A 实施。

## Open Questions

- 当前范围内无阻断级 open questions。
