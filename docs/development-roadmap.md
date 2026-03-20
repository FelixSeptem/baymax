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
- 已归档并稳定：A4-A28（含 A19 性能门禁、A20 全链路示例、A21 外部适配模板与迁移映射、A22 外部适配 conformance harness、A23 脚手架与 drift gate、A24 pre-1 轨道治理收口、A25 状态口径与模块 README 门禁、A26 manifest + runtime compatibility 契约、A27 capability negotiation + fallback 契约、A28 contract profile versioning + replay gate）。
- 进行中：
  - `introduce-task-board-query-contract-a29`

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

### P0：收口 A28 并修复治理缺口（当前阶段）

完成条件：
- A28 收敛：引入 `contract_profile_version` 与 runtime 支持窗口校验（默认 `current + previous`），不兼容 fail-fast。
- 新增 `check-adapter-contract-replay.*` 并接入 `check-quality-gate.*`，shell/PowerShell 阻断一致。
- A22 conformance 与 A23 scaffold 覆盖 manifest + capability + profile/replay 组合路径。
- 状态口径治理收口：`openspec list --json`、archive index、README、roadmap 四者一致。
- 修复 Windows docs gate 语义：`scripts/check-docs-consistency.ps1` 必须在 `go test` 失败时返回 non-zero（禁止“失败仍打印 passed”）。

A28 实施顺序（收敛变更域）：
1. 在 manifest/negotiation 链路接入 `contract_profile_version` 与 deterministic 错误分类。
2. 补齐 profile-versioned replay fixtures（manifest/negotiation/reason taxonomy）。
3. 新增 replay gate 并接入 quality gate（shell + PowerShell 对齐）。
4. 同步更新主干索引、roadmap/README 状态快照，并修复 docs gate 假阳性。

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

1. 状态口径漂移：A27 已归档，但 roadmap/README 快照仍存在“进行中”残留。
2. A28 核心能力未收口：`contract_profile_version`、replay fixtures、replay gate 仍在实施中。
3. docs gate 稳定性缺口：PowerShell 路径存在失败未阻断风险，需修复为 fail-fast。
4. 传播层缺口：当前示例偏工程验证，缺少 session 化学习路径与多语内容结构（作为 DX 增强项进入后续提案，不影响 lib-first 主线）。

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
