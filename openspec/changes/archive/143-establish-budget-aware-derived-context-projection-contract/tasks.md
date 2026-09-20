## Example Impact Assessment

无需示例变更（附理由）

本变更只新增离线契约包、fixture、benchmark、replay 与 gate，不修改 `examples/agent-modes` 的配置键、runtime path、expected markers 或任何可观察语义，也不把派生投影接入运行时上下文（不改 `context/assembler`、不改 ReAct 循环、不改 admission 阈值）。若后续增量 change 决定把投影接入 tail recap 并改变示例可观察输出，必须重新评估并先完成 `MATRIX.md` 与对应模式 README 的文档基线。

## 1. Baseline and Projection Contract

- [x] 1.1 复核 `runtime/config/runtime_react.go`（`MaxIterations`/`ToolCallLimit`/`OnBudgetExhausted`）、`runtime/config/readiness.go`（`budget_admission.v1` 与 cost/latency 决策）、`orchestration/scheduler/types.go`（`ParentRemainingBudget`）、`core/runner/react_plan_notebook.go` 与 `context/assembler/assembler.go`（`buildTaskAwareTailRecap`）；确认 `context/*` 无 remaining budget 投影，并把可复现事实写入 proposal 的 Why。
- [x] 1.2 在 `context/budgetprojection` 定义 `budget_projection.v1`：有界 `BudgetFacts` 快照与 `BudgetProjection` 派生结构、canonical 序列化与 SHA-256 digest、18 个稳定分类码；覆盖 malformed、unknown-version、超界与缺字段校验用例。
- [x] 1.3 定义 nullable + default 口径并落测试：任一维度事实缺失时 `Available=false` 且不填剩余值；历史 fixture 缺字段解析成功；未知字段被安全忽略。

## 2. Derived Projection Semantics

- [x] 2.1 实现 `DeriveBudgetProjection(facts BudgetFacts) BudgetProjection` 纯函数：计算 remaining iteration/tool/time/cost，并断言不读时钟、不读全局状态、无副作用。
- [x] 2.2 实现 pressure level 派生（`none`/`normal`/`elevated`/`critical`/`saturated`）与边界用例：0、0.5、0.8、1.0 与超界点的归属判定。
- [x] 2.3 实现 bounded notes（≤8 条、每条 ≤120 字符、稳定排序）与负向用例：notes 超限、单条超长、排序不稳定都返回对应分类码。
- [x] 2.4 添加负向用例：`used > limit` 返回 `budget_projection_negative_remaining`；比值越界返回 `budget_projection_ratio_out_of_range`；declared pressure 与计算值不符返回 `budget_projection_pressure_level_mismatch`。

## 3. Offline Budget Utilization Benchmark

- [x] 3.1 定义 `budget_benchmark` 轨迹模型：trace → steps（每步预算事实 + `progress`/`repeat`/`terminal_success`/`terminal_exhausted`），并给出 canonical 序列化与 digest。
- [x] 3.2 实现四项确定性指标：`completion_rate`、`repeated_loop_rate`、`budget_utilization`（按维度稳定排序）、`saturation_rate`，并断言同一输入两次计算 digest 一致。
- [x] 3.3 实现 snapshot/recovery 后的确定性重算校验：同一 fixture 二次解析 + 二次计算，digest 不一致时返回 `budget_benchmark_recovery_recompute_drift`。
- [x] 3.4 添加 benchmark 边界用例：traces > 32、单 trace steps > 64 与非法 outcome 分别返回 `budget_benchmark_overflow_drift` 或 `budget_benchmark_schema_drift`。

## 4. Replay and Contract Gates

