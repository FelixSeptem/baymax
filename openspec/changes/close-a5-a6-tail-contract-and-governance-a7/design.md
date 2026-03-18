## Context

A5（composed orchestration）与 A6（distributed scheduler）是主能力建设迭代，A7 目标不是扩功能，而是确保这些新增能力在主干长期可维护：
- 字段不会在高并发/回放下无界膨胀；
- reason 命名与关联字段稳定可检查；
- shared-contract gate 能阻断漂移；
- CI 对关键故障恢复路径有独立门禁。

当前仓库已有 shared multi-agent contract gate 与 single-writer 观测基线，本变更基于现有机制扩展，不引入平行治理体系。

## Goals / Non-Goals

**Goals:**
- 定义多代理新增字段的 bounded-cardinality 与兼容窗口契约。
- 固化 scheduler/subagent reason taxonomy 与 attempt-level 追踪契约。
- 扩展 shared-contract gate 为 A5/A6 收口提供阻断能力。
- 增加 scheduler crash-recovery/takeover 的独立质量门禁。
- 收敛文档与主干契约索引，避免代码先行后文档漂移。

**Non-Goals:**
- 不新增调度主能力（queue/lease/dispatch）本身。
- 不改写 A5/A6 主接口模型和模块边界。
- 不引入控制平面功能（租户、RBAC、审计门户）。

## Decisions

### 1) 兼容窗口以“字段可选 + additive 默认值”固定
- 方案：新增字段必须可空、可缺省，不影响旧消费者。
- 原因：降低升级风险，允许客户端分阶段消费。
- 备选：强制升级。拒绝原因：对既有集成破坏性过高。

### 2) bounded-cardinality 通过运行时聚合与 gate 双重约束
- 方案：运行时约束计数粒度 + gate 校验字段集合与聚合策略。
- 原因：仅靠实现容易回归，必须有治理层阻断。
- 备选：文档约定。拒绝原因：缺乏执行力。

### 3) scheduler/subagent reason 采用 canonical taxonomy
- 方案：固定最小 reason 集合并纳入 shared-contract 检查。
- 原因：避免后续迭代 reason 命名漂移导致观测不可比。
- 备选：放开 reason。拒绝原因：指标与诊断失去可比性。

### 4) crash-recovery 契约门禁独立运行
- 方案：新增 scheduler 专项 gate（可并入现有 CI 但独立 job）。
- 原因：故障恢复链路是分布式调度稳定性的核心风险点。
- 备选：仅在全量 go test 中覆盖。拒绝原因：信号不聚焦，回归定位慢。

## Risks / Trade-offs

- [Risk] 收口约束过严导致短期开发成本上升  
  → Mitigation: 先定义最小必须集，保持渐进扩展。

- [Risk] gate 过多影响 CI 时长  
  → Mitigation: 重点场景抽样 + 分层执行（PR 必跑最小集，夜间跑全量）。

- [Risk] 文档同步滞后导致实施歧义  
  → Mitigation: 把文档更新纳入 A7 tasks 必选项，并在 gate 中检查索引一致性。

## Migration Plan

1. 先固化 A7 spec（bounded-cardinality、taxonomy、gate 要求）。
2. 在 `runtime/diagnostics` 与 `tool/contributioncheck` 实现对应约束。
3. 在 CI 引入/扩展 scheduler recovery gate 并补测试数据集。
4. 更新 docs 与 contract index，完成一次全量回归。

回滚策略：
- 保留已有功能路径，仅回退 A7 附加 gate 与收口字段约束；
- 不回退 A5/A6 主能力。

## Open Questions

- scheduler recovery gate 是独立脚本还是复用 multi-agent shared-contract 脚本分阶段执行？
- bounded-cardinality 的默认上限值是否需要按环境 profile 暴露可调参数？
