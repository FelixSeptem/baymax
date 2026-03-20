## Context

A26 引入了 manifest 兼容校验，A27 引入了 capability negotiation/fallback 语义，但 adapter 合同仍缺少稳定版本锚点：
- 不同迭代中的字段/reason taxonomy 变更难以判定是否兼容；
- 缺少 golden replay 基线，回归发现滞后；
- 现有 gate 能检查“当下是否通过”，不能验证“与既有 profile 是否一致”。

因此需要在 `0.x` 阶段补齐 profile version 与 replay gate，为持续新增能力提供可控演进边界。

## Goals / Non-Goals

**Goals:**
- 定义 adapter contract profile version 规则（首版 `v1alpha1`）。
- 固化 runtime profile 支持窗口（默认 `current + previous`）。
- 建立离线 deterministic replay 基线，覆盖 manifest + negotiation + taxonomy。
- 将 replay 检查纳入 quality gate 阻断。
- 保持 shell/PowerShell 语义一致，默认 fail-fast。

**Non-Goals:**
- 不引入远程 registry 或 profile 分发服务。
- 不建立多版本长期兼容矩阵（仅 current + previous）。
- 不引入 warn-only 默认模式。

## Decisions

### 1) profile 命名采用显式语义版本标签
- 方案：`contract_profile_version` 使用 `v<major>alpha<minor>` 形式，首版 `v1alpha1`。
- 原因：与 pre-1 阶段迭代节奏匹配，可明确语义演进。
- 备选：日期版本。拒绝原因：无法表达兼容关系。

### 2) runtime 支持窗口默认 `current + previous`
- 方案：仅支持当前 profile 与上一版本 profile。
- 原因：控制复杂度并保留最小迁移缓冲。
- 备选：无限回溯支持。拒绝原因：维护成本高且契约不清晰。

### 3) 不兼容与回放漂移默认 fail-fast
- 方案：profile 不兼容、fixture 漂移、taxonomy 不一致均直接 non-zero 失败。
- 原因：adapter 属于契约边界，必须强约束。
- 备选：warn-only。拒绝原因：会放大漂移积累。

### 4) replay fixture 作为 versioned source-of-truth
- 方案：以 profile 版本分目录维护 fixtures，变更必须同 PR 更新差异说明。
- 原因：确保“可追溯 + 可回归 + 可审查”。
- 备选：动态生成期望值。拒绝原因：难以发现不期望变化。

## Risks / Trade-offs

- [Risk] profile 迭代过快导致 fixture 维护负担增加  
  → Mitigation: 限制窗口为 `current + previous` 并要求每次变更附差异说明。

- [Risk] 回放断言过细导致脆弱  
  → Mitigation: 断言语义字段与 reason taxonomy，避免过度依赖非关键噪声字段。

- [Risk] A27/A28 并行实施造成边界混淆  
  → Mitigation: A27 负责协商语义本身；A28 负责 profile 化与回放治理。

## Migration Plan

1. 新增 profile 版本模型与兼容窗口校验。
2. 在 manifest/negotiation 结构中接入 `contract_profile_version`。
3. 建立 profile 目录化 replay fixtures（manifest + negotiation + reason taxonomy）。
4. 增加 `check-adapter-contract-replay.*` 并接入 quality gate。
5. 更新 docs/index/roadmap 及升级指引。

回滚策略：
- 若回放 gate 稳定性不足，可先回滚 quality-gate 接入点；
- profile 字段与 fixture 可保留为非阻断路径迭代。

## Open Questions

- 当前无阻塞问题；按推荐值冻结：
  - 初始 profile: `v1alpha1`
  - 兼容窗口: `current + previous`
  - 默认 fail-fast
  - warn-only 默认关闭
