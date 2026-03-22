## Context

当前协作原语（`orchestration/collab`）虽然已经有 `composer.collab.retry.enabled` 配置字段，但执行与配置校验都显式拒绝开启 retry。  
这导致协作链路在 `delegation sync` 和 `async submit` 入口遇到短暂传输异常时无法自愈，只能直接失败并依赖上层二次调度。

在 A30-A32 主线持续收口的背景下，A33 需要补齐“受控重试”这一缺口，同时避免破坏既有边界：
- 保持 lib-first，不引入平台化控制面。
- 保持 scheduler/recovery 语义稳定，不做双重重试叠加。
- 保持 Run/Stream、memory/file 与 replay 幂等契约。

## Goals / Non-Goals

**Goals:**
- 为协作原语提供默认关闭、可显式开启的有界重试治理能力。
- 固化默认推荐值并纳入 `env > file > default`、fail-fast、热更新回滚语义。
- 固化重试范围与所有权：仅覆盖 sync/async submit 阶段，避免与 scheduler 重试域冲突。
- 扩展诊断字段与 shared gate 契约测试，确保语义可回归。

**Non-Goals:**
- 不引入外部 MQ、控制面、任务看板写操作。
- 不承诺 exactly-once，仅保持 at-least-once + idempotent convergence。
- 不改变 A32 的 async-await 终态收敛机制（callback/reconcile/timeout 仍按既有契约执行）。
- 不把 primitive retry 扩展到 accepted 后的 report/reconcile 阶段。

## Decisions

### 1) 重试默认关闭，推荐值冻结
- 决策：默认 `composer.collab.retry.enabled=false`，新增治理字段默认值：
  - `max_attempts=3`
  - `backoff_initial=100ms`
  - `backoff_max=2s`
  - `multiplier=2.0`
  - `jitter_ratio=0.2`
  - `retry_on=transport_only`
- 原因：保证默认行为无漂移，并为灰度启用提供确定基线。
- 备选：默认开启。拒绝原因：会扩大现网重放/风暴风险，且与 pre-1 保守策略不符。

### 2) 只对 transport 分类失败重试
- 决策：默认仅重试 transport 层失败；protocol/semantic 失败直接返回终态失败。
- 原因：传输失败最具暂时性，重试收益高；协议/语义失败通常应 fail-fast。
- 备选：全分类重试。拒绝原因：会放大无效请求与错误风暴。

### 3) 重试范围限制在 sync delegation 与 async submit 阶段
- 决策：primitive retry 仅作用于：
  - `DelegateSync` 的 submit/wait 入口错误；
  - `DelegateAsync` 的 submit accepted 前错误。
  对于 accepted 后的 report/await/reconcile 不追加 primitive retry。
- 原因：避免与 A31/A32 生命周期收敛域重叠。
- 备选：覆盖整个 async 生命周期。拒绝原因：会与 async-await 收敛契约冲突。

### 4) scheduler 管理路径保持单一重试所有权
- 决策：scheduler 管理的执行路径不叠加 primitive retry，防止出现 scheduler + primitive 双层重试。
- 原因：保证重试预算和失败分类可解释，避免指数级重试放大。
- 备选：允许双层重试。拒绝原因：预算不可控，诊断噪音高。

### 5) 重试延迟采用指数退避 + 抖动并可测试
- 决策：采用 `backoff_initial * multiplier^n` 上限 `backoff_max`，并叠加 `jitter_ratio`；实现需支持测试可验证的边界行为。
- 原因：与已有多代理治理策略对齐，降低瞬时冲击。
- 备选：固定间隔重试。拒绝原因：在高并发下易形成同步突刺。

### 6) 诊断字段采用 additive 扩展并要求 replay 幂等
- 决策：新增协作重试聚合字段（最小集合）并遵循 `additive + nullable + default`。
- 原因：不破坏旧消费者，同时提供治理信号。
- 备选：复用既有 `collab_*` 字段。拒绝原因：无法区分“执行失败”与“重试收敛”。

### 7) shared gate 纳入 collaboration retry suites
- 决策：在既有 `check-multi-agent-shared-contract.*` 门禁内新增重试语义阻断，不新增平行 gate。
- 原因：避免门禁分裂，保持主干收口路径单一。
- 备选：单独重试 gate。拒绝原因：维护成本更高且容易漏跑。

## Risks / Trade-offs

- [Risk] 开启后可能增加远端调用压力  
  → Mitigation: 默认关闭 + 有界重试 + backoff/jitter + transport-only 分类。

- [Risk] 重试分类映射漂移导致误重试  
  → Mitigation: 固化错误分层判定并用 contract suites 覆盖边界。

- [Risk] 与 scheduler/recovery 域叠加导致双重重试  
  → Mitigation: 明确单一所有权并在集成测试中校验“无双重重试”。

- [Risk] 新增诊断字段造成解析差异  
  → Mitigation: additive + nullable + default，不改变既有字段语义。

## Migration Plan

1. 在 `runtime/config` 扩展 `composer.collab.retry.*` 字段与校验逻辑（启动/热更新 fail-fast + rollback）。
2. 在 `orchestration/collab` 实现有界重试执行器与分类策略（transport-only 默认）。
3. 在 `orchestration/invoke` 与协作调用桥接中对齐 sync/async submit 范围边界。
4. 在 diagnostics/recorder 扩展协作重试聚合字段并保证 replay 幂等。
5. 增加 integration/contract suites，并并入 shared gate 阻断路径。
6. 同步更新 roadmap/readme/config-diagnostics/mainline-index 文档映射。

回滚策略：
- 运行时紧急回滚可直接设置 `composer.collab.retry.enabled=false`；
- 保留新增字段（兼容窗口内可空），不回滚历史已记录诊断结构。

## Open Questions

无阻塞项。推荐值已冻结并按本设计执行：
- `enabled=false`
- `max_attempts=3`
- `backoff_initial=100ms`
- `backoff_max=2s`
- `multiplier=2.0`
- `jitter_ratio=0.2`
- `retry_on=transport_only`

