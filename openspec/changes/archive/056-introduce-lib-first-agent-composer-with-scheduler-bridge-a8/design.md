## Context

当前仓库已经具备 `core/runner`、`orchestration/workflow`、`orchestration/teams`、`a2a` 与 `orchestration/scheduler` 的独立能力，并在 A7 收口了多代理 shared-contract gate、reason taxonomy 与兼容窗口语义。  
但业务接入仍需手工组装这些模块，导致以下问题：
- 配置快照消费点分散，`teams.* / workflow.* / a2a.* / scheduler.* / subagent.*` 的生效边界难以统一。
- 组合链路缺少稳定入口，Run/Stream 等价、接管回放、字段注入依赖调用方自约束。
- 示例与文档无法给出单一“推荐接入路径”。

本设计引入 `orchestration/composer`（新包）作为 `library-first` 入口，做组合编排与调度桥接，不引入平台化控制面（租户/RBAC/审计门户）。

## Goals / Non-Goals

**Goals:**
- 提供统一组合入口，封装 `runner + workflow + teams + a2a + scheduler` 的接缝。
- 支持 scheduler-managed 子任务双路径：`local child-run` 与 `a2a child-run`。
- 固化 scheduler 初始化失败降级策略：优先配置 backend，失败降级到 `memory`，并发射诊断/时间线信号。
- 保持 `RuntimeRecorder` 单写入口，组合层仅通过标准事件注入 `run.finished` additive 摘要字段。
- 固化 Run/Stream 语义一致目标：终态类别、关键聚合字段、失败类别保持等价。
- 提供可回归的契约测试与门禁路径，并与文档保持同步。

**Non-Goals:**
- 不引入 control plane（多租户、RBAC、审计流水线、全局调度控制台）。
- 不修改 `core/runner` 主循环状态机为分布式编排器。
- 不新增 provider 协议能力或修改 A2A delivery/version 协商规则。
- 不改变 A7 兼容窗口原则（继续 additive + nullable + default）。

## Decisions

### 1) 新增 `orchestration/composer` 作为组合层，而不是把能力塞进 `core/runner`
- 方案：新增独立 `composer` 包，暴露组合执行 API，并复用既有 runner/workflow/teams/scheduler。
- 原因：保持 runner 内核职责稳定，避免跨模块耦合进入 loop 状态机。
- 备选：在 runner 中增加大量组合 option。拒绝原因：接口膨胀、语义边界变模糊。

### 2) scheduler 后端失败采用“可控降级到 memory”
- 方案：当配置 backend 初始化失败（如 `file` 路径不可用）时，自动切换到 `memory`，并记录 `scheduler_backend_fallback=true` 与原因码。
- 原因：满足你确认的降级策略，优先保证服务可运行，同时保留可观测信号。
- 备选：初始化 fail-fast。拒绝原因：会把可恢复配置问题升级为全链路不可用。

### 3) 子任务执行桥接同时覆盖 local 与 a2a
- 方案：composer 内定义统一 child task envelope，按 target 路由到 local runner 或 a2a adapter，并统一回写 scheduler terminal commit。
- 原因：你已确认双路径都要支持，且统一 envelope 可复用 correlation 与幂等键语义。
- 备选：A8 仅支持 local。拒绝原因：会推迟远端协作落地，导致入口再次分叉。

### 4) 热更新采用 `next_attempt_only` 生效模型
- 方案：新快照仅影响新 `enqueue/spawn/claim`；in-flight attempt 的 lease 语义与超时预算保持创建时快照。
- 原因：避免热更新影响进行中的 lease 判断，降低“运行中配置突变”带来的恢复不确定性。
- 备选：全量实时切换。拒绝原因：可能导致旧 attempt 被错误回收或心跳误判。

### 5) guardrail 执行点固定为 spawn 前 fail-fast
- 方案：在 spawn 前执行 `max_depth/max_active_children/child_timeout_budget` 硬校验，拒绝即发 `subagent.budget_reject`，不入队。
- 原因：最小化无效任务扩散，保持上界可解释与回放稳定。
- 备选：入队后异步拒绝。拒绝原因：会污染调度统计并增加清理复杂度。

### 6) 质量门禁并入现有 shared-contract gate
- 方案：继续使用 `check-multi-agent-shared-contract.*` 作为阻断入口，追加 composer contract suite，不新增平行 gate。
- 原因：减少门禁分裂，保持 CI 阻断信号单一、可维护。
- 备选：新增 `check-composer-contract.*`。拒绝原因：门禁碎片化、维护成本提升。

## Risks / Trade-offs

- [Risk] 自动降级到 memory 可能掩盖持久化后端故障  
  → Mitigation: 增加显式 fallback reason 与计数指标，并在文档标明“降级不等于恢复持久化”。

- [Risk] 组合层引入后 API 认知成本上升  
  → Mitigation: 保持最小 API 面，示例与 README 只推荐一条主路径。

- [Risk] 双路径 child-run 可能引入语义分叉  
  → Mitigation: 用统一 envelope + terminal commit 合流，并用 Run/Stream 等价契约回归。

- [Risk] 热更新 `next_attempt_only` 被误解为“即时全局生效”  
  → Mitigation: 在 runtime-config 文档中明确生效边界，并补充回放场景测试。

## Migration Plan

1. 新增 `orchestration/composer` 包与最小 API，先打通 local child-run 桥接，再接入 a2a child-run。  
2. 接入 `runtime/config.Manager` 快照读取，落地 `next_attempt_only` 生效策略。  
3. 实现 scheduler backend fallback-to-memory 逻辑与统一诊断字段。  
4. 将组合层 summary 注入接入既有 `run.finished` 事件，保持 `RuntimeRecorder` 单写。  
5. 补齐集成契约测试（Run/Stream 等价、takeover、idempotency、fallback、reload 边界）。  
6. 更新脚本门禁、主干索引、README 与模块边界文档。  

回滚策略：
- 关闭 composer 主入口并回退到原有模块级接入方式；
- 保留新增字段 additive，不破坏旧消费者；
- fallback 逻辑可通过配置开关关闭，恢复严格 backend 初始化策略。

## Open Questions

- `orchestration/composer` 的最小公开 API 是否需要同时提供“全托管”与“分阶段 builder”两种风格。
- scheduler fallback 是否需要暴露 `strict_backend=true` 以供强一致环境禁用降级。
- 示例升级优先级：先改 `examples/07` 还是先改 `examples/08` 作为 composer 首推样例。
