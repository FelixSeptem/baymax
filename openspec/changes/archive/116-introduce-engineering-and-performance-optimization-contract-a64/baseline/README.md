# A64 Baseline Capture Record

更新时间：2026-04-07  
覆盖任务：`tasks.md` 1.2

## 1. 采样范围

当前已采样（存在 benchmark 实现）：
- Context 相关：`BenchmarkContextProductionHardeningPressureEvaluation`、`BenchmarkContextPressureSemanticCompactionLatency*`
- Diagnostics 相关：`BenchmarkDiagnosticsQueryRuns*`、`BenchmarkDiagnosticsQueryMailbox`、`BenchmarkDiagnosticsMailboxAggregates`
- Multi-agent 相关：`BenchmarkMultiAgentMainline*`
- Runner/MCP/Runtime-config 代理锚点：`BenchmarkIterationLatency`、`BenchmarkToolFanOut*`、`BenchmarkMCPReconnectOverhead`、`BenchmarkMCPProfileHighReliabilityUnderFailure`、`BenchmarkRuntimeBudgetAdmissionDecisionStability`

当前未采样（benchmark 尚未在主线落地）：
- S5 `BenchmarkSkillLoader*`
- S6 `BenchmarkMemoryFilesystem*`
- S8 `BenchmarkProvider*`
- S10 `BenchmarkRuntimeExporterBatch*` / `BenchmarkEventDispatcherFanout*` / `BenchmarkJSONLoggerEmit*`

以上空缺由后续 `6.4 / 7.4 / 9.4 / 11.4` 子任务补齐。

## 2. 可复现命令

采样前统一设置：

```powershell
$cache = Join-Path (Resolve-Path .) '.gocache'
if (-not (Test-Path $cache)) { New-Item -ItemType Directory -Path $cache | Out-Null }
$env:GOCACHE = $cache
```

执行命令（均带 `-benchmem`，默认 `-benchtime=200ms -count=3`）：

```powershell
go test ./integration -run '^$' -bench '^BenchmarkMultiAgentMainline(SyncInvocation|AsyncReporting|DelayedDispatch|RecoveryReplay)$' -benchmem -benchtime=200ms -count=3

go test ./integration -run '^$' -bench '^BenchmarkDiagnostics(QueryRuns|QueryRunsSandboxEnriched|QueryMailbox|MailboxAggregates)$' -benchmem -benchtime=200ms -count=3

go test ./integration -run '^$' -bench '^BenchmarkContext(ProductionHardeningPressureEvaluation|PressureSemanticCompactionLatency|PressureSemanticCompactionLatencyEmbeddingEnabled|PressureSemanticCompactionLatencyRerankerGovernanceEnabled)$' -benchmem -benchtime=200ms -count=3

go test ./integration -timeout 8m -run '^$' -bench '^Benchmark(IterationLatency|RuntimeBudgetAdmissionDecisionStability|ToolFanOut(HighConcurrency|SlowCall|CancelStorm)|MCPReconnectOverhead|MCPProfileHighReliabilityUnderFailure)$' -benchmem -benchtime=200ms -count=3
```

## 3. 产物路径

- 汇总 JSON：`baseline/a64-baseline-2026-04-06.json`
- 原始输出：
  - `baseline/raw/multi-agent.txt`
  - `baseline/raw/diagnostics.txt`
  - `baseline/raw/context.txt`
  - `baseline/raw/runtime-mcp-runner-stable.txt`
  - `baseline/raw/runtime-mcp-runner.txt`（含长尾异常样本，仅保留证据，不作为基线汇总输入）

## 4. 汇总口径

- 统一记录三指标：`ns/op`、`allocs/op`、`B/op`
- 统计方法：每个 benchmark 取 `count=3` 的中位数（median-of-3）
- 环境信息写入 `a64-baseline-2026-04-06.json`（commit、Go 版本、CPU、采样参数）

## 5. 长耗时异常说明

在一次合并采样（`runtime-mcp-runner.txt`）中出现了长尾异常：
- `BenchmarkToolFanOutDropLowPriority`
- `BenchmarkMCPProfileDefaultUnderFailure`

该次命令运行时间显著异常（分钟级），因此：
- 证据已保留在 `baseline/raw/runtime-mcp-runner.txt`
- 基线汇总改为拆分稳定采样，并在 `go test` 层显式加入 `-timeout 8m`
- 当前 `a64-baseline-2026-04-06.json` 不使用该异常批次数据
