## Why

当前主干已完成多 Provider、CA1-CA4、HITL H2-H4 和多 Agent 示例，但在高并发工具扇出与取消风暴场景下，缺少统一的背压与取消传播收敛基线。  
在进入 A2A/多 Agent 更复杂编排前，需要先加固 runner 主干稳定性，避免 goroutine 泄漏、取消语义漂移和不可观测退化。

## What Changes

- 在 `core/runner` 收敛取消传播与背压策略，默认背压策略为 `block`，保证语义稳定与 fail-fast 一致性。
- 通过 runtime 配置接入并发与背压参数，不新增对外 API；保留后续 `drop_low_priority` 策略扩展 TODO。
- 扩展 Run/Stream 主干契约测试，覆盖 `tool/mcp/skill` 路径在取消风暴下的一致性与无泄漏基线。
- 增加最小诊断字段：`cancel_propagated_count`、`backpressure_drop_count`、`inflight_peak`。
- 在性能与观测基线中新增 `p95 latency` 与 `goroutine peak` 验收口径，统一文档与实现。

## Capabilities

### New Capabilities
无

### Modified Capabilities
- `runtime-concurrency-control`: 增加取消风暴与背压策略的主干行为要求（默认 `block`、Run/Stream 语义一致、tool/mcp/skill 纳入）。
- `runtime-config-and-diagnostics-api`: 增加并发与背压配置项及最小诊断字段契约。
- `action-timeline-events`: 增加取消传播与背压命中的可观测 reason/code 与统计一致性要求。

## Impact

- Affected code:
  - `core/runner/*`
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `observability/event/*`
  - `integration/*`
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/v1-acceptance.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
- API impact: 不新增公开 API，仅增强现有配置与诊断契约。
