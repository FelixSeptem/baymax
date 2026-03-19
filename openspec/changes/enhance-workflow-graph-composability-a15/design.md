## Context

在 A11-A14 主线上，多代理通信闭环（同步、异步、延后）与 tail 治理正在收敛，但 workflow DSL 仍以“扁平步骤”为主。当前实现已经具备：
- 确定性计划与执行顺序；
- step-level retry/timeout；
- A2A 远程 step；
- checkpoint/resume。

缺口在于图级复用：复杂流程需要重复复制步骤和条件，难以维护，也不利于复用标准协作编排片段。A15 聚焦 lib-first 下的最小增强：只在 DSL 编译层增强，不改写既有执行状态机。

## Goals / Non-Goals

**Goals:**
- 支持 workflow 子图复用（subgraph）与条件模板（condition template）。
- 保持编译与执行确定性，确保 Run/Stream 语义等价。
- 提供严格 fail-fast 校验与可回归 contract tests。
- 通过 feature flag 默认关闭，保证向后兼容。

**Non-Goals:**
- 不引入平台化编排控制面。
- 不引入通用表达式引擎或任意脚本模板系统。
- 不改变现有 step 执行器接口与 terminal 语义。
- 不扩展 payload 模板化（本期仅 condition）。

## Decisions

### 1) 新增编译展开层，执行层保持不变
- 方案：DSL 在 Plan 前先进行 subgraph/template 展开，产出扁平 Definition 交给现有引擎。
- 原因：最大化复用现有验证、执行、checkpoint 逻辑，降低回归风险。
- 备选：执行期动态展开。拒绝原因：会显著增加状态机复杂度与 resume 不确定性。

### 2) 子图递归深度上限固定为 3
- 方案：编译器强制最大深度 `3`。
- 原因：限制复杂度与爆炸式展开风险，保持调试可控。
- 备选：无限深度或仅按节点总量限制。拒绝原因：循环/爆炸路径更难提前识别与定位。

### 3) 展开后 step_id 规则固定 `<subgraph_alias>/<step_id>`
- 方案：每次子图实例化使用 alias 前缀生成稳定 ID。
- 原因：可读、可追踪，且在 replay/resume 中具有确定性。
- 备选：随机 UUID 或纯序号。拒绝原因：影响可读性与跨运行对比。

### 4) 模板作用域仅限 condition
- 方案：`condition_templates` 仅用于 condition 展开，不覆盖 payload。
- 原因：先解决分支表达重复问题，避免模板系统膨胀。
- 备选：全字段模板。拒绝原因：会扩大语义面并引入安全与可维护风险。

### 5) 覆盖策略：允许 retry/timeout，禁止 kind
- 方案：子图实例化允许对 `retry`、`timeout` 做局部覆盖；`kind` 不可覆盖。
- 原因：运行策略可定制，执行语义边界稳定。
- 备选：全部允许覆盖。拒绝原因：容易破坏子图语义一致性。

### 6) feature flag 默认关闭
- 方案：新增 `workflow.graph_composability.enabled`，默认 `false`。
- 原因：兼容优先，减少未升级宿主的行为变化风险。
- 备选：默认开启。拒绝原因：会改变现有 DSL 解析与验证路径。

### 7) 非法输入一律编译期 fail-fast
- 方案：模板缺失、变量缺失、循环引用、ID 冲突、深度越界在编译/校验阶段直接失败。
- 原因：错误尽早暴露，避免执行中隐式降级。
- 备选：best-effort 跳过。拒绝原因：会导致行为不可预测。

## Risks / Trade-offs

- [Risk] 编译展开增加 planner 复杂度  
  → Mitigation: 将展开器与执行器解耦，覆盖独立单测和合同测试。 

- [Risk] alias 规则变更可能影响现有消费者匹配逻辑  
  → Mitigation: 新规则仅在 feature flag 启用且使用子图时生效，并通过文档说明。 

- [Risk] 新增诊断字段导致消费者解析偏差  
  → Mitigation: 严格使用 `additive + nullable + default` 并补 parser 合同测试。 

- [Risk] 子图展开后步骤量上升带来性能压力  
  → Mitigation: 深度上限 3 + 基线 benchmark smoke，避免无界增长。

## Migration Plan

1. 增加 DSL 结构与编译展开器（subgraph/template）。  
2. 增加编译校验（深度、循环、变量、ID 冲突、覆盖规则）。  
3. 将展开结果接入现有 plan/execute/checkpoint 流程。  
4. 增加 feature flag 与 runtime 诊断字段。  
5. 补充 integration + contract + shared gate。  
6. 同步更新 README/roadmap/runtime-config docs 与 mainline index。  

回滚策略：
- 关闭 `workflow.graph_composability.enabled` 即回退至扁平 DSL 语义；
- additive 字段保留，不影响旧消费者。

## Open Questions

- 当前关键参数已冻结（深度=3、作用域仅 condition、覆盖策略固定、默认关闭、fail-fast），暂无阻塞问题。
