# Runtime Module Boundaries

更新时间：2026-03-18

## 目标

明确全局 runtime 平台能力与 MCP 子域能力边界，避免配置与诊断入口继续耦合在单个 MCP runtime 包。

## 模块职责

- `runtime/config`
  - 统一配置加载（YAML + env + default）
  - 配置校验与 fail-fast 启动
  - 热更新与原子快照切换
  - MCP profile 解析（作为配置字段的一部分）
- `runtime/diagnostics`
  - 统一诊断数据模型与有界存储
  - `call/run/reload/skill` 记录与查询
  - 配置脱敏输出辅助
- `orchestration/composer`
  - `library-first` 组合入口，统一装配 `runner + workflow + teams + a2a + scheduler`
  - 负责组合层接缝与调度桥接，不吸收 provider 协议逻辑或 transport 内部细节
  - 组合路径摘要仅通过标准事件注入（`run.finished` additive 字段），不直接写 diagnostics store
  - recovery 编排（load/validate/reconcile/resume）仅由 composer 收口；恢复快照与重放结果必须经标准事件进入 RuntimeRecorder 单写路径
- `orchestration/invoke`
  - 统一同步远程调用抽象（`submit + wait + normalize`）
  - 负责 A2A 终态收敛、context 优先取消/超时语义、错误分层与 retryable 提示归一
  - 仅提供 orchestration 复用能力，不持有调度状态，也不直接写 diagnostics store
- `orchestration/teams`
  - Teams 协作编排基线（`serial|parallel|vote`）
  - Mixed local/remote worker 执行目标（`target=local|remote`）与统一任务生命周期
  - 角色与任务生命周期语义（`leader|worker|coordinator` + `pending/running/succeeded/failed/skipped/canceled`）
  - 通过标准事件发射 Teams timeline/摘要元数据（含 `team.dispatch_remote/team.collect_remote`，不直接写 diagnostics store）
- `orchestration/workflow`
  - Workflow DSL 基线（`step/depends_on/condition/retry/timeout`）解析与静态 DAG 校验
  - A2A remote step（`kind=a2a`）在现有 adapter 下执行，复用 workflow retry/timeout/checkpoint 语义
  - 确定性调度、重试/超时、checkpoint/resume 执行语义
  - 通过标准事件发射 Workflow timeline/摘要元数据（含 `workflow.dispatch_a2a`，不直接写 diagnostics store）
- `a2a`
  - A2A 最小互联能力（`submit/status/result`）与任务生命周期语义
  - Agent Card 能力发现与确定性路由输入（静态注册 + capability 匹配）
  - A2A delivery 模式协商与降级（`callback|sse`）以及版本协商（`strict_major + min_supported_minor`）仅在 `a2a/*` 实现，不下沉到 MCP 传输层
  - 保留并透传组合编排关联字段（`workflow_id/team_id/step_id/task_id/agent_id/peer_id`）
  - 通过标准事件发射 A2A timeline/摘要元数据（不直接写 diagnostics store）
- `orchestration/scheduler`
  - 分布式 subagent 调度基线（enqueue/claim/heartbeat/lease_expire/requeue/complete/fail）
  - QoS 治理能力（`scheduler.qos.mode` + fairness 窗口 + retry backoff + DLQ）仅在该模块内实现
  - 维护 task/attempt/lease 状态机与 terminal commit 幂等语义（`task_id+attempt_id`）
  - parent-child guardrail（`max_depth|max_active_children|child_timeout_budget`）fail-fast 拒绝
  - 通过标准事件发射 Scheduler/Subagent timeline（`scheduler.*` / `subagent.*`，不直接写 diagnostics store）
- `mcp/profile`
  - MCP profile 常量与策略解析（仅 MCP 语义）
- `mcp/retry`
  - MCP 重试控制（retryable 分类 + backoff）
- `mcp/diag`
  - MCP 调用摘要字段模型与本地有界缓存
- `mcp/internal/reliability`
  - MCP 内部共享重试/超时/backoff 执行骨架（internal-only）
- `mcp/internal/observability`
  - MCP 内部共享事件发射与诊断映射桥接（internal-only）
- `mcp/http` / `mcp/stdio`
  - 传输实现
  - 消费 `runtime/config.Manager` 配置与诊断 API
