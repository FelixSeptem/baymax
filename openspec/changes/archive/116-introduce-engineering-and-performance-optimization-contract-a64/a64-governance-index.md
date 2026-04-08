# A64 Baseline and Governance Index

更新时间：2026-04-07  
覆盖任务：`tasks.md` 1.1 / 1.3 / 1.4

## 1. A64-S1~S10 模块映射与负责人（Task 1.1）

| S | 优化域 | 主要模块 | 模块负责人（owner label） | 当前基准锚点 |
| --- | --- | --- | --- | --- |
| S1 | Context Assembler + Stage2 hotpath | `context/*`、`core/runner` stage2 path | `context-assembler` | `BenchmarkContextProductionHardeningPressureEvaluation`、`BenchmarkContextPressureSemanticCompactionLatency*` |
| S2 | Diagnostics + RuntimeRecorder hotpath | `runtime/diagnostics`、`observability/event` | `runtime-diagnostics` + `observability` | `BenchmarkDiagnosticsQueryRuns*`、`BenchmarkDiagnosticsMailboxAggregates` |
| S3 | Scheduler/Mailbox/Recovery file persistence | `orchestration/scheduler`、`orchestration/mailbox`、`orchestration/composer` | `orchestration-scheduler` + `orchestration-composer` | `BenchmarkMultiAgentMainlineSyncInvocation`、`BenchmarkMultiAgentMainlineDelayedDispatch`（当前代理锚点） |
| S4 | MCP transport + diag store | `mcp/http`、`mcp/stdio`、`mcp/diag` | `mcp` | `BenchmarkMCPReconnectOverhead`、`BenchmarkMCPProfileHighReliabilityUnderFailure` |
| S5 | Skill loader discover/compile/score | `skill/loader` | `skill-loader` | 暂缺；由 Task 6.4 落地 `BenchmarkSkillLoader*` |
| S6 | Memory filesystem engine | `memory/filesystem` | `memory-filesystem` | 暂缺；由 Task 7.4 落地 `BenchmarkMemoryFilesystem*` |
| S7 | Runner + local dispatch | `core/runner`、`tool/local` | `core-runner` + `tool-local` | `BenchmarkIterationLatency`、`BenchmarkToolFanOut*` |
| S8 | Provider adapter stream/decode | `model/openai`、`model/anthropic`、`model/gemini` | `model-provider` | 暂缺；由 Task 9.4 落地 `BenchmarkProvider*` |
| S9 | Runtime config read path + policy resolve | `runtime/config` | `runtime-config` | `BenchmarkRuntimeConfigReadPath*`、`BenchmarkMCPPolicyResolve*` |
| S10 | Observability pipeline | `observability/*` | `observability` | `BenchmarkRuntimeExporterBatch*`、`BenchmarkEventDispatcherFanout*`、`BenchmarkJSONLoggerEmit*` |

说明：
- owner label 与仓库治理口径对齐（参照 `openspec/governance/go-file-line-budget-exceptions.csv` 与 `docs/runtime-module-boundaries.md`）。
- `S3` 当前使用代理锚点，后续在对应 `*.4` 子任务落地模块级 benchmark 后替换为目标锚点。

## 2. A64 回归阈值与基线更新流程（Task 1.3）

### 2.1 阈值口径

统一要求（A64 新增与更新基准）：
- `ns/op`：候选值相对基线上升不得超过阈值
- `allocs/op`：候选值相对基线上升不得超过阈值
- `B/op`：候选值相对基线上升不得超过阈值

默认阈值（A64 通用，除非子门禁另有更严格口径）：
- `max_ns_degradation_pct = 12`
- `max_allocs_degradation_pct = 12`
- `max_b_degradation_pct = 12`

与既有主线门禁兼容的更严格阈值保持不变：
- `check-context-production-hardening-benchmark-regression.*`：`ns/op 5%`、`p95-ns/op 8%`
- `check-multi-agent-performance-regression.*`：`ns/op 8%`、`p95-ns/op 12%`、`allocs/op 10%`
- `check-diagnostics-query-performance-regression.*`：`ns/op 12%`、`p95-ns/op 15%`、`allocs/op 12%`

