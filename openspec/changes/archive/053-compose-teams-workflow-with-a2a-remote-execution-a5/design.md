## Context

仓库已具备三类独立能力：
- `orchestration/teams`：单进程团队协作（serial/parallel/vote）
- `orchestration/workflow`：确定性 DSL 编排与 checkpoint/resume
- `a2a`：跨 agent 最小互联（submit/status/result）

当前缺口是“跨域组合契约”尚未标准化：workflow 无法直接声明 A2A 远端步骤，teams 无法在统一语义下混编 local 与 remote worker，导致业务侧需要自行拼接并承担可观测与回归成本。

同时，A2A A4 正在加固 delivery/version 协商，本提案应保持与其解耦：A5 只关注编排层如何消费稳定的 A2A 能力，不重新定义传输细节。

## Goals / Non-Goals

**Goals:**
- 在 workflow 中引入可组合的 A2A step 语义并保持确定性执行。
- 在 teams 中引入 local + remote(worker via A2A) 混编语义，并收敛失败/取消策略。
- 统一跨域关联字段、timeline reason 与 run diagnostics 聚合约束。
- 增量补齐组合契约测试，覆盖 Run/Stream 等价与 replay 幂等。
- 保持 A2A/MCP 边界清晰，不引入职责重叠。

**Non-Goals:**
- 不引入 control plane、租户隔离、RBAC、跨地域调度。
- 不改写 MCP 传输语义或把 peer 协商塞入 MCP。
- 不在本提案中实现 A2A delivery/version 的具体协商算法（由 A4 负责）。

## Decisions

### 1) Workflow 通过扩展 step kind 接入 A2A，而不是新增旁路编排引擎
- 方案：扩展 workflow step kind 支持 `a2a`，复用现有 step adapter 模式。
- 原因：保持 DSL 与执行模型单一入口，便于确定性与回放验证。
- 备选：新增 workflow-remote 子引擎。拒绝原因：状态机重复，观测与回归成本翻倍。

### 2) Teams 采用统一 Task 抽象承载 local/remote 执行
- 方案：在 task 定义中增加执行目标（local 或 remote via A2A），对外仍保持统一 task lifecycle。
- 原因：避免为 remote 再建独立生命周期，减少语义分叉。
- 备选：remote task 使用单独状态机。拒绝原因：会破坏现有 team 汇总字段与等价测试基线。

### 3) 组合链路的可观测遵循 additive + single-writer
- 方案：新增字段只做 additive，所有写入仍走 `observability/event.RuntimeRecorder`。
- 原因：沿用当前 idempotency 与 replay 语义，避免并行事实源。
- 备选：orchestration/a2a 直接写 diagnostics store。拒绝原因：违反边界约束并放大并发风险。

### 4) 边界治理采用“契约门禁 + 命名规范”双重约束
- 方案：在 shared contract gate 中新增组合场景检查，强制 `team.*|workflow.*|a2a.*` reason 前缀与 `peer_id` 命名。
- 原因：跨域能力增长时，最容易先漂移的是命名与状态映射。
- 备选：仅靠代码评审人工约束。拒绝原因：无法稳定阻断回归。

### 5) 与 A4 的协作边界
- 方案：A5 只依赖 A2A 暴露的稳定策略结果（delivery mode/version result），不直接实现或覆盖协商策略。
- 原因：降低并行开发冲突，确保提案可独立落地与回滚。
- 备选：A5 同时改 delivery/version 内核。拒绝原因：与在途 A4 高重叠，风险不可控。

## Risks / Trade-offs

- [Risk] Workflow/Teams 引入远端步骤后，Run/Stream 终态可能出现细微偏差  
  → Mitigation: 增加组合契约测试，按“终态 + 关键聚合字段 + reason 分类”三层校验。

- [Risk] 组合链路提升复杂度，诊断字段过多导致可读性下降  
  → Mitigation: 严格 additive 最小字段集，避免引入冗余同义字段。

- [Risk] 与 A4 并行实施产生接口漂移或冲突  
  → Mitigation: 在 A5 中只消费稳定 DTO/字段，配置键命名遵循 `a2a.*` 现有域并在 tasks 中设置联调检查点。

- [Risk] 过早扩展导致边界侵蚀（A2A/MCP 职责混杂）  
  → Mitigation: 将边界检查写入 spec 与 gate，未通过即阻断合入。

## Migration Plan

1. 先扩展 spec 与 docs 契约，再改实现，确保行为变更有明确口径。  
2. 引入 workflow `a2a` step 与 teams remote worker 执行路径（默认关闭或兼容默认值）。  
3. 对齐 runtime config、timeline、diagnostics 字段并完成回放幂等验证。  
4. 补齐组合契约测试矩阵并接入主干索引。  
5. 与 A4 进行接口联调，确认 delivery/version 结果字段消费一致。  

回滚策略：
- 关闭 workflow `a2a` step 能力开关并回退为现有 step kind；
- teams 回退到 local-only 执行路径；
- 保留新增字段为可空 additive，不破坏现有消费者解析。

## Open Questions

- workflow `a2a` step 的默认 timeout/retry 是否与 `a2a.client_timeout` 共用，还是保留 workflow 侧独立覆盖？
- teams mixed 执行中，remote worker 的失败是否默认计入 `team_task_failed`，还是引入细分计数字段？
- 组合门禁脚本是否应单独新增 `check-composed-orchestration-contract.*`，还是并入现有 shared-contract gate？
