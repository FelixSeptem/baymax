# Design: 预算感知派生上下文投影

## 背景与定位

预算事实已有四个独立 owner，各自拥有不同的维度与语义，且**互不知道对方的剩余量**：

| Owner | 事实 | 维度 | 现状 |
| --- | --- | --- | --- |
| `runtime/config` `RuntimeReactConfig` | `MaxIterations`、`ToolCallLimit`、`OnBudgetExhausted` | iteration、tool-call | 只有上限，无已用/剩余；越界即 fail-fast 终态 |
| `runtime/config` readiness admission | `budget_admission.v1` 快照与 `allow`/`degrade`/`deny` 决策 | cost、latency | 准入时点一次性估算，不是运行剩余 |
| `orchestration/scheduler` | `SpawnRequest.ParentRemainingBudget` | time | 只用于父→子准入裁决与 timeout 裁剪 |
| `core/runner` `reactPlanNotebook` | `ChangeTotal`、`RecoverCount`、plan status | plan churn | 只服务 plan lifecycle |

因此派生投影的**唯一正确形态**是：由这些 owner 各自提供的**事实快照**作为输入，计算出一个**只读、有界、nullable** 的派生视图；投影层不拥有任何事实，也不接受回写。

## 架构决策

### 决策 1：契约包放在 `context/budgetprojection`，且只依赖标准库

投影的消费方在未来是 `context/assembler` 的 tail recap（本提案不接线）。放在 `context/` 下与「上下文派生投影」定位一致，也满足 `docs/runtime-module-boundaries.md` 的约束（`context/*` 不引入 provider SDK）。

为避免形成 `context → runtime/orchestration` 的反向依赖，**契约包不 import 任何 Baymax 包**：`BudgetFacts` 是与四个 owner 字段对齐的纯数据快照结构，运行时接线（若有）由调用方负责填充，本提案只提供 fixture 填充路径。这样契约包可被 `tool/diagnosticsreplay` 与 `tool/contributioncheck` 安全复用，也不会把 runtime 依赖拖进 replay。

### 决策 2：`facts → projection` 是纯函数，缺失维度一律 `available=false`

投影计算 `DeriveBudgetProjection(facts BudgetFacts) BudgetProjection` 必须是纯函数：不读时钟、不读全局状态、不产生副作用。任一维度缺少事实（limit 缺失、used 缺失、时间预算缺失、cost 阈值缺失）时，该维度输出 `Available=false` 且**不填剩余值**；`Complete` 仅在四个维度全部 available 时为 true。

这条口径直接对应 roadmap 的 `additive + nullable + default`：历史/缺失数据不得被解释为"剩余 0"或"预算充足"，因为两者都会诱导模型做出错误决策。

### 决策 3：pressure level 是派生的枚举，不是事实

`Pressure` 由最大维度使用率派生：`none`（无可用维度）/ `normal`(<0.5) / `elevated`(0.5–0.8) / `critical`(0.8–1.0) / `saturated`(≥1.0)。它是**投影层的派生表达**，不回写任何 owner，也不作为终态判定依据（终态仍由 ReAct termination 常量与 admission decision 拥有）。

### 决策 4：benchmark 必须是计算型、离线、确定性的

roadmap 要求「先 benchmark 后策略」。真跑模型不可重复、不可在 CI 中断言，因此本提案的 benchmark 是**轨迹重算型**：fixture 给出每条 trace 的逐步预算事实与步骤结果（`progress` / `repeat` / `terminal_success` / `terminal_exhausted`），benchmark 计算四项指标并产出 canonical digest：

- `completion_rate` = `terminal_success` trace 数 / trace 总数
- `repeated_loop_rate` = `repeat` 步数 / 总步数
- `budget_utilization` = 各维度 `used/limit` 的均值（按维度稳定排序输出）
- `saturation_rate` = `terminal_exhausted` trace 数 / trace 总数

snapshot/recovery 后的确定性重算用「同一 fixture 二次解析 + 二次计算 + digest 对比」表达，不引入持久化。这满足 roadmap 要求的「recovery 后预算投影的确定性重算」，同时保持 `computational-first`。

### 决策 5：不改运行时接线，只交付证据