治理原则：
- A64 新增 `B/op` 阈值时，不得放宽上述既有阈值；
- 同一 benchmark 若同时被 A64 门禁和既有门禁覆盖，按“更严格者优先”。

### 2.2 基线更新触发条件

满足任一条件可发起 re-baseline：
- benchmark 集合发生变化（新增、重命名、删除）；
- 同一硬件/Go 版本下，连续 2 次独立采样显示稳定偏移（中位数变化超过 `8%`）；
- 运行环境基线变化（Go 版本升级、CPU 架构变化、CI 规格变化）。

### 2.3 基线更新审批口径

基线更新必须在同一 PR 同步提交以下证据：
- 原始 benchmark 输出（至少 `count=3`）；
- 中位数汇总（`ns/op`、`allocs/op`、`B/op`）；
- 触发原因与风险说明；
- 受影响 S 项映射（S1~S10）与 impacted-contract suites。

审批要求：
- 至少 2 个 owner 同意：
  - 对应 S 项模块 owner；
  - 质量门禁 owner（`go-quality-gate` 维护方）。

阻断语义：
- baseline 缺失/非数值、阈值参数非法、输出不可解析必须 fail-fast；
- 门禁脚本禁止自动回写 baseline 文件。

## 3. 默认开关与回滚策略（Task 1.4）

| S | 默认开关状态 | 回滚开关（语义） | 失败回滚策略 |
| --- | --- | --- | --- |
| S1 | 默认关闭优化路径（走基线路径） | `runtime.optimization.a64.s1.*.enabled=false` | 热更新非法值 fail-fast；`runtime/config.Manager` 原子回滚到上一快照 |
| S2 | 默认关闭优化路径 | `runtime.optimization.a64.s2.*.enabled=false` | 同上 |
| S3 | 默认关闭优化路径 | `runtime.optimization.a64.s3.*.enabled=false` | 同上；持久化批写参数非法值不得生效 |
| S4 | 默认关闭优化路径 | `runtime.optimization.a64.s4.*.enabled=false` | 同上 |
| S5 | 默认关闭优化路径 | `runtime.optimization.a64.s5.*.enabled=false` | 同上 |
| S6 | 默认关闭优化路径 | `runtime.optimization.a64.s6.*.enabled=false` | 同上；durability 语义默认不变 |
| S7 | 默认关闭优化路径 | `runtime.optimization.a64.s7.*.enabled=false` | 同上 |
| S8 | 默认关闭优化路径 | `runtime.optimization.a64.s8.*.enabled=false` | 同上 |
| S9 | 默认关闭优化路径 | `runtime.optimization.a64.s9.*.enabled=false` | 同上；`env > file > default` 优先级不变 |
| S10 | 默认关闭优化路径 | `runtime.optimization.a64.s10.*.enabled=false` | 同上；慢 handler 隔离失败不得改变业务终态语义 |

实施约束：
- 在各 S 子任务落地时，必须为对应 `enabled` 路径补齐：
  - baseline/optimized 双路径语义等价测试；
  - 热更新非法值原子回滚测试；
  - Run/Stream + replay 等价回归。
- 若开关尚未落代码实现，视为“仅 baseline 路径可用”；不得提前默认开启。

## 4. A64 Impacted-Contract Suites 显式命令映射（Task 12.7）

说明：
- 下表为 A64 `S1~S10` 的最低必跑命令映射；`shell` 与 `PowerShell` 必须语义等价。
- 若同一改动命中多个 S 项，需合并执行所有命中的最低必跑集合。

