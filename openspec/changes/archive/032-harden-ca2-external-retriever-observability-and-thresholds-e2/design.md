## Context

E1 已完成 external retriever profile 与基础错误分层落地，但 CA2 Stage2 的治理能力仍停留在单次 run 诊断。当前缺口是：

1. 缺少 provider 维度趋势聚合，不利于判断慢化与错误漂移。
2. 缺少静态阈值配置与触发信号，无法形成“可观测 -> 决策”的闭环。
3. 缺少统一窗口口径，跨环境比较成本高。

本提案聚焦 E2 可观测增强，不引入自动降级策略，不扩展调度行为。

## Goals / Non-Goals

Goals:
- 提供 CA2 external retriever provider 维度趋势聚合（diagnostics API）。
- 增加静态阈值配置（`p95_latency_ms`/`error_rate`/`hit_rate`）与触发信号输出。
- 统一默认窗口为 `15m`（可配置）。
- 统一错误分层基线并允许新增枚举扩展。
- 维持 Run/Stream 与 `fail_fast/best_effort` 语义不变。

Non-Goals:
- 不新增自动降级或自动切 provider 动作。
- 不新增 CLI 接口。
- 不引入 provider 专用 SDK adapter。
- 不改 assembler/runner 主状态机。

## Decisions

### Decision 1: 阈值仅用于信号，不驱动自动动作
- 方案：阈值命中写入 diagnostics/event 信号；调用链行为保持不变。
- 原因：控制变更风险，先保证观测口径稳定。

### Decision 2: 趋势窗口默认 15m
- 方案：提供 time-window 查询与默认 `15m` 配置。
- 原因：与现有观测场景一致，足够用于回归判断。

### Decision 3: provider 维度聚合最小指标固定
- 方案：固定输出 `p95_latency_ms`、`error_rate`、`hit_rate`。
- 原因：覆盖性能、稳定性、有效性三个核心维度。

### Decision 4: 错误分层允许枚举扩展
- 方案：保留 `transport|protocol|semantic` 基线，同时允许新增枚举值。
- 原因：兼容后续 provider 差异，避免重复破坏性重构。

## Data Model (Conceptual)

- `ca2.external_observability.window`：默认 `15m`
- `ca2.external_observability.thresholds`
  - `p95_latency_ms`
  - `error_rate`
  - `hit_rate`
- `CA2ExternalProviderTrendRecord`
  - `provider`
  - `window_start`
  - `window_end`
  - `p95_latency_ms`
  - `error_rate`
  - `hit_rate`
  - `threshold_hits`（命中字段集合）
  - `error_layer_distribution`（允许新增层枚举）

## Validation Plan

- Contract tests:
  - provider 趋势查询字段完整性与窗口语义。
  - 阈值命中信号正确性（不触发自动动作）。
  - Run/Stream 等价语义。
  - 错误层扩展枚举兼容性。
- Concurrency:
  - `go test -race ./...` 覆盖趋势统计与查询并发。
- Benchmark:
  - `BenchmarkCA2ExternalRetrieverTrendAggregation`，输出 `p95-ns/op`。

## Risks / Trade-offs

- 风险：趋势统计开销增加。
  - 缓解：窗口有界 + 增量聚合 + benchmark 监控。
- 风险：阈值过敏导致信号噪音。
  - 缓解：静态阈值可配置，首期不自动动作。
- 风险：错误层扩展导致消费者解析差异。
  - 缓解：维持基线层枚举与向后兼容字段语义。
