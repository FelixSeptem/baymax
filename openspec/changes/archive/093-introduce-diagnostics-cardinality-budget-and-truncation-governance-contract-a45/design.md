## Context

`runtime/diagnostics` 在 A40-A44 持续扩展后，`RunRecord` 和查询输出字段数量快速增长。A42 已提供 query 性能回归 gate，但主要关注 `ns/op`、`p95-ns/op`、`allocs/op` 的结果指标，尚未约束“输入数据形态”本身的膨胀风险。

当前缺口：
- map/list/string 字段缺统一预算约束；
- 超预算后的处理方式未统一（截断/拒绝策略分散）；
- 截断结果若非确定性，会导致 replay/query 对账漂移。

## Goals / Non-Goals

**Goals:**
- 引入 diagnostics cardinality budget 配置域，覆盖 map/list/string 三类高风险字段。
- 固化超预算策略：`truncate_and_record`（默认）与 `fail_fast`。
- 保障截断结果 deterministic（同输入同配置输出一致）。
- 增加 additive 观测字段并保持 replay idempotency。
- 在 quality gate 中新增 cardinality drift 阻断套件，保持 shell/PowerShell parity。

**Non-Goals:**
- 不引入外部存储、平台化日志系统或离线 ETL。
- 不改变业务状态机与 Run/Stream 业务语义。
- 不替代 A42 性能回归脚本，仅补充输入数据形态治理。

## Decisions

### Decision 1: 预算治理分三类并统一配置入口

- 方案：新增
  - `max_map_entries`
  - `max_list_entries`
  - `max_string_bytes`
- 原因：这三类是当前 diagnostics 膨胀主来源，且与 JSON 序列化成本直接相关。

### Decision 2: 默认使用 `truncate_and_record`

- 方案：超预算默认截断并记录摘要，避免因观测数据异常直接中断主流程。
- 原因：符合 lib-first 稳定性优先原则，兼顾可运行与可观测。
- 备选：默认 `fail_fast`。缺点是对用户流量更激进，回归风险高。

### Decision 3: 截断必须 deterministic

- 方案：对 map key 排序后截断；list 保持原顺序前 N 项；string 按字节截断并标记。
- 原因：保障 replay、query、contract test 的稳定对账能力。

### Decision 4: 诊断新增字段保持 additive + bounded-cardinality

- 方案：新增总量与字段摘要，但字段摘要使用受控集合，不写入高基数自由文本。
- 原因：避免治理字段自身再次成为高基数来源。

## Risks / Trade-offs

- [Risk] 默认截断可能掩盖上游异常膨胀  
  -> Mitigation: 强制记录 `budget_hit` 与 `truncated_fields`，并在 gate 里校验。

- [Risk] 截断逻辑引入额外 CPU 开销  
  -> Mitigation: 仅在超预算时走截断路径；与 A42 联动观察 query 性能变化。

- [Risk] `fail_fast` 策略被误开启导致兼容问题  
  -> Mitigation: 默认 `truncate_and_record`，并对 `overflow_policy` 非法值 fail-fast + 回滚。

## Migration Plan

1. 在 `runtime/config` 增加 `diagnostics.cardinality.*` 配置、默认值、校验与热更新回滚。
2. 在 `runtime/diagnostics` 添加预算检查与 deterministic 截断实现。
3. 为截断行为新增 additive 观测字段并确保 replay idempotency。
4. 在 integration 增加 cardinality contract suites（budget/truncation/replay/Run-Stream parity）。
5. 将 suites 接入 `check-quality-gate.*` 并更新文档索引与 roadmap。

## Open Questions

- `truncated_fields` 是否按固定枚举输出，还是允许有限动态字段名（建议固定枚举优先）。
- string 截断是否保留 UTF-8 边界校正（建议保留，避免乱码影响排障可读性）。