| S | Scope | Shell | PowerShell |
| --- | --- | --- | --- |
| S1 | context assembler + stage2 provider + journal | `go test ./context/assembler ./context/provider ./context/journal -count=1`<br>`bash scripts/check-diagnostics-replay-contract.sh` | `go test ./context/assembler ./context/provider ./context/journal -count=1`<br>`pwsh -File scripts/check-diagnostics-replay-contract.ps1` |
| S2 | runtime recorder + diagnostics | `bash scripts/check-diagnostics-replay-contract.sh`<br>`bash scripts/check-diagnostics-query-performance-regression.sh` | `pwsh -File scripts/check-diagnostics-replay-contract.ps1`<br>`pwsh -File scripts/check-diagnostics-query-performance-regression.ps1` |
| S3 | scheduler/mailbox/composer recovery + query | `bash scripts/check-multi-agent-shared-contract.sh`<br>`go test ./orchestration/scheduler ./orchestration/composer -count=1` | `pwsh -File scripts/check-multi-agent-shared-contract.ps1`<br>`go test ./orchestration/scheduler ./orchestration/composer -count=1` |
| S4 | MCP transport invoke path | `go test ./mcp/http ./mcp/stdio ./mcp/retry -count=1`<br>`bash scripts/check-multi-agent-shared-contract.sh` | `go test ./mcp/http ./mcp/stdio ./mcp/retry -count=1`<br>`pwsh -File scripts/check-multi-agent-shared-contract.ps1` |
| S5 | skill loader discover/compile/scoring | `go test ./skill/loader ./runtime/config -count=1` | `go test ./skill/loader ./runtime/config -count=1` |
| S6 | memory filesystem engine | `bash scripts/check-memory-contract-conformance.sh`<br>`bash scripts/check-memory-scope-and-search-contract.sh` | `pwsh -File scripts/check-memory-contract-conformance.ps1`<br>`pwsh -File scripts/check-memory-scope-and-search-contract.ps1` |
| S7 | runner loop + local dispatch | `bash scripts/check-security-policy-contract.sh`<br>`bash scripts/check-security-event-contract.sh`<br>`bash scripts/check-security-delivery-contract.sh`<br>`bash scripts/check-security-sandbox-contract.sh` | `pwsh -File scripts/check-security-policy-contract.ps1`<br>`pwsh -File scripts/check-security-event-contract.ps1`<br>`pwsh -File scripts/check-security-delivery-contract.ps1`<br>`pwsh -File scripts/check-security-sandbox-contract.ps1` |
| S8 | provider adapters | `bash scripts/check-react-contract.sh` | `pwsh -File scripts/check-react-contract.ps1` |
| S9 | runtime config read-path + policy resolve | `bash scripts/check-policy-precedence-contract.sh`<br>`bash scripts/check-runtime-budget-admission-contract.sh`<br>`bash scripts/check-sandbox-rollout-governance-contract.sh` | `pwsh -File scripts/check-policy-precedence-contract.ps1`<br>`pwsh -File scripts/check-runtime-budget-admission-contract.ps1`<br>`pwsh -File scripts/check-sandbox-rollout-governance-contract.ps1` |
| S10 | observability dispatcher/logger/exporter pipeline | `bash scripts/check-observability-export-and-bundle-contract.sh`<br>`bash scripts/check-diagnostics-replay-contract.sh` | `pwsh -File scripts/check-observability-export-and-bundle-contract.ps1`<br>`pwsh -File scripts/check-diagnostics-replay-contract.ps1` |

横切兜底（所有 A64 子项合并前必跑）：
- Shell: `bash scripts/check-quality-gate.sh`
- PowerShell: `pwsh -File scripts/check-quality-gate.ps1`

## 5. Harnessability Scorecard 治理与阈值更新（Task 12.9~12.14）

### 5.1 机器可读输出与阻断指标

`check-a64-harnessability-scorecard.sh/.ps1` 必须输出 machine-readable JSON（默认 `./.artifacts/a64/harnessability-scorecard.json`），并包含以下阻断指标：

