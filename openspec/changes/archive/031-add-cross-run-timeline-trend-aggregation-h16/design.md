## Context

H1.5 已完成单 run 的 Action Timeline phase 聚合，但当前诊断视角仍以 run 粒度为主。对生产排障更关键的是“窗口趋势”：
- 在最近 N 次/最近 T 时间内，phase 与 status 的分布是否变化；
- latency p95 是否持续抬升；
- Run 与 Stream 在窗口层是否保持等价。

本提案将该能力作为 H16 增量收敛，保持库接口优先、单进程内存窗口、fail-fast 语义不变。

## Goals / Non-Goals

Goals:
- 提供跨 run 窗口趋势聚合，支持 `last_n_runs` + `time_window` 双模式。
- 输出 `phase + status` 双维度聚合，包含最小指标集与 `latency_p95_ms`。
- 保持 Run/Stream 语义一致。
- 保持单写入与幂等，不因 replay/重放导致统计膨胀。
- 提供可配置窗口参数，默认开启。

Non-Goals:
- 不引入持久化历史聚合（本期仅内存窗口，单进程）。
- 不新增 CLI 面向该能力。
- 不改变现有 run 级聚合字段语义。

## Decisions

### Decision 1: 双窗口并存
- 方案：同时支持 `last_n_runs` 与 `time_window`，由查询或配置指定。
- 默认值：`last_n_runs=100`，`time_window=15m`。
- 原因：运行态既有按次数回看需求，也有按时间窗口监控需求。

### Decision 2: 双维度输出
- 方案：趋势聚合同时支持 `phase` 与 `status` 两维组合。
- 原因：仅 phase 无法定位失败/取消结构变化，仅 status 无法定位责任阶段。

### Decision 3: 指标最小集固定包含 p95
- 方案：固定输出 `count_total/failed_total/canceled_total/skipped_total/latency_avg_ms/latency_p95_ms/window_start/window_end`。
- 原因：p95 是回归判定基线，必须稳定可得。

### Decision 4: 默认启用 + 向后兼容
- 方案：默认启用趋势聚合；以增量字段/增量查询形式提供，不破坏现有调用。
- 原因：降低接入成本并保持兼容。

### Decision 5: 内存窗口实现
- 方案：首期仅做单进程内存窗口，不做持久化。
- 原因：控制 scope，先闭合语义与测试，再评估跨进程/持久化演进。

## Data Model (Conceptual)

- TrendWindowConfig
  - `enabled` (default true)
  - `last_n_runs` (default 100)
  - `time_window` (default 15m)
- TrendBucketKey
  - `phase`
  - `status`
- TrendMetrics
  - `count_total`
  - `failed_total`
  - `canceled_total`
  - `skipped_total`
  - `latency_avg_ms`
  - `latency_p95_ms`
  - `window_start`
  - `window_end`

## Execution Flow

1. Runtime recorder 继续单路径写入 run 级 timeline 聚合数据。
2. Diagnostics store 基于去重后的 run 记录维护内存窗口索引。
3. 趋势查询按 `last_n_runs` 或 `time_window` 截取样本后做 `phase+status` 聚合。
4. 输出稳定字段；无样本时返回空集合，不伪造统计。

## Risks / Trade-offs

- 风险：窗口聚合增加内存与计算开销。
  - 缓解：使用有界窗口与增量聚合；benchmark smoke 观测回归。
- 风险：Run/Stream 在窗口层统计口径偏移。
  - 缓解：新增契约测试保证等价语义。
- 风险：replay 造成重复计数。
  - 缓解：复用现有 single-writer + idempotency 路径。

## Validation Plan

- Contract tests:
  - 双窗口模式输出正确性（N 模式、T 模式）。
  - `phase+status` 双维度聚合正确性。
  - Run/Stream 等价语义。
  - replay/duplicate 不重复计数。
- Concurrency:
  - `go test -race ./...` 覆盖 trend 查询与写入并发场景。
- Benchmark smoke:
  - 覆盖趋势查询开销与 `latency_p95_ms` 字段稳定性。

## Rollout

1. 增量接入配置与 store 数据结构。
2. 增加趋势查询 API 与字段输出。
3. 契约测试 + race + benchmark smoke 收敛。
4. README/docs 一次性同步。
