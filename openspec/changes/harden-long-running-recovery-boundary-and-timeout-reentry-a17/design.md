## Context

A9 已建立基础恢复能力，A14 完成 tail 治理收口，A16 正在推进统一协作原语。随着 orchestration 路径复杂度上升，恢复语义的关键风险从“是否可恢复”转向“恢复边界是否一致”：
- 恢复后哪些 in-flight 状态可继续、哪些必须重建；
- timeout 场景是否允许重入、重入上限如何定义；
- resume 是否会导致已终态任务回溯执行。

A17 聚焦边界治理，不新增平台能力，不替换现有恢复存储模型。

## Goals / Non-Goals

**Goals:**
- 定义统一恢复边界策略：`next_attempt_only` + `no_rewind`。
- 定义 timeout 重入策略：`single_reentry_then_fail`，并固定每 task 最大重入 `1`。
- 保持恢复冲突策略继续 `fail_fast`。
- 让 composer/scheduler/workflow 在恢复边界下保持 Run/Stream 语义等价。
- 通过 timeline/diagnostics/contract gate 固化边界可观测与可回归性。

**Non-Goals:**
- 不引入平台化任务编排或外部协调组件。
- 不修改恢复后端类型（仍为 memory/file）。
- 不新增顶级 reason namespace。
- 不做业务侧调度策略扩展（qos/backoff 算法本身不变）。

## Decisions

### 1) 恢复边界模式固定为 `next_attempt_only`
- 方案：恢复后的策略更新仅作用于“下一尝试”，不 retroactive 改写当前已确定尝试。
- 原因：与现有 next-attempt-only 配置语义保持一致，降低不确定性。
- 备选：恢复时对所有 in-flight 尝试立即重算。拒绝原因：会破坏确定性与可审计性。

### 2) in-flight 策略固定为 `no_rewind`
- 方案：已终态任务绝不回溯；in-flight 任务只允许按边界策略推进到下一尝试或终态。
- 原因：避免重复执行导致副作用膨胀。
- 备选：允许按快照重跑 in-flight。拒绝原因：易造成重复提交与语义歧义。

### 3) timeout 重入采用 `single_reentry_then_fail`
- 方案：每 task 允许 1 次重入，超过上限直接失败归并。
- 原因：在可靠性与复杂度之间取平衡，阻止无限重入。
- 备选：多次重入或无限重入。拒绝原因：放大资源占用并增加尾部风险。

### 4) 边界策略启用条件跟随 `recovery.enabled`
- 方案：`recovery.enabled=true` 时边界策略自动启用，不单独新增全局主开关。
- 原因：减少配置歧义。
- 备选：独立 enable flag。拒绝原因：增加配置组合复杂度。

### 5) 冲突策略维持 `fail_fast`
- 方案：边界冲突并入既有 fail-fast 恢复策略与错误分类。
- 原因：保持 A9 契约连续性与运维认知一致。
- 备选：best-effort merge。拒绝原因：容易产生隐性数据偏差。

### 6) timeline reason 继续复用 `recovery.*` 与 `scheduler.*`
- 方案：新增边界事件 reason 只使用既有 namespace。
- 原因：保证 shared gate taxonomy 一致性。
- 备选：新增 `recovery_boundary.*` 顶级 namespace。拒绝原因：破坏既有 reason 体系简洁性。

### 7) 兼容策略保持 `additive + nullable + default`
- 方案：新增字段全部 additive，旧字段语义不变。
- 原因：保护现有消费者。
- 备选：重构旧字段结构。拒绝原因：兼容成本高。

## Risks / Trade-offs

- [Risk] 单次重入上限过于保守导致部分可恢复任务提前失败  
  → Mitigation: 先以默认值 1 落地，后续通过配置扩展而非放开默认。 

- [Risk] `no_rewind` 可能与部分业务“希望重跑”预期冲突  
  → Mitigation: 文档明确边界语义并提供可观测字段解释判定结果。 

- [Risk] 恢复边界规则增加实现复杂度  
  → Mitigation: 将边界判定逻辑集中在恢复路径，不侵入主执行状态机。 

- [Risk] 组合场景测试成本上升  
  → Mitigation: 用最小关键矩阵覆盖 crash/restart/replay/timeout 高风险组合。

## Migration Plan

1. 增加恢复边界配置与默认值，并在 manager 校验 fail-fast。  
2. 在 composer/scheduler/workflow 恢复流程接入边界判定。  
3. 增加 timeout 重入计数与上限控制。  
4. 扩展 timeline/diagnostics additive 字段。  
5. 增加 contract tests 与 shared gate 检查项。  
6. 同步文档与 contract index。  

回滚策略：
- 关闭 recovery（`recovery.enabled=false`）回到非恢复路径；
- 保留 additive 字段，不影响旧消费者。

## Open Questions

- 当前关键参数已按推荐值冻结，暂无阻塞性开放问题。
