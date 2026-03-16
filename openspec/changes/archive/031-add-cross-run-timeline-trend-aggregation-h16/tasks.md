## 1. Config and Diagnostics Contract

- [x] 1.1 在 runtime 配置中新增跨 run 趋势窗口配置（默认启用、`last_n_runs=100`、`time_window=15m`）
- [x] 1.2 增加配置校验（非法窗口值 fail-fast，热更新保持回滚语义）
- [x] 1.3 在 diagnostics API 中新增趋势查询结构，保持现有 run 级字段兼容

## 2. Trend Aggregation Implementation

- [x] 2.1 在 diagnostics store 增加内存窗口索引与有界样本维护（单进程）
- [x] 2.2 实现 `last_n_runs` 模式聚合
- [x] 2.3 实现 `time_window` 模式聚合
- [x] 2.4 输出 `phase+status` 双维度指标与最小字段集（含 `latency_p95_ms`）
- [x] 2.5 复用 idempotency 语义，确保 replay/duplicate 不重复计数

## 3. Semantic Consistency

- [x] 3.1 对齐 Run/Stream 在趋势聚合口径上的语义一致性
- [x] 3.2 明确无样本窗口的返回语义（空集合，不伪造统计）

## 4. Tests and Benchmark

- [x] 4.1 增加契约测试：双窗口模式 + 双维度字段完整性
- [x] 4.2 增加契约测试：Run/Stream 等价
- [x] 4.3 增加契约测试：replay/duplicate 幂等
- [x] 4.4 执行并通过 `go test -race ./...`
- [x] 4.5 增加并通过 benchmark smoke（覆盖趋势查询与 p95 字段）

## 5. Documentation Sync

- [x] 5.1 更新 `README.md`（趋势聚合能力与最小字段说明）
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`（配置、查询、语义与边界）
- [x] 5.3 更新 `docs/development-roadmap.md`（标注 H16 落点）
- [x] 5.4 更新 `docs/v1-acceptance.md`（新增验收条目）
- [x] 5.5 更新 `docs/mainline-contract-test-index.md`（登记新增契约测试）
