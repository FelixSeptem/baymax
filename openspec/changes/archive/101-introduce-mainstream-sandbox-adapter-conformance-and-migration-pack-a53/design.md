## Context

A51 收敛了 sandbox 执行隔离 contract，A52 正在收敛运行治理 contract；当前瓶颈已经从“语义定义”转向“主流后端落地效率与一致性验证”。  
在缺少统一 adapter profile pack、模板、迁移映射、conformance matrix 的情况下，接入方会在 nsjail/bwrap/OCI/Windows Job 上各自实现 glue layer，导致：
- capability 声明和实际行为不一致；
- session lifecycle 语义漂移；
- readiness/gate 无法稳定阻断回归；
- 后端切换时重复改造成本过高。

## Goals / Non-Goals

**Goals:**
- 冻结主流 sandbox backend 的 adapter profile pack 与 manifest 声明语义。
- 提供 offline deterministic 的 backend matrix conformance harness。
- 提供可执行模板与迁移映射，降低 sandbox adapter onboarding 成本。
- 将 profile 可用性/兼容性接入 readiness preflight 与 quality gate。
- 保持 shell/PowerShell 验证语义等价，并支持独立 required-check 暴露。

**Non-Goals:**
- 不改变 A51/A52 的 sandbox action/rollout/capacity contract 语义。
- 不引入平台化控制面或跨租户调度治理。
- 不承诺各后端底层实现一致，仅要求 canonical 合同输出一致。

## Decisions

### Decision 1: 采用 profile-pack 作为接入主抽象，而非 backend-specific 分散脚本

- 方案：
  - 统一 `sandbox_profile_pack` 描述 backend + capability profile + session mode 支持矩阵。
  - Manifest 声明通过 profile id 与 backend enum 绑定，运行时只认 canonical 字段。
- 取舍：
  - 增加前期 schema 设计成本，但显著降低后续接入语义分叉。

### Decision 2: Conformance harness 以 backend matrix + 场景分层执行

- 方案：
  - Matrix 维度：`linux_nsjail|linux_bwrap|oci_runtime|windows_job`。
  - 场景维度：capability negotiation、session lifecycle、fallback/error taxonomy。
  - 平台不可用 backend 允许跳过，但跳过必须可审计并有 deterministic reason。
- 取舍：
  - 测试编排更复杂，但可把跨后端漂移前置到 CI 阻断。

### Decision 3: Template 与 migration mapping 必须可执行并绑定 conformance case

- 方案：
  - 每个模板都映射到至少一个 conformance case id。
  - migration mapping 采用 capability-domain + code-snippet 双结构，覆盖旧 command wrapper 到新 adapter profile 的迁移。
- 取舍：
  - 文档维护成本上升，但可避免“示例能看不能跑”的失真。

### Decision 4: Readiness preflight 只做 profile 可用性判定，不复刻执行细节

- 方案：
  - preflight 新增 `sandbox.adapter.*` finding，关注 profile 缺失、host 不兼容、backend 不支持。
  - 执行细节与运行时策略继续由 A51/A52 contract 承担。
- 取舍：
  - 职责边界清晰，避免 readiness 与 runner 语义重复。

### Decision 5: Gate 独立化，避免把 adapter conformance 混入通用测试噪声

- 方案：
  - 新增 `check-sandbox-adapter-conformance-contract.sh/.ps1`。
  - 接入 `check-quality-gate.*`，并在 CI 作为独立 required-check 候选暴露。
- 取舍：
  - CI 时长略增，但定位回归更直接。

## Risks / Trade-offs

- [Risk] profile pack 设计过细导致接入门槛上升。  
  -> Mitigation: 默认 profile 最小必填，进阶字段采用 additive 扩展。

- [Risk] backend matrix 在不同 runner 上可用性不一致。  
  -> Mitigation: 平台条件化执行 + 强制输出 skip reason + required minimal matrix。

- [Risk] 模板与实现漂移。  
  -> Mitigation: 模板必须绑定 conformance case，drift 由 gate 阻断。

- [Risk] readiness finding 过多影响可读性。  
  -> Mitigation: 固定 canonical finding namespace 与 severity 映射。

## Migration Plan

1. 扩展 adapter manifest schema，新增 sandbox backend/profile/session 声明字段与校验器。
2. 在 conformance harness 增加 backend matrix 场景与 drift 分类输出。
3. 增加 sandbox adapter template + migration mapping，并绑定 conformance case id。
4. 在 readiness preflight 增加 `sandbox.adapter.*` findings 与 strict/non-strict 分类。
5. 新增 sandbox adapter conformance gate 脚本并接入 quality gate。
6. 增加 `sandbox.v1` profile replay fixtures，保持既有 profile fixture 向后兼容。
7. 更新 roadmap/readme/mainline index/docs 并执行 docs consistency 校验。

## Open Questions

- None for A53 scope. 本提案聚焦主流 sandbox adapter 接入 DX 与 conformance 治理，后续仅做 profile pack 增量扩展，不再拆“同类接入治理”子提案。
