## A63 Go File Line-Budget Inventory (2026-04-05)

### Method

```powershell
$files = git ls-files '*.go'
$result = foreach($f in $files){
  $lines = (Get-Content $f | Measure-Object -Line).Lines
  [PSCustomObject]@{Lines=$lines; Path=$f}
}
```

### Snapshot Summary

- total `.go` files: `333`
- files `> 800` lines: `33`
- files `> 1000` lines: `27`
- files `> 1200` lines (hard threshold candidates): `20`
- files `> 1500` lines: `14`

### Top 20 Largest Go Files

1. `core/runner/runner_test.go` (6805)
2. `runtime/config/config.go` (5945)
3. `core/runner/runner.go` (5771)
4. `tool/diagnosticsreplay/arbitration.go` (3907)
5. `runtime/config/config_test.go` (3695)
6. `observability/event/runtime_recorder_test.go` (3242)
7. `runtime/config/manager_test.go` (2899)
8. `runtime/config/readiness.go` (2749)
9. `runtime/diagnostics/store_test.go` (2509)
10. `runtime/diagnostics/store.go` (2316)
11. `context/assembler/assembler_test.go` (1871)
12. `orchestration/composer/composer.go` (1781)
13. `runtime/config/readiness_test.go` (1739)
14. `integration/sandbox_execution_isolation_contract_test.go` (1646)
15. `a2a/interop.go` (1406)
16. `orchestration/scheduler/state.go` (1404)
17. `integration/benchmark_test.go` (1404)
18. `context/assembler/assembler.go` (1287)
19. `context/assembler/context_pressure_recovery.go` (1219)
20. `skill/loader/loader_test.go` (1201)

### Hard-Threshold Exception Baseline (`>1200`)

The initial debt baseline is captured in:

- `openspec/governance/go-file-line-budget-policy.env`
- `openspec/governance/go-file-line-budget-exceptions.csv`

All exception entries are recorded with:

- `owner`
- `reason`
- `expiry`
- `baseline_lines`
- `allow_growth` (default `false`)

### Priority Guidance (A63 Split Sequencing)

1. `core/runner/*`
2. `runtime/config/*`
3. `tool/diagnosticsreplay/arbitration.go`
4. `runtime/diagnostics/*` and `observability/event/*`
5. `context/assembler/*`

This priority ordering is used for follow-up semantic-preserving split tasks.