- [x] 4.1 在 `tool/diagnosticsreplay` 实现 `budget_projection.v1` 解析、有界归一化、离线只读执行与幂等校验。
- [x] 4.2 添加 drift/gap 分类测试：schema、unknown version、四个维度缺失、negative remaining、ratio 越界、pressure mismatch、notes 越界、digest 不一致、overflow、benchmark metric mismatch、recovery recompute。
- [x] 4.3 新增版本化 fixture `tool/diagnosticsreplay/testdata/budget_projection.v1.json`，覆盖全维度/缺维度/历史缺字段/未知字段/越界/benchmark 正常与恢复重算。
- [x] 4.4 新增 `scripts/check-budget-aware-context-projection-contract.sh` 与 `.ps1` 对等 gate：只读性、无第二本账本、无新配置键、有界性、taxonomy 稳定、replay 幂等、benchmark 确定性。
- [x] 4.5 把 gate 接入 `scripts/check-quality-gate.sh` 与 `scripts/check-quality-gate.ps1`，并验证缺失 fixture 或 gate 证据会阻断质量门禁。

## 5. Ownership, Privacy and Documentation

- [x] 5.1 添加 `tool/contributioncheck/budget_projection_boundary_test.go`：契约包零 Baymax 依赖、无 raw payload 字段、有界上界、taxonomy 文档一致性、无新配置键。
- [x] 5.2 验证本变更不新增诊断字段、不落盘 raw reasoning/transcript/credentials，且 `RuntimeRecorder` 单写入口未被绕过。
- [x] 5.3 更新 `docs/mainline-contract-test-index.md`（新 gate 映射）、`context/README.md`（派生投影契约与 owner 边界）、`docs/development-roadmap.md`（当前状态与候选收敛）、`docs/runtime-module-boundaries.md`（`context/budgetprojection` 职责）与 `README.md`（里程碑快照）。
- [x] 5.4 运行 `bash scripts/check-openspec-roadmap-status-consistency.sh` 与状态一致性 Go 测试，确认无 `roadmap-status-drift` 与文档不一致。

## 6. Integrated Verification and Handoff

- [x] 6.1 运行聚焦套件（`context/budgetprojection`、`tool/diagnosticsreplay`、`tool/contributioncheck`）并记录正/负/边界/幂等证据。
- [x] 6.2 运行 `openspec validate --all`、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml` 与两个 budget contract gate；记录精确结果与环境受限命令及原因。
- [x] 6.3 对照非目标做最终范围复核：确认未新增预算账本、runtime 配置键、admission 阈值变更、scheduler 语义变更、ReAct 策略变更、tail recap 接线、诊断字段或 raw payload 落盘。

## Verification Evidence

- 聚焦测试 `go test ./context/budgetprojection ./tool/diagnosticsreplay ./tool/contributioncheck -count=1` 及对应 `-race` 套件均通过；覆盖正向、负向、边界与 replay 幂等。
- `openspec validate --all` 结果为 123 passed、0 failed；`go test ./... -count=1`、`go test -race ./... -count=1 -timeout 20m` 与 `golangci-lint run --config .golangci.yml` 均通过（lint：`0 issues.`）。
- `pwsh -File scripts/check-budget-aware-context-projection-contract.ps1`、示例影响声明与文档一致性 gate 均通过；完整 `check-quality-gate.ps1` 已通过前 68 步（包括全量普通/竞态测试与 lint），在第 69 步因全局 900 秒预算耗尽而终止。被中断的 `pwsh -File scripts/check-a64-performance-regression.ps1` 已单独完整通过。
- 本 Windows sandbox 中的 `C:\\Windows\\System32\\bash.exe` 会在 WSL 初始化阶段返回 `Bash/Service/CreateInstance/E_ACCESSDENIED`，因此无法执行两个 Bash 对等 gate；PowerShell 对等实现已通过，且 `tool/contributioncheck` 的静态回归用例覆盖 Shell quality-gate 的独立接线形状。
- 最终范围复核与 `git diff --check` 均完成；投影包维持零 Baymax 依赖，没有 runtime 配置、scheduler、ReAct、tail recap、诊断 schema 或 raw payload 持久化改动。
