## Context

在 A14-A17 持续收敛多代理契约后，诊断字段和组合路径显著增加。当前库接口以 `RecentRuns/RecentCalls/RecentSkills` 为主，适合最近窗口浏览，但不适合按 `run/team/workflow/task` 执行域进行统一检索。A18 聚焦补齐统一查询 API，保持 lib-first，不引入外部查询服务。

## Goals / Non-Goals

**Goals:**
- 提供统一查询接口，支持 run/team/workflow/task 维度过滤。
- 固化分页、排序和游标语义，保证跨调用稳定性。
- 固化参数校验 fail-fast 语义与空集返回语义。
- 保持与既有 `Recent*` API 兼容共存。
- 将查询契约纳入质量门禁和主干索引。

**Non-Goals:**
- 不引入外部数据库或搜索引擎。
- 不做全文检索或复杂 DSL 查询语言。
- 不做平台化 API 服务层。
- 不移除现有 `Recent*` 接口。

## Decisions

### 1) 查询 API 采用统一 QueryRequest / QueryResult 模型
- 方案：在 runtime diagnostics 提供统一查询结构，manager 透出对应 API。
- 原因：减少调用方碎片化拼接逻辑，便于持续扩展。
- 备选：继续扩展多个 `Recent*` 变体。拒绝原因：接口膨胀且语义分散。

### 2) 过滤组合语义固定为 `AND`
- 方案：多维条件采用 AND 组合，避免歧义。
- 原因：结果可预测且更安全。
- 备选：支持 OR。拒绝原因：首版复杂度高且易引入误解。

### 3) 分页默认值固定 50，上限 200
- 方案：默认 page size=50，max=200，越界参数 fail-fast。
- 原因：在性能与可用性之间平衡，并防止单次查询放大。
- 备选：无上限或默认 1000。拒绝原因：内存和延迟风险。

### 4) 排序默认 `time desc`
- 方案：默认按时间倒序返回。
- 原因：与现有 Recent 语义一致，最符合观测场景。
- 备选：升序。拒绝原因：不符合排障常用路径。

### 5) 游标采用 opaque cursor
- 方案：返回不透明游标字符串，隐藏内部实现细节。
- 原因：允许内部索引演进而不破坏兼容。
- 备选：公开 offset。拒绝原因：在动态数据下稳定性差。

### 6) `task_id` 不存在返回空集，不报错
- 方案：对于合法但不存在的 task_id 查询返回空结果。
- 原因：查询语义应区分“参数非法”与“无匹配结果”。
- 备选：返回 not_found 错误。拒绝原因：影响上层批量查询体验。

### 7) 非法参数 fail-fast
- 方案：非法 page size、非法时间范围、非法 cursor 立即返回错误。
- 原因：快速暴露调用问题，减少模糊结果。
- 备选：静默矫正。拒绝原因：会掩盖上游 bug。

### 8) feature flag 不新增，默认可用
- 方案：作为 diagnostics API 正常能力直接可用。
- 原因：行为是新增查询入口，不改变执行语义。
- 备选：加开关。拒绝原因：增加配置复杂度收益低。

## Risks / Trade-offs

- [Risk] 查询接口首版范围过大  
  → Mitigation: 首版只覆盖关键过滤维度和基础分页排序，不做全文检索。 

- [Risk] 游标实现错误导致重复/漏数据  
  → Mitigation: 增加游标稳定性和边界回归测试。 

- [Risk] 高并发写入下查询一致性预期不清  
  → Mitigation: 文档声明“近实时快照”语义，并在 contract tests 固定行为边界。 

- [Risk] 新 API 与 Recent* 语义混淆  
  → Mitigation: 文档明确适用场景，并保留兼容路径。

## Migration Plan

1. 在 diagnostics store 增加统一查询模型与过滤器。  
2. 在 runtime manager 暴露统一查询 API。  
3. 增加 cursor + pagination + sorting 实现与校验。  
4. 补齐 contract tests（过滤组合、cursor、空集/错误语义、replay-idempotent）。  
5. 更新 docs/index/roadmap 并纳入 shared gate。  

回滚策略：
- 保留 `Recent*` API 作为兼容路径；
- 下线统一查询 API 不影响既有执行链路与诊断写入。

## Open Questions

- 关键参数已按推荐值冻结，暂无阻塞性开放问题。
