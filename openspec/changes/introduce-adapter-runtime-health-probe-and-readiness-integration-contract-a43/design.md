## Context

Baymax 在 adapter 维度已经具备静态契约（manifest）、能力协商（capability）与离线一致性验证（conformance harness）。但这些能力主要覆盖“接入前与调用时语义”，缺少“运行中可用性”的统一契约。

当前运行时虽然已有 readiness preflight（A40），但 readiness 的核心覆盖在 scheduler/mailbox/recovery 等基础域，尚未把 adapter 运行健康纳入同一判定与降级模型。结果是：
- required adapter 失效时，阻断策略不统一；
- optional adapter 退化时，降级与观测口径不一致；
- 调用方难以通过单一库级接口判定“可运行/可降级/应阻断”。

## Goals / Non-Goals

**Goals:**
- 引入 adapter runtime health probe 契约，统一状态分级与 finding taxonomy。
- 在 `runtime/config` 增加 `adapter.health.*` 配置并保持 `env > file > default`、fail-fast、热更新回滚一致性。
- 将 adapter health 纳入 readiness 结果映射，兼容 strict/non-strict 策略。
- 为 diagnostics 提供 adapter health additive 字段，满足可回放、可聚合、可查询。
- 将 adapter health 契约测试纳入 conformance 与 quality gate 阻断路径。

**Non-Goals:**
- 不引入平台化控制面、租户维度探针编排、外部服务发现中心。
- 不改变既有 manifest/capability 的 required/optional 语义定义。
- 不将 health probe 结果直接重写业务终态，仅影响 readiness 与降级决策。

## Decisions

### Decision 1: 使用三态健康模型并与 readiness 分层解耦

- 方案：adapter health 使用 `healthy|degraded|unavailable`，readiness 继续使用 `ready|degraded|blocked`。
- 映射：
  - required adapter `unavailable` -> readiness finding（strict 可升级 blocked）
  - optional adapter `unavailable` -> degraded finding + 可运行降级
  - `degraded` 始终保留 finding 并可观测
- 原因：保留 adapter 层语义表达力，同时复用 A40 readiness 判定框架。

### Decision 2: 默认关闭健康探测，逐步启用

- 方案：`adapter.health.enabled=false`、`adapter.health.strict=false`。
- 原因：避免对现有接入方形成突发阻断；先观测再收紧。
- 备选：默认开启严格模式。缺点是对当前外部 adapter 接入兼容风险高。

### Decision 3: 探测结果使用短 TTL 缓存避免探测风暴

- 方案：引入 `probe_timeout` 与 `cache_ttl`，在 TTL 内复用健康结果。
- 原因：减少热点调用下重复探测开销，控制外部依赖抖动。
- 备选：每次请求强制实时探测。缺点是放大尾延迟与外部依赖抖动。

### Decision 4: 诊断字段坚持 additive + bounded-cardinality

- 方案：新增 run-level additive 字段和计数聚合，不记录高基数自由文本。
- 原因：保持兼容窗口与查询成本可控，避免再次造成 diagnostics cardinality 漂移。
- 备选：直接输出每次 probe 原始细节。缺点是查询成本与存储膨胀。

### Decision 5: 通过 conformance + quality gate 双层阻断

- 方案：adapterconformance 增加 health matrix；quality gate 增加对应脚本/测试步骤。
- 原因：保证语义回归在 PR 前即可发现，且 shell/PowerShell 行为一致。

## Risks / Trade-offs

- [Risk] 健康探测误报导致 readiness 过度降级  
  -> Mitigation: 默认 non-strict + TTL 缓存 + canonical reason code，减少瞬时抖动影响。

- [Risk] 探测逻辑增加启动与热更新路径复杂度  
  -> Mitigation: 复用现有 config fail-fast/rollback 模式，新增字段最小化。

- [Risk] conformance 矩阵变大导致验证耗时上升  
  -> Mitigation: health 用例保持离线 deterministic，聚焦 required/optional 核心路径。

- [Risk] 与 A40 readiness 文档口径发生漂移  
  -> Mitigation: 同步更新 runtime-config-diagnostics 与 mainline index，并纳入 docs consistency gate。

## Migration Plan

1. 新增 adapter health 配置结构与默认值，补齐 validation 和热更新回滚测试。
2. 在 adapter 层定义 health probe 接口与标准结果模型，提供最小默认实现。
3. readiness preflight 接线 adapter health 映射逻辑，保持 strict/non-strict 一致。
4. diagnostics 新增 additive 字段并接入 QueryRuns/聚合输出。
5. conformance + quality gate 接入 adapter health contract suites。
6. 文档同步并执行全量验证命令（`go test ./...`、`go test -race ./...`、`check-quality-gate`、`check-docs-consistency`）。

## Open Questions

- health probe 是否需要区分 `startup_probe` 与 `runtime_probe` 两套超时（当前建议先单一超时）。
- optional adapter 退化计数是否需要按 adapter category 分桶（当前建议先 run-level 总量）。