- `core/runner` / `tool/local` / `skill/loader`
  - 消费全局 runtime 配置快照
  - 产出标准运行时事件（不直接写诊断存储）
- `observability/event`
  - 事件日志与分发
  - `RuntimeRecorder` 作为诊断唯一写入入口，将事件映射为统一诊断记录

## 依赖方向

允许方向（简化）：

`runtime/*` -> (no dependency on `mcp/http` or `mcp/stdio`)

`mcp/*`, `core/*`, `tool/*`, `skill/*`, `observability/*`, `orchestration/*` -> `runtime/*`

禁止方向：

- `runtime/config` 或 `runtime/diagnostics` 反向依赖 `mcp/http` / `mcp/stdio`
- 非 `mcp/*` 包依赖 `mcp/internal/*`
- Teams 编排直接写 `runtime/diagnostics` 存储（必须经 `observability/event.RuntimeRecorder` 单写入口）
- Workflow 编排直接写 `runtime/diagnostics` 存储（必须经 `observability/event.RuntimeRecorder` 单写入口）
- A2A 模块直接写 `runtime/diagnostics` 存储（必须经 `observability/event.RuntimeRecorder` 单写入口）
- Scheduler 模块直接写 `runtime/diagnostics` 存储（必须经 `observability/event.RuntimeRecorder` 单写入口）
- Composer 模块直接写 `runtime/diagnostics` 存储（必须经 `observability/event.RuntimeRecorder` 单写入口）
- 将 peer 协作语义下沉到 `mcp/*`（A2A/MCP 职责重叠）

CI 通过 `scripts/check-runtime-boundaries.sh` 做静态检查。
治理型评审可结合 `docs/modular-e2e-review-matrix.md` 执行“模块 + 主干链路”双视角核验。

R4 多代理共享契约前置门禁（阻断级）：
- Linux/macOS: `bash scripts/check-multi-agent-shared-contract.sh`
- Windows: `pwsh -File scripts/check-multi-agent-shared-contract.ps1`
- required-check 候选: `multi-agent-shared-contract-gate`
- Scheduler/Subagent 收口要求：reason 必须为 `scheduler.*|subagent.*`，且 scheduler 管理路径需携带 `task_id` / `attempt_id` 关联字段。
- Scheduler QoS 收口要求：`scheduler.qos_claim|scheduler.fairness_yield|scheduler.retry_backoff|scheduler.dead_letter` 必须经 timeline 事件进入 RuntimeRecorder 单写路径。
- Composer 收口要求：`orchestration/composer` 仅做 orchestration glue；scheduler fallback 与子任务摘要信号必须以事件方式进入 RuntimeRecorder 单写路径。
- Recovery 收口要求：`orchestration/composer` 负责恢复状态机与冲突终止语义（`fail_fast`）；恢复 reason（`recovery.restore|recovery.replay|recovery.conflict`）与 run 摘要字段必须走事件单写入口。
- 明确禁止：`orchestration/scheduler` 直接写 `runtime/diagnostics`（必须经 `observability/event.RuntimeRecorder` 单写入口）。

## Owner 建议

- `runtime/config`：平台基础设施 owner
- `runtime/diagnostics`：可观测性 owner
- `mcp/profile`、`mcp/retry`、`mcp/diag`：MCP owner
- `skill/loader`：Skill owner

## 扩展约束

- 新增全局配置字段时，必须同步：
  - `runtime/config` schema + validation
  - `docs/runtime-config-diagnostics.md` 字段索引
- 新增诊断记录类型时，必须同步：
  - `runtime/diagnostics` record 定义
  - 文档中的字段与语义说明

## 全局限制（职责分工重点）

- Context Assembler 与 Model Provider 的职责必须分离：
  - `context/assembler` 只做策略编排与触发时机控制（例如 CA3 压力分区、阈值判定、计数调用节流）。
  - `model/*` 负责 provider 协议细节与官方 SDK 调用（包括 token count、能力探测、流式映射）。
- 禁止在 `context/*` 中直接引入 provider 官方 SDK（OpenAI/Anthropic/Gemini），避免跨层耦合与升级扩散。
- 任何新增 provider 级能力（例如 token count、模型元数据查询）应先落在 `model/<provider>`，再由上层通过接口复用。
