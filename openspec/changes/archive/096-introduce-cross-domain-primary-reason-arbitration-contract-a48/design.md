## Context

A40/A41/A43/A44/A46/A47 已分别收敛 readiness、timeout-resolution、adapter-health、admission 与 cross-domain replay fixture 基线。当前缺口是“跨域 primary reason 选取”的单点权威规则：当多个域同时命中时，不同路径可能给出不同 primary code/domain，影响诊断解释、准入判定可追溯性与回放稳定性。

A48 目标是在 lib-first 边界内冻结一套 deterministic arbitration 语义，并让 runtime/readiness/admission/diagnostics/replay/gate 使用同一规则。

## Goals / Non-Goals

**Goals:**
- 固化跨域 primary reason 优先级与 tie-break 规则，保持 deterministic。
- 在 readiness/admission/diagnostics/replay 路径统一消费 arbitration 结果。
- 增加 additive 可观测字段并保持 replay idempotency。
- 将 arbitration drift 纳入 quality gate 阻断。

**Non-Goals:**
- 不引入平台化控制面或外部协调状态存储。
- 不改变业务终态机和执行路径，仅治理解释层语义。
- 不引入可配置优先级策略（首版固定规则）。

## Decisions

### Decision 1: 使用固定优先级裁决（不开放配置）

- 方案：
  1) timeout exhausted/reject
  2) readiness blocked
  3) adapter required unavailable
  4) readiness degraded / adapter optional unavailable
  5) 其余 warning/info
- 原因：先冻结语义，避免策略维度扩大导致门禁难以收敛。
- 备选：可配置优先级。缺点：测试矩阵膨胀与跨环境不可比。

### Decision 2: 同级冲突使用 canonical code 字典序 tie-break

- 方案：同级候选按 canonical code 词典序取最小值作为 primary。
- 原因：简单、可复现、易实现、便于 replay 对账。

### Decision 3: 输出统一 primary 字段

- 方案：新增并统一维护：
  - `runtime_primary_domain`
  - `runtime_primary_code`
  - `runtime_primary_source`
  - `runtime_primary_conflict_total`
- 原因：让排障与回放比对有单一入口，不依赖各域私有字段拼装。

### Decision 4: A47 fixture 作为 A48 语义回归主门禁

- 方案：A48 新增/更新 fixture case，A47 gate 负责阻断 drift。
- 原因：避免重复造 gate，延续当前收敛路径。

## Risks / Trade-offs

- [Risk] 固定优先级可能无法覆盖少数业务偏好
  -> Mitigation: 在 A48 明确记录 rule source，后续如需配置化另开提案。

- [Risk] 字段新增导致消费方解析差异
  -> Mitigation: 保持 `additive + nullable + default`，并在文档中给出兼容窗口。

- [Risk] 同级字典序 tie-break 可读性一般
  -> Mitigation: diagnostics/replay 输出保留候选列表摘要与冲突计数。

## Migration Plan

1. 在 runtime 实现 cross-domain arbitration helper 并接入 readiness/admission。
2. 在 diagnostics 与 recorder 落地 primary 字段与冲突聚合。
3. 在 replay tooling 增加 arbitration fixture 对账与 drift 分类。
4. 在 integration 与 quality gate 接入 arbitration suites（Run/Stream/replay）。
5. 更新文档索引与状态快照。

## Open Questions

- 是否在 A48 首版输出“secondary candidates”摘要列表（建议输出受控有界摘要）。
- 是否在后续 A49+ 评估“按 operation profile 分组优先级”扩展（本提案不做）。