- `contract_coverage_pct`：通过 `check-a64-impacted-gate-selection.*` 的 `full` 模式验证 `S1~S10` 映射覆盖率；
- `drift`：inferential/drift fixture 计数与 `unclassified_drift_count`；
- `gate_coverage_pct`：`check-quality-gate.*` 中 A64 必选 gate 接线覆盖率（shell + PowerShell 双端）；
- `docs_consistency.issue_count`：A64 关键文档 marker 一致性计数；
- `roi`：按复杂度档位（`lightweight|standard|enhanced`）计算 `token/latency/quality` 三维开销与阈值比较；
- `hierarchy.computational_first_compliant`：客观 correctness 阻断必须来自 computational suites；
- `inferential_evidence`：结构化证据（`input_snapshot`、`prompt_version`、`scoring_summary`、`uncertainty_pct`）。

默认阈值来源：`scripts/a64-harnessability-scorecard-baseline.env`。

### 5.2 ROI 与深度分层口径

- 复杂度档位由 `BAYMAX_A64_HARNESS_COMPLEXITY_TIER` 指定，默认 `standard`；
- 每档位维护独立 baseline 与阈值：
  - `BAYMAX_A64_HARNESS_BASELINE_*_<TIER>`
  - `BAYMAX_A64_HARNESS_MAX_*_PCT_<TIER>`
  - `BAYMAX_A64_HARNESS_MIN_QUALITY_DELTA_PCT_<TIER>`
- 超阈值时必须给出降级建议（`enhanced -> standard -> lightweight`），并阻断合入；
- `inferential` 信号只可作为主观补充，不能替代 computational 阻断。

### 5.3 不确定性防误阻断口径

- `BAYMAX_A64_INFERENTIAL_BLOCKING_REQUESTED=true` 视为层级违规，直接阻断；
- 当 `uncertainty_pct > BAYMAX_A64_HARNESS_SCORECARD_MAX_INFERENTIAL_UNCERTAINTY_PCT` 时，inferential 结论不得作为阻断依据；
- 证据缺失（输入快照、提示版本、评分摘要任一缺失）必须阻断。

### 5.4 Harnessability 基线更新流程

触发条件（任一满足即可发起）：
- 新增/删除 scorecard 指标字段或阈值口径；
- complexity tier baseline 发生结构性变化；
- 连续 2 次独立采样显示稳定偏移（中位数偏移 > `8%`）。

提交要求（同一 PR）：
- 更新 `scripts/a64-harnessability-scorecard-baseline.env`；
- 附 scorecard 原始 JSON（建议保存在 `./.artifacts/a64/` 并在 PR 描述引用）；
- 说明触发原因、风险、回滚策略与受影响 S 项；
- 通过 shell/PowerShell parity 复核（见 5.5）。

审批要求：
- 对应 S 项 owner + `go-quality-gate` owner 至少各 1 人同意（最少 2 人）。

### 5.5 Shell / PowerShell parity 复核命令

- Shell:
  - `bash scripts/check-a64-harnessability-scorecard.sh`
- PowerShell:
  - `pwsh -File scripts/check-a64-harnessability-scorecard.ps1`

## 6. 门禁耗时基线与更新流程（Task 12.18）

### 6.1 基线文件与优先级

- 基线文件：`scripts/a64-gate-latency-baseline.env`
- 默认字段：
  - `BAYMAX_A64_GATE_LATENCY_MAX_STEP_SECONDS`
  - `BAYMAX_A64_GATE_LATENCY_MAX_TOTAL_SECONDS`
- 优先级固定：`env > baseline file > script default`
- `check-a64-gate-latency-budget.sh/.ps1` 必须在 baseline 解析异常时 fail-fast。

### 6.2 证据留存与审批口径

基线更新 PR 必须包含：
- 更新后的 `a64-gate-latency-baseline.env`；
- step-level latency 报告（建议 `BAYMAX_A64_GATE_LATENCY_REPORT_PATH=.artifacts/a64/gate-latency-budget-report.json`）；
- 触发原因、风险说明、回滚方案；
- Shell/PowerShell 同步执行证据（命令、时间戳、结果）。

审批要求：
- A64 门禁 owner + `go-quality-gate` owner 至少各 1 人同意。

### 6.3 parity 复核命令

- Shell:
  - `bash scripts/check-a64-gate-latency-budget.sh`
- PowerShell:
  - `pwsh -File scripts/check-a64-gate-latency-budget.ps1`
