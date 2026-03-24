## Why

随着 A18/A29/A30-A41 持续扩展 diagnostics 与 mailbox 观测字段，`QueryRuns`、`QueryMailbox`、`MailboxAggregates` 的查询负载与排序开销持续上升。当前仓库缺少针对 diagnostics 查询路径的独立性能回归门禁，无法在功能迭代中稳定识别查询尾延迟与分配放大回归。

## What Changes

- 新增 diagnostics 查询性能基线能力，覆盖 `QueryRuns`、`QueryMailbox`、`MailboxAggregates` 的 benchmark 矩阵与固定数据集生成规则。
- 新增 diagnostics-query 回归脚本（Shell + PowerShell）与 baseline 文件，采用相对阈值判定 `ns/op`、`p95-ns/op`、`allocs/op`。
- 固化默认执行参数（`benchtime=200ms`、`count=5`）与默认数据集规模，并支持环境变量覆盖。
- 固化失败语义：baseline 缺失/非法、参数非法、benchmark 输出不可解析必须 fail-fast 并阻断质量门禁。
- 将 diagnostics-query 性能 gate 接入 `check-quality-gate.*`，保证本地与 CI 语义一致。
- 同步更新 performance policy、主干契约索引与 roadmap 文档映射。

## Capabilities

### New Capabilities
- `diagnostics-query-performance-baseline`: 定义 diagnostics 查询 benchmark 矩阵、baseline 比较口径、阈值治理与 fail-fast 语义。

### Modified Capabilities
- `go-quality-gate`: 将 diagnostics 查询性能回归 gate 纳入标准阻断路径并保持 shell/PowerShell parity。

## Impact

- 代码：
  - `integration/benchmark_test.go`（新增 diagnostics query benchmarks 与固定数据集基准）
  - `scripts/check-diagnostics-query-performance-regression.sh`
  - `scripts/check-diagnostics-query-performance-regression.ps1`
  - `scripts/diagnostics-query-benchmark-baseline.env`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
- 测试与验证：
  - `go test ./integration -run ^$ -bench '^BenchmarkDiagnostics(QueryRuns|QueryMailbox|MailboxAggregates)' -benchmem ...`
  - diagnostics-query regression 脚本本地/CI 一致执行
- 文档：
  - `docs/performance-policy.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
