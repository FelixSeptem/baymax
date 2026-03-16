## Why

当前并发基线 R5 已完成 `block` 背压与取消风暴收敛，但在高 fanout 场景下仍缺少“可控降载”策略，导致队列压力下尾延迟和等待放大。为在不破坏 Run/Stream 语义稳定的前提下提升高负载弹性，需要引入 `drop_low_priority` 作为可选背压模式，并通过配置规则显式定义可丢弃范围。

## What Changes

- 在并发背压模式中新增 `drop_low_priority` 枚举，并保持默认模式不变（仍为 `block`）。
- 首期只在 `tool/local` dispatch 路径启用低优先级丢弃策略，不扩展到 `mcp/skill`。
- 优先级判定仅基于配置规则（tool 名称 + keyword），不引入参数显式优先级字段。
- 新增可配置“允许被丢弃的优先级集合”控制项，支持策略收敛与灰度。
- 语义收敛：同一轮工具调用若全部被 drop，立即 fail-fast 终止。
- 新增 timeline reason：`backpressure.drop_low_priority`，并与 run 诊断字段保持一致。
- 扩展契约测试与 benchmark，按相对提升百分比 + `p95` + `goroutine-peak` 验收。
- 同步更新 README、runtime-config-diagnostics、v1-acceptance、roadmap、contract-index。

## Capabilities

### New Capabilities
- `runtime-backpressure-drop-low-priority`: 定义 `drop_low_priority` 背压模式、配置规则、fail-fast 终止语义与可观测契约。

### Modified Capabilities
- `runtime-concurrency-control`: 扩展背压枚举与高负载行为约束（仅 tool/local 范围）。
- `action-timeline-events`: 增加 `backpressure.drop_low_priority` reason 的事件语义。
- `runtime-config-and-diagnostics-api`: 增加 drop 策略配置字段与诊断语义对齐要求。

## Impact

- Affected code:
  - `core/runner/*`
  - `tool/local/*`
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `observability/event/*`
  - `integration/*`
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/v1-acceptance.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
- API impact:
  - 不新增公开 API；配置字段与枚举为兼容扩展。
