## 1. Config and Contract

- [x] 1.1 在 runtime 配置新增 CA2 external retriever observability 配置（默认窗口 `15m`）
- [x] 1.2 新增静态阈值字段：`p95_latency_ms`、`error_rate`、`hit_rate`
- [x] 1.3 增加配置校验（非法阈值与窗口 fail-fast，热更新保持回滚语义）

## 2. Diagnostics and Aggregation

- [x] 2.1 在 diagnostics store 增加 provider 维度趋势聚合结构
- [x] 2.2 暴露 diagnostics API 查询接口（库接口）用于 CA2 external provider 趋势
- [x] 2.3 输出最小字段：`provider/window_start/window_end/p95_latency_ms/error_rate/hit_rate`
- [x] 2.4 输出阈值命中信号字段（仅信号，不触发自动策略动作）

## 3. Error Layering and Semantics

- [x] 3.1 统一 Stage2 error layer 统计口径并允许新增枚举扩展
- [x] 3.2 保持 Run/Stream 在 CA2 external 趋势与阈值信号语义一致
- [x] 3.3 保持 `fail_fast/best_effort` 行为不回归（仅观测增强）

## 4. Tests and Baseline

- [x] 4.1 增加契约测试：provider 趋势字段与窗口语义
- [x] 4.2 增加契约测试：阈值命中信号（无自动动作）
- [x] 4.3 增加契约测试：Run/Stream 等价 + 错误层扩展兼容
- [x] 4.4 执行并通过 `go test ./...`
- [x] 4.5 执行并通过 `go test -race ./...`
- [x] 4.6 新增并通过 `BenchmarkCA2ExternalRetrieverTrendAggregation`（baseline）

## 5. Docs Sync

- [x] 5.1 更新 `README.md`（CA2 external observability 与阈值配置）
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`（API、字段、阈值语义）
- [x] 5.3 更新 `docs/development-roadmap.md`（标记 E2 落点）
- [x] 5.4 更新 `docs/v1-acceptance.md`（新增验收条目）
- [x] 5.5 更新 `docs/mainline-contract-test-index.md`（登记新增契约与基准）
