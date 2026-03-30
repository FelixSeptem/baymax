## Context

A51 已将 sandbox 接入 contract 一次性冻结在执行隔离层（动作决策、ExecSpec/ExecResult、capability negotiation、readiness/admission、可观测、回放、门禁）。  
当前缺口不在“能不能 sandbox”，而在“如何安全上线 sandbox”：不同业务会自己实现灰度开关、健康阈值与容量保护，容易导致跨路径不一致、回滚不可重复、后端语义漂移不可阻断。

因此 A52 聚焦运行治理层 contract：把 rollout、health budget、capacity action、freeze/rollback 变成可配置、可解释、可回放、可门禁的统一语义，并显式复用 A51 的底层契约。

## Goals / Non-Goals

**Goals:**
- 定义 sandbox rollout phase 状态机与合法迁移，避免业务侧自定义语义分叉。
- 定义健康预算输入与 breach 判定 contract，支撑自动冻结与 deterministic 回滚。
- 定义容量治理动作（`allow|throttle|deny`）并接入 readiness/admission 统一判定。
- 将 rollout/capacity/freeze 语义写入 timeline/diagnostics/replay/gate，全链路可回归。
- 保持 Run/Stream 等价与 side-effect free deny 语义。

**Non-Goals:**
- 不新增或替换 A51 的 ExecSpec/ExecResult/capability contract。
- 不引入平台化控制面（多租户运营面板、中心化调度控制平面）。
- 不承诺跨主机全局容量编排，仅定义单 runtime contract 语义。

## Decisions

### Decision 1: rollout phase 采用固定状态机并限制迁移边界

- 方案：
  - Canonical phase：`observe|canary|baseline|full|frozen`。
  - 合法迁移：`observe->canary->baseline->full`；任意活动态可进入 `frozen`；`frozen` 仅允许进入 `canary` 或 `observe`。
  - 非法迁移在 startup/hot reload fail-fast。
- 取舍：
  - 固定状态机牺牲灵活性，但可显著降低灰度脚本分叉和回放复杂度。

### Decision 2: 健康预算输入冻结为五类 canonical 指标

- 方案：
  - 预算输入固定为：
    - launch failure rate
    - timeout rate
    - violation rate
    - p95 latency delta
    - admission deny rate
  - 使用固定 window 和 breach 次数阈值触发 freeze。
- 取舍：
  - 指标范围更窄，但可保证跨后端可比性与 deterministic breach 判定。

### Decision 3: 容量治理动作保持三态并接入 admission 语义

- 方案：
  - 容量动作：`allow|throttle|deny`。
  - `throttle` 走 degraded policy（`allow_and_record|fail_fast`）；
  - `deny` 一律 side-effect free。
- 取舍：
  - 不引入复杂优先级队列策略，保持 contract 简洁且可测试。

### Decision 4: 自动冻结必须与显式解冻 token 绑定

- 方案：
  - breach 达阈值自动进入 `frozen`。
  - 通过 `cooldown` + `manual_unfreeze_token` 才允许退出 frozen。
  - freeze reason code 固定在 `sandbox.rollout.*` namespace。
- 取舍：
  - 增加运维操作步骤，但避免“短周期抖动自动反复开关”。

### Decision 5: 可观测性继续走 additive + single-writer 路径

- 方案：
  - `runtime/diagnostics` 只增不改字段（nullable + default）。
  - `RuntimeRecorder` 维持单写与幂等规则。
  - timeline 增加 rollout/capacity/freeze canonical reasons。
- 取舍：
  - 字段数量增加，但能保持兼容窗口并保证 replay 可断言。

### Decision 6: 质量门禁增加独立 rollout governance gate

- 方案：
  - 新增 `check-sandbox-rollout-governance-contract.sh/.ps1`。
  - 接入 `check-quality-gate.*`，并暴露为独立 required-check 候选。
- 取舍：
  - 增加 CI 时长，但能阻断 rollout/freeze/capacity 语义漂移。

## Risks / Trade-offs

- [Risk] 健康预算阈值过紧导致误冻结。  
  -> Mitigation: 默认 `phase=observe`，先 canary 小流量验证并记录窗口内 breach 分布。

- [Risk] 容量 deny 在突发高峰下放大用户侧失败率。  
  -> Mitigation: 提供 `throttle + allow_and_record` 过渡策略，并强制输出 admission explainability。

- [Risk] 后端能力波动导致 freeze 频繁触发。  
  -> Mitigation: 使用 cooldown + manual unfreeze token，避免自动抖动回切。

- [Risk] 新增字段影响 QueryRuns 性能。  
  -> Mitigation: A52 同步纳入 diagnostics-query benchmark sandbox-enriched 数据集阈值阻断。

## Migration Plan

1. 在 `runtime/config` 增加 `security.sandbox.rollout.*` 与容量参数，并补齐 startup/hot reload fail-fast 测试。
2. 在 readiness preflight 中接入 rollout/freeze/capacity finding，保持 strict/non-strict 映射稳定。
3. 在 admission 路径接入 `allow|throttle|deny` 与 frozen fail-fast，并验证 deny side-effect free。
4. 在 timeline/diagnostics/RuntimeRecorder 中补齐 rollout/capacity/freeze additive 字段与 reason taxonomy。
5. 在 replay tooling 增加 `a52.v1` fixture 与 drift 分类（phase/health/capacity/freeze）。
6. 在 quality gate 增加 rollout governance gate，接入 shell/PowerShell parity 与独立 required-check。
7. 更新 roadmap/readme/mainline index/runtime diagnostics 文档并执行 docs consistency 校验。

## Open Questions

- None for A52 scope. 本提案聚焦“运行治理层 contract”，后续只做同 contract 下的实现扩展与阈值参数调优，不再拆 rollout 语义子提案。
