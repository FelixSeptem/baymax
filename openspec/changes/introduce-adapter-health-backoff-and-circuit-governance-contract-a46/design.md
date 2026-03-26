## Context

A43 已建立 adapter health probe 基线（`healthy|degraded|unavailable`）并接入 readiness preflight；A44 已把 readiness 结果纳入 managed Run/Stream admission。当前缺口在于探测治理层：当外部 adapter 或网络短时抖动时，缺少统一 backoff/circuit 约束，会造成高频重试与 readiness 抖动放大。

同时，A45 正在推进 diagnostics cardinality 治理；A46 需要在新增 health 观测字段时同步遵守 bounded-cardinality 与 replay-idempotent 约束，避免“治理字段反向制造高基数”。

## Goals / Non-Goals

**Goals:**
- 引入 `adapter.health.backoff.*` 与 `adapter.health.circuit.*` 配置域，并保持 `env > file > default`、startup/hot-reload fail-fast + 回滚。
- 固化 adapter health circuit 状态机与 canonical 转移语义（`closed|open|half_open`）。
- 固化 probe 退避策略（指数 + 抖动）并定义半开探测预算，抑制探测风暴。
- 将 circuit/backoff 结果映射到 readiness canonical findings，保持 strict/non-strict 语义一致。
- 增加 diagnostics additive 字段与 conformance/gate 阻断套件，确保 replay idempotency 与 Run/Stream 等价。

**Non-Goals:**
- 不引入平台化控制面（UI/RBAC/多租户运维）。
- 不引入外部协调存储（如 Redis 全局熔断）。
- 不改变 Run/Stream 业务终态语义，仅治理 health probe 与 readiness 准入稳定性。
- 不替代 A45 的 cardinality 契约，仅对齐其约束。

## Decisions

### Decision 1: Backoff 默认开启并采用指数 + 抖动

- 方案：
  - `adapter.health.backoff.enabled=true`
  - `initial=200ms`
  - `max=5s`
  - `multiplier=2.0`
  - `jitter_ratio=0.2`
- 原因：在 adapter 不稳定时优先抑制探测放大，且默认值与现有 runtime 时延级别兼容。
- 备选：默认关闭 backoff。缺点：异常流量下探测风暴风险高。

### Decision 2: Circuit 状态机采用 closed/open/half_open 三态

- 方案：
  - `failure_threshold=3`
  - `open_duration=30s`
  - `half_open_max_probe=1`
  - `half_open_success_threshold=2`
- 转移规则：
  - `closed -> open`: 连续失败达到阈值；
  - `open -> half_open`: open 窗口到期；
  - `half_open -> closed`: 连续成功达到阈值；
  - `half_open -> open`: 任一失败即回到 open。
- 原因：语义简单、可解释、可验证，便于 conformance/gate 收敛。

### Decision 3: Readiness 映射保持 canonical taxonomy

- 方案：新增 circuit/backoff 相关 findings code，但保持 namespace 与 A43 一致（`adapter.health.*`）。
- 原因：避免 A43/A44 已归档语义漂移，减少运维侧规则改写成本。
- 备选：使用独立 namespace。缺点：会增加并行 taxonomy 成本。

### Decision 4: Diagnostics 字段保持 additive + bounded-cardinality

- 方案：新增聚合计数与有限状态字段，禁止输出高基数自由文本；状态与原因码使用受控集合。
- 原因：与 A45 同步，防止 health 治理字段本身带来查询性能回归。

## Risks / Trade-offs

- [Risk] backoff/circuit 默认开启可能延迟恢复探测
  -> Mitigation: 半开探测窗口 + 可配置阈值，保证恢复路径可调。

- [Risk] 状态机实现偏差导致 readiness 判定不稳定
  -> Mitigation: 在 conformance + shared gate 增加状态转移矩阵与 replay-idempotency 阻断测试。

- [Risk] 配置域增大导致误配概率上升
  -> Mitigation: startup/hot-reload 严格校验与原子回滚，非法值 fail-fast。

## Migration Plan

1. 在 `runtime/config` 引入 `adapter.health.backoff.*` 与 `adapter.health.circuit.*` 默认值、解析与校验。
2. 在 `adapter/health` 实现 backoff + circuit 状态机与半开探测预算。
3. 将状态机输出接入 `runtime/config/readiness` finding 分类与 strict/non-strict 收敛逻辑。
4. 在 `runtime/diagnostics` 增加 additive 字段并保证 replay 幂等聚合。
5. 在 `integration/adapterconformance` 与 `scripts/check-*.{sh,ps1}` 接入阻断套件并保持跨平台 parity。

## Open Questions

- 是否需要在后续版本支持“按 adapter 分组”的定制阈值覆盖（本提案先不做，保持全局默认）。
- 是否需要在未来引入跨进程共享熔断状态（本提案明确不做）。