与归档 142（请求侧投影）一致：本提案证明「没有投影」这一事实、钉住可复现的缺失分类、给出可比对的 benchmark 口径，但**不修改任何运行时行为**。是否把投影接入 tail recap，必须由后续增量 change 依据本提案的 benchmark 证据决定。

## 数据模型

```
BudgetFacts (source-owned 快照，只读输入)
├── Iteration {Limit, Used}
├── ToolCall  {Limit, Used}
├── Time      {BudgetMS, ElapsedMS}          // 可缺失
├── Cost      {EstimateTotal, HardThreshold, DegradeThreshold}  // 可缺失
├── Admission {Decision, DegradeAction}      // allow|degrade|deny
├── Parent    {RemainingMS}                  // 可缺失
└── Plan      {ChangeTotal, RecoverCount, Status}

BudgetProjection (派生输出，有界只读)
├── Version   "budget_projection.v1"
├── Complete  bool
├── Remaining {Iteration, ToolCall, TimeMS, CostHeadroom}   // 各带 Available bool
├── Pressure  none|normal|elevated|critical|saturated
├── Notes     []string   // ≤8 条，每条 ≤120 字符，稳定排序
└── Digest    sha256(canonical)
```

有界性上界（超界 fail-fast，不截断）：

- `MaxProjectionBytes = 4096`
- `MaxNotes = 8`、`MaxNoteChars = 120`
- `MaxFixturesBytes = 2 MiB`
- benchmark：`MaxTraces = 32`、`MaxStepsPerTrace = 64`

## 分类词表（18 个稳定码）

投影与契约：

1. `budget_projection_schema_drift`
2. `budget_projection_unknown_version`
3. `budget_facts_missing_iteration_limit`
4. `budget_facts_missing_tool_call_limit`
5. `budget_facts_missing_time_budget`
6. `budget_facts_missing_cost_threshold`
7. `budget_projection_negative_remaining`
8. `budget_projection_ratio_out_of_range`
9. `budget_projection_pressure_level_mismatch`
10. `budget_projection_note_unbounded`
11. `budget_projection_digest_mismatch`
12. `budget_projection_overflow_drift`
13. `budget_projection_writeback_shape_detected`

benchmark 与 replay：

14. `budget_benchmark_schema_drift`
15. `budget_benchmark_metric_mismatch`
16. `budget_benchmark_recovery_recompute_drift`
17. `budget_benchmark_overflow_drift`
18. `budget_projection_replay_not_idempotent`

## 不变量与 gate 断言

| 不变量 | 断言方式 |
| --- | --- |
| `budget_derived_projection_readonly` | `context/budgetprojection` 不得 import `runtime/*`、`orchestration/*`、`model/*`、`observability/*`；不得出现 setter/回写形状 |
| `budget_no_new_ledger` | 不得新增 `runtime.admission.*` 或其它预算配置键；`runtime/config`、`orchestration/scheduler`、`core/runner`、`context/assembler` 现有代码零修改 |
| `budget_projection_nullable_default` | 缺失维度 `Available=false` 且无剩余值；历史 fixture 缺字段解析成功；未知字段安全忽略 |
| `budget_projection_bounded` | fixture ≤2 MiB、projection ≤4 KiB、notes ≤8/≤120；benchmark traces ≤32、steps ≤64 |
| `budget_projection_deterministic` | 同一 facts 两次投影 digest 一致；benchmark recovery 重算 digest 一致 |
| `budget_projection_taxonomy_stable` | 18 个码同时存在于契约常量、`context/README.md` 与 gate 脚本 |

## 验证命令

```bash
go test ./context/budgetprojection ./tool/diagnosticsreplay ./tool/contributioncheck -count=1
go test ./context/budgetprojection ./tool/diagnosticsreplay -race -count=1
go test ./tool/diagnosticsreplay -run BudgetProjection -count=2
bash scripts/check-budget-aware-context-projection-contract.sh
pwsh -File scripts/check-budget-aware-context-projection-contract.ps1
bash scripts/check-openspec-roadmap-status-consistency.sh
openspec validate establish-budget-aware-derived-context-projection-contract --strict
openspec validate --all
```
