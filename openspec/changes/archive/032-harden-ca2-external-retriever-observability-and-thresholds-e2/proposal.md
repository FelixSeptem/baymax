## Why

CA2 external retriever 已具备 `file/http/rag/db/elasticsearch` 的可运行主路径与基础诊断字段，但仍缺少“运营可治理”的统一阈值与趋势视角。当前外部检索路径在出现慢化或错误抬升时，缺乏稳定的 provider 维度聚合与阈值触发信号，导致是否进入 E3（provider 专用 adapter）缺少客观依据。

本提案收敛 E2：在不改变主流程策略（不自动降级、不新增调度动作）的前提下，补齐 CA2 external retriever 的可观测阈值治理。

## What Changes

- 为 CA2 external retriever 增加 provider 维度趋势聚合（diagnostics API）：
  - 指标：`p95_latency_ms`、`error_rate`、`hit_rate`
  - 默认窗口：`15m`（可配置）
  - 仅库接口，不新增 CLI。
- 增加静态阈值配置与触发信号输出（告警/诊断用途）：
  - 仅发出阈值命中信号，不自动执行降级/切换动作。
- 统一错误分层映射口径并允许新增枚举：
  - 基线：`transport|protocol|semantic`
  - 允许扩展枚举，保持向后兼容语义。
- 保持 Run/Stream 与 `fail_fast/best_effort` 语义一致，不改 runner 主状态机行为。
- 增加契约测试、race 测试与 benchmark baseline：
  - 新增 `BenchmarkCA2ExternalRetrieverTrendAggregation`。
- 同步文档与实现口径（README/docs/acceptance/roadmap/contract index）。

## Capabilities

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 扩展 CA2 external retriever 观测阈值配置与 provider 趋势查询契约。
- `context-assembler-stage-routing`: 扩展 Stage2 观测字段与阈值信号发射语义，不改变 stage policy 行为。
- `action-timeline-events`: 补充 external retriever 阈值命中在 Run/Stream 等价语义上的约束。

### New Capabilities
- None.

## Impact

- 受影响模块：
  - `runtime/config`
  - `runtime/diagnostics`
  - `context/provider` / `context/assembler`
  - `observability/event`
  - `integration`（benchmark + contract tests）
- 受影响文档：
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
  - `docs/mainline-contract-test-index.md`
- 兼容性：
  - 新增字段与查询能力均为增量扩展，现有消费者可继续读取旧字段。
  - 本期不新增自动策略动作，默认行为不变。
