## Purpose

本能力定义一套版本化、有界、只读、可离线回放的**派生预算投影**契约：把既有预算 owner（ReAct iteration/tool-call limit、runtime budget admission、scheduler 父预算、Plan Notebook）各自拥有的事实快照，归一化为一份可供模型决策上下文消费的**有界只读剩余预算投影**，并配套计算型、确定性的预算利用 benchmark。它让「预算增加是否改善完成质量，还是只增加重复循环」成为可被 fixture 捕获、可被 gate 阻断、可被 benchmark 比较的显式证据，而不是散落在四个 owner 内部的隐式状态。

本能力**不拥有任何预算事实**：它只消费 source-owned facts 的快照并派生只读视图，不建立第二本预算账本，不新增配置键，不改变终态判定。

## ADDED Requirements

### Requirement: 派生预算投影 SHALL 由版本化有界契约表达

系统 MUST 提供版本化、provider-neutral 的派生预算投影契约 `budget_projection.v1`，用于表达一次决策点上的剩余预算事实视图。契约 MUST 同时包含 `facts`（source-owned 事实快照）与 `projection`（派生输出），并 MUST 为投影提供确定性 canonical 序列化与 SHA-256 digest。

契约 MUST NOT 依赖或引入 `runtime/*`、`orchestration/*`、`model/*` 或 `observability/*` 的任何实现包；MUST NOT 记录 raw reasoning、完整 transcript、工具输出正文、credentials 或任何无界 payload；文本类事实 MUST 只以枚举、计数器与 digest 表达。

#### Scenario: 契约对同一事实快照产生稳定投影
- **WHEN** 同一 `BudgetFacts` 被投影两次
- **THEN** canonical 序列化与 digest 完全一致，且不产生任何副作用

#### Scenario: 契约超界时确定性失败
- **WHEN** notes 条数、单条长度、projection 字节数或 benchmark 规模超出契约边界
- **THEN** 契约以稳定的 overflow 分类 fail-fast，且不返回截断后的部分成功结果

#### Scenario: 未知版本被拒绝
- **WHEN** 解析到的 fixture 版本不是 `budget_projection.v1`
- **THEN** 解析失败并返回 unknown version 分类，不尝试兼容未知版本

### Requirement: 缺失维度 SHALL 按 nullable + default 表达且不得被伪造

投影 MUST 为 iteration、tool-call、time、cost 四个维度分别表达 `available` 标志。任一维度缺少事实（缺少 limit、缺少 used、缺少时间预算或缺少 cost 阈值）时，该维度 MUST 输出 `available=false` 且 MUST NOT 填充剩余值。契约 MUST NOT 把缺失解释为"剩余为 0"或"预算充足"。

历史 fixture 缺少新增字段 MUST 解析成功并按缺省处理；未知字段 MUST 被安全忽略，且 MUST NOT 影响投影、digest 或 drift 分类。`Complete` MUST 仅在四个维度全部 available 时为 true。

#### Scenario: 时间预算缺失不被解释为剩余为零
- **WHEN** facts 未提供时间预算或已用时间
- **THEN** 时间维度输出 `available=false` 且无剩余值，`Complete=false`，且分类码为 `budget_facts_missing_time_budget`

#### Scenario: 历史 fixture 缺字段仍可解析
- **WHEN** 一个历史 fixture 未包含新增的维度字段
- **THEN** 解析成功并按缺省语义处理，不产生未知版本或 schema 分类

#### Scenario: 未知字段被安全忽略
- **WHEN** fixture 包含契约未定义的字段
- **THEN** 解析成功且该字段不影响投影、digest 或 drift 分类

### Requirement: 投影 SHALL 只读且不得形成第二本预算账本

投影 MUST 是 source-owned facts 的纯派生函数：MUST NOT 读时钟、MUST NOT 读全局或运行时状态、MUST NOT 产生副作用，MUST NOT 提供任何回写事实源的入口。pressure level MUST 只作为派生表达，MUST NOT 参与终态判定；终态仍由 ReAct termination 常量与 admission decision 各自拥有。

