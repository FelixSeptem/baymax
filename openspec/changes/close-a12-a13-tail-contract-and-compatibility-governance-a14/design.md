## Context

A12 已完成异步回报能力，A13 正在补齐 `not_before` 延后调度；两者叠加后，主线能力从“可用”进入“可治理”阶段。当前主要风险不是缺功能，而是多处契约边界可能出现漂移：
- shared contract gate 尚未把 delayed reason 作为强制冻结项；
- async/delayed 组合场景在 Run/Stream、qos、recovery 下缺少统一矩阵口径；
- run summary 对 A12/A13 additive 字段的兼容窗口未形成闭环 parser 语义；
- docs/index 与代码状态可能产生滞后。

A14 采用“收口治理提案”方式，不新增平台能力、不改变 A12/A13 业务行为，只做契约、门禁、文档与回归矩阵的收敛。

## Goals / Non-Goals

**Goals:**
- 固化 A12/A13 合并后的 shared taxonomy 与必需关联字段约束。
- 建立同步/异步/延后 × Run/Stream × qos/recovery 的最小跨模式合同矩阵。
- 固化 A12/A13 additive 字段的 `additive + nullable + default` 兼容窗口与 parser 语义。
- 使 gate、contract index、roadmap、runtime 文档与代码状态在同一提案内收敛。

**Non-Goals:**
- 不新增 A2A/scheduler/composer 的业务功能。
- 不引入 MQ、控制面、平台化治理系统。
- 不改写 A12/A13 已确定的终态语义与失败语义。

## Decisions

### 1) A14 定位为“收口治理”而非“能力扩展”
- 方案：只做 contract/gate/doc/index 收敛，不新增业务开关或新执行路径。
- 原因：A12/A13 已覆盖通信能力，当前瓶颈是可回归性与语义漂移风险。
- 备选：继续叠加新能力。拒绝原因：会在收口前扩大变更面，增加回归不确定性。

### 2) 共享 reason 冻结采用单一门禁来源
- 方案：以 shared multi-agent contract gate 作为 A12/A13 canonical reason 与关联字段的统一阻断入口。
- 原因：避免“代码、spec、脚本各自维护一套规则”导致分叉。
- 备选：分散到多个 gate。拒绝原因：维护成本高，故障定位慢。

### 3) 跨模式矩阵采用“最小覆盖优先”
- 方案：优先覆盖三类通信方式（sync/async/delayed）在 Run/Stream 和 qos/recovery 下的关键组合，而非全排列。
- 原因：保证 CI 成本可控，同时阻断高风险语义漂移。
- 备选：全排列矩阵。拒绝原因：测试成本高且收益递减。

### 4) 兼容窗口语义强制落到 parser 契约
- 方案：对 A12/A13 additive 字段明确“缺失=默认值、未知字段=忽略、旧字段语义不变”。
- 原因：仅文档声明不足以保障消费者升级安全，必须有契约测试约束。
- 备选：仅在 README 说明。拒绝原因：无法阻断回归。

### 5) A14 对 A12/A13 采用冻结点依赖
- 方案：A14 任务默认依赖 A12/A13 进入 contract freeze（至少 spec/tasks 冻结）。
- 原因：避免 A14 与在建实现相互覆盖、反复改口径。
- 备选：并行推进。拒绝原因：容易造成提案与实现错位。

## Risks / Trade-offs

- [Risk] A12/A13 仍在变动导致 A14 冻结点反复修改  
  → Mitigation: 明确“先冻结再收口”的依赖顺序，并把变更集中在 gate 与 docs 映射层。 

- [Risk] 跨模式矩阵增加 CI 时长  
  → Mitigation: 采用最小关键组合，保留可扩展矩阵而非一次性全覆盖。 

- [Risk] 兼容窗口规则过于宽松，掩盖真实回归  
  → Mitigation: 增加语义等价断言（旧字段语义不变）与 replay-idempotent 联合检查。 

- [Risk] 文档同步遗漏导致“测试通过但文档漂移”  
  → Mitigation: 在质量门禁中保留 docs consistency + contract index traceability 检查。

## Migration Plan

1. 冻结 A12/A13 合并后的 reason taxonomy、summary 字段与关联字段清单。  
2. 扩展 shared multi-agent gate：纳入 delayed reasons 与 cross-mode 基础矩阵检查。  
3. 补 integration 合同矩阵（sync/async/delayed × Run/Stream × qos/recovery 关键组合）。  
4. 增加 parser 兼容窗口合同测试（字段缺失/新增/默认值）。  
5. 同步更新 `docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`。  
6. 以统一质量门禁验证并归档。  

回滚策略：
- 保留 A12/A13 功能实现，仅回退 A14 新增 gate/matrix/doc 收口约束；
- 兼容窗口字段保持 additive，不反向删除。

## Open Questions

- A12/A13 的最终冻结点以“归档前最后一次 spec 版本”为准，还是以“实现 PR 合并时点”为准（建议：归档前最后一次 spec 版本）。
- cross-mode 最小矩阵中 recovery 组合是否先覆盖 scheduler/composer 主路径，其他路径后续补充（建议：先覆盖主路径）。
