# Development Roadmap

更新时间：2026-03-19

## 定位

Baymax 的主线定位是 `library-first + contract-first`：
- 优先交付可嵌入 Go runtime 能力，而不是平台化控制面。
- 所有行为变化以 OpenSpec + 契约测试驱动。
- 文档、代码、测试必须同一节奏更新。

## 现状快照（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更实时口径：`openspec list --json`
- 已归档口径：`openspec/changes/archive/INDEX.md`

截至 2026-03-19：
- 已归档并稳定：R1-R3 主线能力、A4-A16。
- 进行中：
  - `harden-long-running-recovery-boundary-and-timeout-reentry-a17`
  - `introduce-unified-run-team-workflow-task-query-api-a18`
- 待规划/探索：A19+。

## 当前代码能力（简版）

已稳定主干：
- Runner Run/Stream 统一语义（并发与背压基线）。
- 多 Provider（OpenAI/Anthropic/Gemini）与能力探测/降级。
- Context Assembler CA1-CA4。
- 安全治理 S1-S4。
- 多代理基础编排：`teams` / `workflow` / `a2a` / `scheduler` / `composer`。

已收口能力（已归档）：
- A12 异步回报：`a2a.async_*` reason taxonomy、回报重试与幂等去重。
- A13 延后调度：`scheduler.delayed_*` reason taxonomy、`not_before` 语义。
- A14 尾项治理：shared gate + cross-mode 矩阵 + docs/index 收敛。
- A15 workflow 图可组合：subgraph/template、compile-before-plan、canonical ID 与 fail-fast 校验。

## 近期变更进度

### A14（已归档）

目标：完成 A12/A13 契约冻结、门禁矩阵与文档收敛。  
口径：`close-a12-a13-tail-contract-and-compatibility-governance-a14`（archived）。

### A15（已归档）

目标：在不改写执行状态机前提下增强 workflow 图编译复用能力。  
口径：`enhance-workflow-graph-composability-a15`（archived）。

### A16（已归档）

目标：补齐统一协作原语（handoff/delegation/aggregation）并收敛语义与门禁。  
口径：`introduce-multi-agent-collaboration-primitives-a16`（archived）。
结果：`orchestration/collab` 原语包、teams/workflow/composer 接入、`composer.collab.*` 配置、`collab_*` 诊断字段、A16 integration 套件与 shared gate 已收口。

### A17（进行中）

目标：收敛长任务恢复边界（resume/in-flight/timeout reentry）并强化恢复一致性。  
口径：`harden-long-running-recovery-boundary-and-timeout-reentry-a17`（active）。
范围：`recovery.resume_boundary/inflight_policy/timeout_reentry_*` 配置、composer/scheduler 恢复边界判定、A17 合同矩阵与 shared gate 收敛。

### A18（进行中）

目标：补齐按 `run/team/workflow/task` 的统一诊断检索 API 与分页/排序/游标/校验契约。  
口径：`introduce-unified-run-team-workflow-task-query-api-a18`（active）。
范围：`runtime/diagnostics.QueryRuns` 统一入口、`page_size=50`/`<=200`、`time desc` 默认排序、opaque cursor、`task_id` 无匹配空集语义、shared gate 阻断校验。

## 主线进度（摘要）

通信与尾项治理：
1. A11：同步执行并等待结果（已实施）。
2. A12：异步执行后汇报（已归档）。
3. A13：定时延后执行（已归档）。
4. A14：A12/A13 尾项收口治理（已归档）。

编排与恢复增强：
1. A15：workflow 图可组合能力（已归档）。
2. A16：协作原语统一契约（已归档）。
3. A17：长任务恢复边界与重入策略（进行中）。

诊断检索增强：
1. A18：统一 run/team/workflow/task 查询契约（进行中）。

## Lib-First Multi-Agent 差距收敛清单（2026-03-19）

P0（必须优先）：
- [x] A9 恢复能力主干落地并归档。
- [x] A10 调度治理落地并验收。
- [x] A14 尾项收口（shared gate + matrix + docs/index）完成并归档。
- [x] reason taxonomy 与 run summary 兼容语义统一（`additive + nullable + default`）。

P1（紧随其后）：
- [x] workflow 图能力增强（A15）。
- [x] multi-agent 协作原语增强（A16 已归档）。
- [ ] 长任务恢复边界收敛（A17 进行中）。
- [ ] 按 run/team/workflow/task 的统一检索 API（A18 进行中，库接口优先）。
- [ ] 多代理主链路性能基线纳入 CI（吞吐/延迟/重试放大/recovery 时间）。

P2（可选，DX/生态）：
- [ ] 最小 replay CLI 工具链。
- [ ] 全链路示例（team + workflow + a2a + scheduler + recovery）。
- [ ] 外部适配样板与迁移映射文档。

推荐顺序：
1. 完成并归档 A17（恢复边界）。
2. 完成并归档 A18（统一检索 API）。
3. 推进多代理性能基线门禁。
4. 补齐 P2 DX/生态能力。

## 非近期范围（为保持简洁，明确延后）

以下能力不进入当前主线：
- 多租户控制面、RBAC、审计平台化。
- Web 控制台与运营平台。
- 全局调度控制面与跨租户负载均衡。

说明：相关方向保留在长期 `R4 platformization` 议题，不影响当前 lib-first 路线。

## 执行规则

- 优先单变更推进；若并行推进多个 change，必须显式声明依赖边界并保持每个 PR 聚焦单一 change。
- 提案到实现必须遵循顺序：`proposal/design/spec/tasks -> code -> tests -> docs`。
- 合并前最少验证：
  - `go test ./...`
  - `go test -race ./...`
  - `pwsh -File scripts/check-docs-consistency.ps1`
  - `pwsh -File scripts/check-multi-agent-shared-contract.ps1`
