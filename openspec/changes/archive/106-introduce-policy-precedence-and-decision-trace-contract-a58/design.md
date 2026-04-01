## Context

A56 已归档后，A57 正在实施 sandbox egress 与 adapter allowlist。联调阶段会出现多个策略层同时参与决策的场景：`action gate`、`security s2`、`sandbox action/egress`、`allowlist`、`readiness/admission`。

当前缺口不是“有没有策略”，而是“同一请求如何跨策略层做唯一且可解释的裁决”。A58 的目标是在不改既有策略语义的前提下，冻结跨层 precedence 与 decision trace 合同，降低并行改造后的判定漂移风险。

## Goals / Non-Goals

**Goals**
- 冻结跨策略层 precedence matrix 与 deterministic tie-break。
- 冻结 `deny_source`、`winner_stage`、`policy_decision_path` 的解释字段语义。
- 将 precedence 输出贯通 preflight/admission/diagnostics/replay/gate。
- 保持 Run/Stream 等价与 deny side-effect-free。

**Non-Goals**
- 不引入平台化策略编排控制面（UI/RBAC/策略中心）。
- 不重写 A57 的 egress/allowlist 具体规则，仅收敛跨层裁决。
- 不改变 A48/A49/A50 的 primary reason taxonomy，仅新增与 policy stack 对齐字段。

## Decisions

### Decision 1: 采用单一路径 precedence evaluator，禁止分散裁决

- 方案：在运行态引入单一 policy stack evaluator，统一消费各层策略候选并输出 winner。
- 备选：各模块各自判定，再由调用方拼接。
- 取舍：单点 evaluator 更容易保证 Run/Stream 与多入口一致性，也便于 replay 与 gate 做稳定断言。

### Decision 2: 固化 stage precedence 顺序并作为 contract 字段暴露

- 方案：固定顺序为：
  1. `action_gate`
  2. `security_s2`
  3. `sandbox_action`
  4. `sandbox_egress`
  5. `adapter_allowlist`
  6. `readiness_admission`
- 取舍：避免“模块内局部最优”导致全局冲突，所有策略层都必须按同一序列参与仲裁。

### Decision 3: 同层冲突按 deterministic tie-break 收敛

- 方案：同一 stage 多个候选冲突时按 `canonical code lexical order` + `stable source order` 选 winner，并记录 `tie_break_reason`。
- 取舍：可解释且可重放，避免非确定性 map 遍历导致的随机 winner。

### Decision 4: decision trace 采用 additive 字段并复用现有 recorder 主链路

- 方案：通过 `RuntimeRecorder` 单写入口新增 `policy_decision_path`、`deny_source`、`winner_stage`、`tie_break_reason`。
- 取舍：保持观测写入边界不扩散，兼容现有 replay 和 QueryRuns。

### Decision 5: replay 与 gate 必须与合同行为同步冻结

- 方案：新增 `policy_stack.v1` fixture 与 drift taxonomy，并接入独立 gate。
- 取舍：避免“实现先行、门禁滞后”导致规则漂移在主线长期潜伏。

## Architecture

1. `runtime/config`
- 新增 `runtime.policy.precedence.*`、`runtime.policy.tie_breaker.*`、`runtime.policy.explainability.*`；
- 保持 `env > file > default`、非法配置 fail-fast、热更新原子回滚。

2. `runtime/security` + `core/runner`
- 汇总 action/s2/sandbox/allowlist/readiness 候选；
- 统一调用 precedence evaluator 输出 winner 与 decision trace。

3. `runtime/config/readiness` + `admission`
- preflight 输出 canonical finding 与候选列表；
- admission 消费 precedence 结果并保持 deny side-effect-free。

4. `runtime/diagnostics` + `RuntimeRecorder`
- 新增 policy additive 字段；
- 保持 bounded-cardinality 与 replay idempotency。

5. `tool/diagnosticsreplay` + gate
- 新增 `policy_stack.v1` loader、normalization、drift 分类；
- 新增 `check-policy-precedence-contract.*` 并接入 `check-quality-gate.*`。

## Risks / Trade-offs

- 风险：precedence 规则过于刚性导致特殊场景难以覆盖。  
  - 缓解：规则版本化（`runtime.policy.precedence.version`）+ explainability 字段保留上下文。

- 风险：新增字段提升诊断体积与查询成本。  
  - 缓解：沿用 A45 cardinality budget 与 A42 查询回归门禁阈值。

- 风险：A57 与 A58 并行实施引入联调冲突。  
  - 缓解：A58 不改 A57 具体策略逻辑，只负责跨层裁决与解释输出。

## Migration Plan

1. 增加 `runtime.policy.*` 配置 schema 与 validator。
2. 实现 policy stack evaluator 与 deterministic tie-break。
3. 接入 preflight/admission 并冻结 deny side-effect-free 路径。
4. 扩展 diagnostics + recorder policy 字段。
5. 新增 `policy_stack.v1` fixtures 与 drift 分类。
6. 接入 `check-policy-precedence-contract.*` 与 quality gate。
7. 同步 README/roadmap/mainline index/runtime config 文档。

## Open Questions

- None. A58 按“一次性冻结跨策略层 precedence 与 decision trace 合同”推进，不预留同主题拆案必需项。
