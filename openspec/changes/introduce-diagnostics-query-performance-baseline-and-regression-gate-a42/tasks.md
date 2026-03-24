## 1. Diagnostics Query Benchmark Matrix

- [ ] 1.1 在 `integration/benchmark_test.go` 新增 `QueryRuns` 基准（固定数据集、分页查询、排序路径）。
- [ ] 1.2 在 `integration/benchmark_test.go` 新增 `QueryMailbox` 基准（多维过滤 + 分页 + cursor 路径）。
- [ ] 1.3 在 `integration/benchmark_test.go` 新增 `MailboxAggregates` 基准（聚合计数与 reason totals 路径）。
- [ ] 1.4 抽取可复用的 diagnostics benchmark 数据集构造器，固定记录规模与分布，避免随机漂移。

## 2. Regression Gate Scripts

- [ ] 2.1 新增 `scripts/check-diagnostics-query-performance-regression.sh`，实现 baseline 读取、参数校验、阈值比较、fail-fast。
- [ ] 2.2 新增 `scripts/check-diagnostics-query-performance-regression.ps1`，保持与 shell 脚本语义等价。
- [ ] 2.3 新增 `scripts/diagnostics-query-benchmark-baseline.env`，提供 QueryRuns/QueryMailbox/MailboxAggregates 的三指标基线值。
- [ ] 2.4 为脚本补齐“baseline 缺失/非法、参数非法、输出不可解析”的负例测试或最小可验证用例。

## 3. Quality Gate Integration

- [ ] 3.1 更新 `scripts/check-quality-gate.sh`，将 diagnostics-query perf gate 作为阻断步骤接入。
- [ ] 3.2 更新 `scripts/check-quality-gate.ps1`，保持步骤顺序与失败语义 parity。
- [ ] 3.3 更新 gate 日志标签，确保 diagnostics-query 回归失败可独立定位。

## 4. Contract and Documentation Alignment

- [ ] 4.1 更新 `docs/performance-policy.md`，补充 diagnostics-query benchmark 与阈值默认值。
- [ ] 4.2 更新 `docs/mainline-contract-test-index.md`，补充 A42 的 benchmark 与 gate 映射行。
- [ ] 4.3 更新 `docs/development-roadmap.md` 与 `README.md` 的状态快照和质量门禁描述，避免口径漂移。

## 5. Validation and Acceptance

- [ ] 5.1 执行 `go test ./integration -run ^$ -bench '^BenchmarkDiagnostics(QueryRuns|QueryMailbox|MailboxAggregates)$' -benchmem -benchtime=200ms -count=5` 并记录结果。
- [ ] 5.2 执行 `pwsh -File scripts/check-diagnostics-query-performance-regression.ps1` 验证阈值与 fail-fast 语义。
- [ ] 5.3 执行 `pwsh -File scripts/check-quality-gate.ps1` 与 `pwsh -File scripts/check-docs-consistency.ps1`，确认 gate 与文档一致性通过。
