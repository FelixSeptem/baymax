## Context

A12/A13/A14 已把多代理通信闭环与 tail 治理收敛到可回归状态；A15 正在补 workflow 图复用表达力。下一阶段主要缺口是“协作原语抽象层”——当前 handoff/delegation/aggregation 语义分散于 teams/workflow/composer 的具体路径，缺少统一契约与可复用 API。A16 在不平台化的前提下引入协作原语层，目标是统一语义与治理，而不是扩展控制面能力。

## Goals / Non-Goals

**Goals:**
- 提供统一 `orchestration/collab` 协作原语接口（handoff/delegation/aggregation）。
- 固化聚合策略首版：`all_settled`、`first_success`，默认 `all_settled`。
- 固化失败策略默认 `fail_fast`，并保持 Run/Stream 语义等价。
- 保持协作原语与 sync/async/delayed 三种通信方式可组合。
- 统一 timeline/diagnostics/contract gate 口径并保持兼容窗口语义。

**Non-Goals:**
- 不做平台化能力（多租户/RBAC/控制台）。
- 不引入外部 MQ/任务平台。
- 不重写 scheduler/composer 主状态机。
- 不新增顶级 timeline reason namespace（不引入 `collab.*`）。

## Decisions

### 1) 协作原语独立成新包 `orchestration/collab`
- 方案：以库内新包承载协作原语抽象，teams/workflow/composer 作为消费者。
- 原因：隔离语义层与编排层，降低重复实现与漂移。
- 备选：继续模块内各自实现。拒绝原因：长期维护成本高，契约难统一。

### 2) 聚合策略首版仅支持 `all_settled` 与 `first_success`
- 方案：限制首版策略面，默认 `all_settled`。
- 原因：覆盖核心需求同时控制复杂度。
- 备选：首版引入 quorum/vote。拒绝原因：策略面过大，验证矩阵成本高。

### 3) 失败策略默认 `fail_fast`
- 方案：统一默认失败行为为 fail-fast，减少长链路级联浪费。
- 原因：与现有可靠性治理方向一致，易于排障。
- 备选：best-effort 默认。拒绝原因：更易掩盖错误并扩大尾部开销。

### 4) 协作原语层默认不做内建重试
- 方案：原语层重试默认关闭，重试留在 scheduler/retry 治理路径。
- 原因：避免双重重试策略冲突。
- 备选：原语层也支持默认重试。拒绝原因：故障分类与退避策略容易分叉。

### 5) feature flag 默认关闭
- 方案：新增协作原语开关默认 `false`，显式启用。
- 原因：兼容优先，避免对既有业务路径造成行为突变。
- 备选：默认启用。拒绝原因：升级风险高。

### 6) timeline reason 复用既有 canonical namespace
- 方案：协作原语相关 reason 必须映射到 `team.*`/`workflow.*`/`a2a.*`/`scheduler.*`。
- 原因：保持主干 reason taxonomy 连续性与 gate 可复用性。
- 备选：新增 `collab.*`。拒绝原因：增加 taxonomy 分叉与迁移成本。

### 7) 兼容语义保持 `additive + nullable + default`
- 方案：新增配置/诊断字段全部 additive，不改旧字段语义。
- 原因：保障旧消费者平滑兼容。
- 备选：替换旧字段。拒绝原因：破坏兼容窗口。

## Risks / Trade-offs

- [Risk] 新抽象层引入额外心智负担  
  → Mitigation: 首版只覆盖三类原语与两种聚合策略，保持 API 最小化。 

- [Risk] 原语层与现有模块语义不一致  
  → Mitigation: 强制 shared contract + Run/Stream 等价矩阵。 

- [Risk] 诊断字段膨胀  
  → Mitigation: additive 字段最小集 + bounded-cardinality 校验。 

- [Risk] 默认 fail-fast 在部分场景过于激进  
  → Mitigation: 保留可配置策略，但默认值明确且文档化。

## Migration Plan

1. 增加 `orchestration/collab` 原语抽象与数据模型。  
2. 在 composer/workflow/teams 接入原语层（先适配不改主流程）。  
3. 增加配置开关与策略字段（默认关闭）。  
4. 扩展 timeline/diagnostics additive 字段与 reason 映射。  
5. 增加合同测试矩阵与 shared gate 阻断。  
6. 同步文档与主干测试索引。  

回滚策略：
- 关闭协作原语 feature flag，回退到现有模块内协作路径；
- additive 字段保留，不影响旧消费者。

## Open Questions

- 当前默认值与策略已冻结（按推荐值），暂无阻塞性开放问题。
