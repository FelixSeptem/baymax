## ADDED Requirements

### Requirement: Replay SHALL support budget projection fixtures

Diagnostics replay MUST 解析版本化 `budget_projection.v1` fixture，覆盖四个预算维度（iteration / tool-call / time / cost）的齐全与缺失、negative remaining、ratio 越界、pressure level 一致性、notes 有界性、digest 自洽、unknown version、未知字段兼容，以及预算利用 benchmark 的指标重算与恢复重算。Replay MUST 离线、只读、确定性，MUST 与历史 fixture 版本兼容，MUST NOT 调用模型、provider、工具或修改运行时状态。

#### Scenario: 合法预算投影 fixture 可回放
- **WHEN** replay 收到合法且 digest 自洽的 `budget_projection.v1` fixture
- **THEN** replay 成功完成，不触发网络调用、不修改运行时状态、不产生副作用

#### Scenario: 非法 fixture 快速失败
- **WHEN** 版本、维度事实、pressure、notes、benchmark 规模或 digest 字段缺失或非法
- **THEN** replay 以确定性 schema 校验失败返回，且不产生部分成功结果

#### Scenario: 重复回放幂等
- **WHEN** 同一 fixture 被重复回放
- **THEN** 归一化结果与 drift 分类完全一致，计数器与源状态不增长

### Requirement: Replay SHALL classify budget projection drift canonically

Replay MUST 至少分类以下稳定码：`budget_projection_schema_drift`、`budget_projection_unknown_version`、`budget_facts_missing_iteration_limit`、`budget_facts_missing_tool_call_limit`、`budget_facts_missing_time_budget`、`budget_facts_missing_cost_threshold`、`budget_projection_negative_remaining`、`budget_projection_ratio_out_of_range`、`budget_projection_pressure_level_mismatch`、`budget_projection_note_unbounded`、`budget_projection_digest_mismatch`、`budget_projection_overflow_drift`、`budget_projection_writeback_shape_detected`、`budget_benchmark_schema_drift`、`budget_benchmark_metric_mismatch`、`budget_benchmark_recovery_recompute_drift`、`budget_benchmark_overflow_drift`、`budget_projection_replay_not_idempotent`。码集合 MUST 与 `context/budgetprojection` 常量、spec、包 README 与本 gate 文档一致。

#### Scenario: 预算投影漂移被分类
- **WHEN** 归一化投影与 fixture 期望在维度可用性、剩余值、pressure、notes 或 digest 上不一致
- **THEN** replay 返回对应的稳定 drift 分类

#### Scenario: 分类词表漂移被阻断
- **WHEN** drift 码集合与契约常量或文档不再一致
- **THEN** 对等 gate 失败并给出 taxonomy drift 分类

### Requirement: Budget replay SHALL keep derived projection read-only and bounded

Replay MUST 拒绝任何把预算投影回写事实源的形状，MUST 保证缺失维度按 nullable + default 表达，MUST 保证 notes、projection 字节数与 benchmark 规模不超界，且 MUST NOT 因 replay 引入第二本预算账本、新增配置键或落盘 raw reasoning / transcript / 无界 payload。

#### Scenario: 缺失维度不得被伪造
- **WHEN** fixture 缺少某维度事实而期望投影给出了剩余值
- **THEN** replay 返回对应维度缺失分类并失败

#### Scenario: 投影超界被阻断
- **WHEN** 投影字节数、notes 条数或 benchmark 规模超出契约上界
- **THEN** replay 返回 overflow 分类并失败，不返回截断结果

#### Scenario: 回放不产生诊断写入
- **WHEN** replay 处理任意 fixture
- **THEN** 不写入 `runtime/diagnostics`、不绕过 `RuntimeRecorder` 单写入口、不落盘事实源快照正文
