## Context

在 A51-A53 已归档语义中，sandbox contract 已覆盖执行隔离、rollout/capacity、主流 backend 接入 conformance；但网络外呼与 adapter 供应链准入尚未形成统一合同。A56 正在推进 ReAct tool loop，工具执行频次提高后，egress/allowlist 缺口会直接放大为运行与合规风险。

A57 目标是在不引入平台化控制面的前提下，复用现有治理主链路（`runtime/config`、`RuntimeRecorder`、readiness/admission、replay、quality gate）一次性补齐：
- sandbox egress policy contract；
- adapter allowlist activation contract；
- 观测、回放、门禁的一体化阻断链路。

## Goals / Non-Goals

**Goals**
- 新增 `security.sandbox.egress.*` 配置域并冻结 deny-first 默认语义。
- 新增 `adapter.allowlist.*` 激活边界，禁止未授权 adapter 进入运行路径。
- 把 egress/allowlist finding 纳入 readiness + admission，确保执行前阻断可解释且 side-effect-free。
- 新增 `sandbox_egress.v1` replay fixture 与 drift taxonomy，保证回归可重放。
- 新增独立 A57 gate 并接入 shell/PowerShell parity + CI required-check 候选。

**Non-Goals**
- 不引入平台化控制面（RBAC、多租户运营界面）。
- 不改 A56 ReAct loop 主合同语义，仅复用其执行链路。
- 不承诺第三方网络策略引擎实现一致，仅冻结 canonical contract 输出。

## Decisions

### Decision 1: Egress 治理由 runtime config 驱动，默认 deny-first

- 方案：在 `security.sandbox.egress.*` 定义策略（enabled/default_action/by_tool/allowlist/profile），默认 `deny`。
- 取舍：默认拒绝可降低误放开风险；需要显式配置才能开放网络出口。

### Decision 2: Adapter allowlist 作为激活前 fail-fast 边界

- 方案：adapter 激活前校验 allowlist（publisher/id/version/signature 状态），不通过即阻断加载。
- 取舍：把风险前置到 admission 前，减少运行期不确定行为。

### Decision 3: Readiness/Admission 统一消费 egress/allowlist findings

- 方案：preflight 输出 canonical finding，admission 只做 deterministic 映射并保持 side-effect-free deny。
- 取舍：避免执行期遇错再兜底，提升排障与审计可解释性。

### Decision 4: Replay 与 Gate 同步冻结

- 方案：新增 `sandbox_egress.v1` fixture，drift class 直接接入 quality gate。
- 取舍：避免“代码先行、治理滞后”导致语义漂移积累。

## Architecture

1. `runtime/config`
- 新增 `security.sandbox.egress.*`、`adapter.allowlist.*` 配置域；
- 保持 `env > file > default`、启动 fail-fast、热更新原子回滚。

2. `runtime/security` + `core/runner`
- sandbox tool dispatch 前做 egress 决策；
- 对违规外呼统一输出 canonical violation code。

3. `adapter/manifest` + activation
- 在 manifest 校验链路加入 allowlist 元信息；
- 非 allowlisted adapter 直接 fail-fast，禁止进入 runtime。

4. `runtime/config/readiness` + admission
- preflight 增加 `sandbox.egress.*` 与 `adapter.allowlist.*` findings；
- admission 对应 `allow|deny` 映射，deny 保持无副作用。

5. `runtime/diagnostics` + `RuntimeRecorder`
- 新增 A57 additive 字段（egress decision/violations/allowlist state）；
- 保持 bounded-cardinality + replay idempotency。

6. `tool/diagnosticsreplay` + quality gate
- 新增 `sandbox_egress.v1` schema + drift classes；
- 新增 `check-sandbox-egress-allowlist-contract.*` 并接入 `check-quality-gate.*`。

## Risks / Trade-offs

- 风险：策略过严造成误拒绝，影响可用性。  
  - 缓解：提供 per-tool 显式放开与可观测分类，不做隐式 allow。

- 风险：allowlist 元信息治理负担上升。  
  - 缓解：最小必填字段 + conformance 自动校验 + 文档迁移映射。

- 风险：新增字段导致 diagnostics 查询成本上升。  
  - 缓解：沿用 A45 cardinality budget 与 A42 查询回归门禁阈值。

## Migration Plan

1. 增加配置 schema 与 validator（egress + allowlist）。
2. 实现 runtime/security 与 adapter activation 的执行边界。
3. 接入 readiness/admission finding 与 explainability。
4. 扩展 diagnostics/replay fixtures 与 drift taxonomy。
5. 接入 quality gate 和 CI required-check。
6. 同步 README/roadmap/mainline index/runtime config 文档。

## Open Questions

- None. A57 范围按“一次性补齐 egress + allowlist 主合同”收敛，不预留同主题拆案必需项。
