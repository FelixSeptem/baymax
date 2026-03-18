## Context

当前代码中同步远程调用主链路已存在，但实现分散：
- A2A 客户端提供 `Submit` 与 `WaitResult`；
- workflow/teams/composer/scheduler 在各自适配路径内重复拼装同步调用流程；
- 超时、取消、终态和错误分层语义主要依赖调用方约定，缺乏单一契约入口。

这种结构在功能上可用，但会放大后续演进成本：
- 同一语义需要在多模块重复维护；
- 回归测试需要跨多个模块独立兜底；
- A12（异步主动回报）与 A13（业务级延后执行）落地时，容易出现路径间行为分叉。

A11 的定位是把“同步执行并等待结果”收敛成可复用且可测试的基础契约，不改变 lib-first 与既有默认行为。

## Goals / Non-Goals

**Goals:**
- 新增统一同步调用抽象，形成跨模块复用入口。
- 统一 `Submit -> WaitResult -> terminal normalize` 语义，覆盖超时、取消、错误分层、重试提示。
- workflow/teams/composer/scheduler 路径对齐统一契约，减少重复实现。
- 补齐契约测试矩阵并并入既有 shared-contract gate。
- 维持向后兼容，不破坏既有配置与 reason taxonomy。

**Non-Goals:**
- 不引入独立异步主动回报通道（留给 A12）。
- 不引入 `not_before/execute_at` 业务级延后调度能力（留给 A13）。
- 不引入平台化控制面、队列中间件或外部编排依赖。
- 不重构 scheduler QoS/DLQ/backoff 主逻辑。

## Decisions

### 1) 新增统一同步调用包 `orchestration/invoke`
- 方案：提供统一 `InvokeSync` 入口，封装 A2A 同步调用收敛逻辑。
- 原因：保持 orchestrator 领域复用，避免在 `workflow/teams/composer/scheduler` 分别复制语义。
- 备选：继续在各模块内保留本地拼装。拒绝原因：重复逻辑难以保证长期一致。

### 2) API 采用函数式最小接口
- 方案：`InvokeSync(ctx, client, req, options)` + 轻量结果结构，避免新增复杂对象生命周期。
- 原因：调用接入成本低，适合 lib-first 最小抽象。
- 备选：引入 manager/service 对象。拒绝原因：对当前规模属于过度设计。

### 3) `poll_interval` 默认值保持 `20ms`
- 方案：沿用现有 `WaitResult` 默认行为。
- 原因：兼容现有时序与测试基线，避免隐式行为变化。
- 备选：改为更大默认值降低轮询频率。拒绝原因：会改变现有响应时延和测试假设。

### 4) 超时与取消遵循 `context` 单一权威
- 方案：调用链不叠加额外全局超时，直接继承上游 `context`。
- 原因：避免双重超时冲突和诊断歧义。
- 备选：新增统一全局超时配置。拒绝原因：会引入新的优先级冲突与迁移成本。

### 5) callback 仅保留兼容钩子语义
- 方案：保留 callback 参数能力，但不将其定义为主动回报主路径。
- 原因：与现状兼容；主动回报将在 A12 单独设计。
- 备选：A11 直接升级为提交后独立回报机制。拒绝原因：范围扩大且与 A12 重叠。

### 6) Scheduler 终态映射保持兼容
- 方案：A2A `canceled` 在 scheduler terminal commit 继续按 `failed` 路径收敛，同时保留错误层信息。
- 原因：维持现有 commit 接口约束（`succeeded|failed`）与回归兼容。
- 备选：扩展 scheduler terminal 状态集合。拒绝原因：会扩大跨模块契约变更范围。

### 7) 不新增 timeline reason taxonomy
- 方案：A11 只收敛调用语义，不新增 reason 命名空间。
- 原因：降低迁移噪声，保持文档与门禁稳定。
- 备选：新增 `invoke.*` reason。拒绝原因：收益小于治理成本。

## Risks / Trade-offs

- [Risk] 统一调用入口引入跨模块耦合点  
  → Mitigation: 保持 API 最小化并限制在 orchestration 层，避免反向依赖扩散。

- [Risk] 默认行为“看起来不变”但边界条件发生细微变化  
  → Mitigation: 增加契约测试覆盖 `timeout/cancel/error-layer` 关键路径并纳入共享门禁。

- [Risk] scheduler 路径与新抽象对齐时出现回归  
  → Mitigation: 保持 terminal commit 数据结构不变，仅替换调用编排逻辑。

- [Risk] 文档与实现口径漂移  
  → Mitigation: 同步更新 `README`、诊断文档、contract index，并执行 docs consistency gate。

## Migration Plan

1. 新增 `orchestration/invoke` 并实现最小同步调用契约与单测。  
2. 先接入 scheduler A2A 适配路径，验证 retryable/error-layer 对齐。  
3. 接入 composer `ChildTargetA2A` 路径。  
4. 接入 workflow `StepKindA2A` 与 teams `TaskTargetRemote` 路径。  
5. 补全 integration/contract 测试矩阵并接入 shared gate。  
6. 更新文档与索引，执行全量质量门禁。  

回滚策略：
- 保留各模块原始路径的可回退点（按 commit 颗粒回滚调用接入），
- 不回滚新增字段与兼容性文档。

## Open Questions

- 当前无阻塞问题；A11 的参数与边界已按确认口径固定。
