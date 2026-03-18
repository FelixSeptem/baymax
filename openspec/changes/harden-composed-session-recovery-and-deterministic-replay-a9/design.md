## Context

当前多代理能力已经具备：
- 组合编排（Teams/Workflow/A2A）语义与字段收敛；
- 分布式 subagent scheduler（enqueue/claim/heartbeat/requeue/terminal commit）；
- shared-contract gate 与 single-writer diagnostics 基线。

但在跨会话恢复上仍存在空洞：进程崩溃或重启后，composer 级执行上下文无法按统一契约恢复，恢复链路与重放幂等由宿主自行拼装，容易导致重复提交、重复终态、计数膨胀与语义漂移。

A9 目标是在 `library-first` 约束下引入标准化会话恢复，不引入控制平面能力。

## Goals / Non-Goals

**Goals:**
- 定义 composer 级恢复契约与最小恢复单元（run/workflow/scheduler/a2a/replay cursor）。
- 提供 `RecoveryStore` 抽象并支持 `memory|file` 两类后端。
- 默认恢复关闭（显式启用），避免行为突变。
- 冲突处理统一 `fail_fast`（快照与实时状态不一致时快速终止）。
- 将 A2A in-flight 状态纳入恢复路径，保证子任务收敛可追踪。
- 保持 Run/Stream 语义等价与 additive + nullable + default 兼容窗口。
- 把恢复回归套件纳入现有 shared multi-agent gate（阻断级）。

**Non-Goals:**
- 不引入多租户、RBAC、审计控制面。
- 不实现跨地域全局调度或分布式一致性协议升级。
- 不重写 core/runner 状态机为分布式执行引擎。
- 不改变既有 A2A delivery/version 协商机制。

## Decisions

### 1) 恢复能力放在 composer 层统一编排
- 方案：在 composer 层引入恢复编排入口，消费 workflow/scheduler/a2a 既有能力。
- 原因：恢复是跨模块横切能力，放在单模块会导致语义分裂。
- 备选：分别在 workflow/scheduler 增加独立恢复入口。拒绝原因：恢复顺序与冲突决策无法统一。

### 2) RecoveryStore 先支持 `memory|file`
- 方案：定义统一存储接口，内存后端用于测试与临时运行，文件后端用于跨会话恢复。
- 原因：满足你确认的“都做”策略，兼顾可落地性与可测试性。
- 备选：仅 file。拒绝原因：测试复杂度上升，开发反馈回路变慢。

### 3) 恢复最小单元必须包含 A2A in-flight
- 方案：恢复快照中纳入 A2A in-flight 任务状态和映射关系，恢复时与 scheduler/workflow 一并收敛。
- 原因：仅恢复 workflow/scheduler 会丢失远端协作上下文，导致 join/commit 不完整。
- 备选：A2A in-flight 延后。拒绝原因：恢复路径会出现不可观测“黑洞”。

### 4) 默认关闭恢复，显式开启后生效
- 方案：`recovery.enabled=false` 默认值；只有显式配置为 true 才启用恢复逻辑。
- 原因：避免升级后行为变化，保持 v1 兼容与可控发布节奏。
- 备选：默认开启。拒绝原因：对现有用户是隐式行为变更。

### 5) 冲突策略固定为 fail-fast
- 方案：当恢复快照与实时状态（attempt/version/cursor）冲突时直接终止恢复，输出标准错误与事件。
- 原因：你确认选择 fail-fast；且该策略最符合契约稳定性优先。
- 备选：best-effort merge。拒绝原因：会引入不可解释状态合并与重放不确定性。

### 6) 恢复门禁并入现有 shared-contract gate
- 方案：在 `check-multi-agent-shared-contract.*` 中加入恢复回归套件，不新增平行脚本。
- 原因：减少门禁碎片化，保持阻断口径单一。
- 备选：新建独立 recovery gate。拒绝原因：CI 维护成本和信号分散增加。

## Risks / Trade-offs

- [Risk] file 存储损坏导致恢复不可用  
  → Mitigation: 快照校验失败直接 fail-fast，回退到新会话启动并输出原因码。

- [Risk] 恢复范围扩大（纳入 A2A in-flight）提升实现复杂度  
  → Mitigation: 先定义最小状态模型与明确恢复顺序，避免一次性扩大语义面。

- [Risk] 默认关闭导致用户误以为“已有自动恢复”  
  → Mitigation: 在 README/runtime-config 文档明确默认值与启用方式，并在 run summary 输出恢复开关状态字段。

- [Risk] fail-fast 冲突策略会增加恢复失败率  
  → Mitigation: 提供标准冲突原因分类和可复现日志，确保失败可诊断、可重试。

## Migration Plan

1. 定义 recovery 配置域与 `RecoveryStore` 接口；实现 `memory|file` 后端。  
2. 引入恢复快照模型（run/workflow/scheduler/a2a/cursor）与序列化版本字段。  
3. 在 composer 中实现恢复入口与恢复顺序（load -> validate -> reconcile -> resume）。  
4. 增加冲突 fail-fast 检查与标准事件/诊断字段。  
5. 补齐恢复契约测试（重启恢复、重复重放、冲突中止、Run/Stream 等价）。  
6. 将 suite 并入 shared-contract gate，更新文档与主干索引。  

回滚策略：
- 关闭 `recovery.enabled` 即可回到现有非恢复路径；
- 新增字段保持 additive，不影响旧消费者；
- 恢复相关门禁可在回滚阶段单独降级为非阻断（仅限临时紧急回滚场景）。

## Open Questions

- `RecoveryStore` 文件后端是否需要 WAL（A9 是否仅 JSON 快照即可满足稳定性目标）。
- A2A in-flight 恢复时是否需要“最大恢复重试次数”独立配置项。
- 恢复事件 reason 命名是否单独引入 `recovery.*` 命名空间，或复用既有子域 reason。
