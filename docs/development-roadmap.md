# Development Roadmap

更新时间：2026-03-20

## 定位

Baymax 主线保持 `library-first + contract-first`：
- 交付可嵌入 Go runtime，而非平台化控制面。
- 以 OpenSpec + 契约测试驱动行为变更。
- 代码、测试、文档同一 PR 同步收敛。

## 当前状态（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更：`openspec list --json`
- 已归档变更：`openspec/changes/archive/INDEX.md`

截至 2026-03-20：
- 已归档并稳定：A4-A29（含 A19 性能门禁、A20 全链路示例、A21 外部适配模板与迁移映射、A22 外部适配 conformance harness、A23 脚手架与 drift gate、A24 pre-1 轨道治理收口、A25 状态口径与模块 README 门禁、A26 manifest + runtime compatibility 契约、A27 capability negotiation + fallback 契约、A28 contract profile versioning + replay gate、A29 task board query contract）。
- 进行中：
  - `introduce-unified-mailbox-coordination-contract-a30`

## 版本阶段口径（延续 0.x）

当前仓库**不做 `1.0.0` / prod-ready 承诺**，继续沿用 `0.x` 治理口径（见 `docs/versioning-and-compatibility.md`）。
在 `0.x` 阶段，版本号用于表达变更范围，不构成稳定兼容承诺；主线目标是“持续收敛、可回归迭代”。
`0.x` 阶段**允许新增能力型提案**，不采用“仅治理/仅修复”的限制；新增能力需满足准入字段与质量门禁要求。

1. 运行时主干稳定：
- Runner Run/Stream 统一语义与并发背压基线。
- Multi-provider（OpenAI/Anthropic/Gemini）统一 contract。
- Context Assembler CA1-CA4、Security S1-S4 已归档能力。

2. 多代理主链路稳定：
- A11-A18（同步/异步/延后、恢复边界、协作原语、统一诊断查询）语义收口。
- Shared contract gate 与 Run/Stream 等价约束保持阻断。

3. 质量与可回归稳定：
- A19 性能回归门禁（基线 + 相对阈值）。
- A20 全链路示例 smoke 阻断门禁。

4. 外部接入稳定：
- A21 模板与迁移映射（已归档）。
- A22 conformance harness（已归档）。
- A23 scaffold + conformance bootstrap（已归档）。

## 近期收口优先级（0.x）

### P0：推进 A29 Task Board 查询契约（当前阶段）

完成条件：
- 在 `orchestration/scheduler` 落地只读 `QueryTasks` 查询面，覆盖 canonical filters（`task_id/run_id/workflow_id/team_id/state/priority/agent_id/peer_id/parent_run_id/time_range`）。
- 固化分页/排序/游标语义：`page_size=50`（默认）/`<=200`（上限）、默认 `updated_at desc`、排序字段 `updated_at|created_at`、opaque cursor + query boundary 绑定。
- unit + integration 覆盖：过滤 AND 语义、非法参数 fail-fast、cursor 确定性、memory/file parity、snapshot restore 前后语义稳定。
- shared multi-agent gate 纳入 Task Board contract suites，shell/PowerShell 阻断一致。
- README/runtime docs/mainline index 同步 A29 scope 与 non-goals，状态口径保持与 `openspec list --json` 一致。

A29 非目标（当前阶段不做）：
- 任务写操作（cancel/retry/reassign/priority mutate）。
- 平台化任务控制台、RBAC、多租户运维面板。
- 引入外部数据库或全文检索引擎。

### P1：0.x 质量与治理持续收敛

完成条件：
- 所有变更继续通过质量门禁（`check-quality-gate.*`）与契约索引追踪。
- 允许新增能力按“小步提案 + 契约测试 + 文档同步”推进，不引入平台化控制面范围。
- 对外发布继续以 `0.x` 说明风险与兼容预期。

## 下一提案方向（lib-first 优先）

在 A28 完成后，推荐进入“coding-agent 协作能力”的库级收敛，但保持非平台化边界：

1. Task Board（推荐）
- 以 `orchestration/scheduler + runtime/diagnostics` 提供只读任务看板查询接口（过滤、排序、分页、时间窗），不引入控制面 UI。

2. Mailbox Contract（推荐）
- 提供 agent 间消息 envelope（command/event/result）与 `ack/retry/ttl/dlq/idempotency-key` 契约，统一同步等待与异步回报语义。

非目标（当前阶段不做）：
- 平台化任务控制台与多租户运维面板。
- 跨租户统一调度控制平面。

## 当前主要缺失点清单（对齐本轮评审）

1. A29 尚未形成稳定主干能力：scheduler 任务看板查询目前缺少统一 contract 与 gate 覆盖。
2. 跨后端语义仍需持续守护：memory/file 在后续演进中仍有漂移风险，需依赖 contract suite 阻断。
3. 传播层缺口：当前示例偏工程验证，缺少 session 化学习路径与多语内容结构（作为 DX 增强项进入后续提案，不影响 lib-first 主线）。

## 维护提示（状态快照更新）

每次归档或切换活跃 change 后，维护者应同步执行以下最小流程，避免触发 A25 口径漂移阻断：

1. 以 `openspec list --json` 与 `openspec/changes/archive/INDEX.md` 作为唯一状态权威源。
2. 更新 `README.md` 的里程碑快照和 `docs/development-roadmap.md` 的“当前状态”在研列表，确保 active/archived 语义一致。
3. 若状态变更涉及门禁映射，更新 `docs/mainline-contract-test-index.md` 的对应行。
4. 提交前执行 `pwsh -File scripts/check-docs-consistency.ps1`（或 shell 等价脚本）确认无漂移。

## 新增提案准入规则（0.x 阶段）

从本文件生效起，`0.x` 阶段新增提案需满足：

1. 允许新增能力型提案进入近期执行，但必须直接服务于以下至少一类目标：
- 契约一致性（Run/Stream、reason taxonomy、错误分层、兼容语义）。
- 可靠性与安全（fail-fast、回滚、幂等、恢复边界、安全治理）。
- 质量门禁回归治理（contract/perf/docs gate regression）。
- 外部接入 DX（模板、迁移、脚手架、conformance）且可被 gate 验证。
2. 必须保持 lib-first 边界，不引入平台化控制面能力。
3. 必须在提案内说明：`Why now`、风险、回滚点、文档影响、验证命令。
4. 不满足以上条件的需求，统一记录为长期方向，不进入近期执行。

## 长期方向（不进入近期主线）

以下方向明确延后：
- 平台化控制面（多租户、RBAC、审计与运营面板）。
- 跨租户全局调度与控制平面。
- 市场化/托管化 adapter registry 能力。

说明：上述方向在 `0.x` 阶段只登记，不作为当前迭代实施输入。

## 执行与验收规则

- 单变更优先；并行变更需显式依赖边界。
- 严格顺序：`proposal/design/spec/tasks -> code -> tests -> docs`。
- 合并前最少验证：
  - `go test ./...`
  - `go test -race ./...`
  - `pwsh -File scripts/check-docs-consistency.ps1`
  - `pwsh -File scripts/check-quality-gate.ps1`