系统 MUST NOT 因本能力新增预算配置键、新增预算计数器类型、修改 admission 阈值/维度/降级动作、修改 scheduler `ParentRemainingBudget` 语义，或修改 Plan Notebook lifecycle。

#### Scenario: 投影是纯函数
- **WHEN** 同一 facts 在无时钟与无状态环境下被投影
- **THEN** 结果可重复且与调用顺序、调用次数无关

#### Scenario: 回写形状被阻断
- **WHEN** 契约或被测投影引入可写回事实源的字段或设置器
- **THEN** 校验以 `budget_projection_writeback_shape_detected` 失败

#### Scenario: 终态语义未被投影改变
- **WHEN** 投影给出 `saturated` pressure
- **THEN** 该表达不触发、不替代、不提前任何既有终态判定路径

### Requirement: 预算利用 benchmark SHALL 确定性且可离线重算

系统 MUST 提供由 fixture 轨迹驱动的预算利用 benchmark：每条 trace 由若干步骤组成，每步携带预算事实快照与步骤结果（`progress` / `repeat` / `terminal_success` / `terminal_exhausted`）。benchmark MUST 计算 `completion_rate`、`repeated_loop_rate`、`budget_utilization`（按维度稳定排序）与 `saturation_rate`，并 MUST 为结果提供 canonical 序列化与 digest。

benchmark MUST NOT 调用模型、MUST NOT 依赖时钟或网络、MUST NOT 产生运行时副作用。snapshot/recovery 后的重算 MUST 与首次计算完全一致。

#### Scenario: 指标可重算且幂等
- **WHEN** 同一 benchmark fixture 被计算两次
- **THEN** 四项指标与 digest 完全一致

#### Scenario: 恢复后重算不一致被阻断
- **WHEN** 同一 fixture 二次解析并重算后 digest 与首次不同
- **THEN** 校验以 `budget_benchmark_recovery_recompute_drift` 失败

#### Scenario: 期望指标与实际不符被阻断
- **WHEN** fixture 声明的期望指标与计算结果不一致
- **THEN** 校验以 `budget_benchmark_metric_mismatch` 失败

#### Scenario: benchmark 规模超界被阻断
- **WHEN** trace 数超过 32 或单 trace 步骤数超过 64
- **THEN** 校验以 `budget_benchmark_overflow_drift` 失败，不返回部分结果

### Requirement: 派生预算投影 SHALL 可回放且分类稳定

系统 MUST 提供离线、只读、确定性的 replay，解析 `budget_projection.v1` 并校验投影与 fixture 期望、benchmark 指标与重算结果。Replay MUST NOT 调用模型、provider、工具或修改运行时状态。

Replay MUST 至少分类以下稳定码：`budget_projection_schema_drift`、`budget_projection_unknown_version`、`budget_facts_missing_iteration_limit`、`budget_facts_missing_tool_call_limit`、`budget_facts_missing_time_budget`、`budget_facts_missing_cost_threshold`、`budget_projection_negative_remaining`、`budget_projection_ratio_out_of_range`、`budget_projection_pressure_level_mismatch`、`budget_projection_note_unbounded`、`budget_projection_digest_mismatch`、`budget_projection_overflow_drift`、`budget_projection_writeback_shape_detected`、`budget_benchmark_schema_drift`、`budget_benchmark_metric_mismatch`、`budget_benchmark_recovery_recompute_drift`、`budget_benchmark_overflow_drift`、`budget_projection_replay_not_idempotent`。码集合 MUST 与契约常量、spec、包 README 与 gate 文档一致。

#### Scenario: 合法 fixture 可离线回放
- **WHEN** replay 收到合法且 digest 自洽的 `budget_projection.v1` fixture
- **THEN** replay 成功完成，不触发模型/网络调用、不修改运行时状态

#### Scenario: 重复回放幂等
- **WHEN** 同一 fixture 被重复回放
- **THEN** 归一化结果与 drift 分类完全一致

#### Scenario: 分类词表漂移被阻断
- **WHEN** drift 码集合与契约常量或文档不再一致
- **THEN** 对等 gate 失败并给出 taxonomy drift 分类
