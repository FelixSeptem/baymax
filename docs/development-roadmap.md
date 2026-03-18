# Development Roadmap

更新时间：2026-03-18

## 定位

Baymax 的主线定位是 `library-first + contract-first`：
- 优先交付可嵌入 Go runtime 能力，而不是平台化控制面。
- 所有行为变化以 OpenSpec + 契约测试驱动。
- 文档、代码、测试必须同一节奏更新。

## 现状快照（以代码与 OpenSpec 为准）

状态口径：
- 活跃变更实时口径：`openspec list --json`
- 已归档口径：`openspec/changes/archive/INDEX.md`

截至 2026-03-18：
- 已归档并稳定：R1-R3 主线能力、A4-A8。
- 已完成（未归档）：`harden-composed-session-recovery-and-deterministic-replay-a9`（23/23）。
- 进行中：`introduce-scheduler-qos-fairness-and-deadletter-governance-a10`（0/25）。

## 当前代码能力（简版）

已稳定主干：
- Runner Run/Stream 统一语义（含并发与背压基线）。
- 多 Provider（OpenAI/Anthropic/Gemini）与能力探测/降级。
- Context Assembler CA1-CA4。
- 安全治理 S1-S4。
- 多代理基础编排：`teams` / `workflow` / `a2a` / `scheduler` / `composer`。

已落代码但默认关闭：
- Composer 恢复域 `recovery.*`（A9）：`memory|file` 后端、冲突策略 `fail_fast`、恢复摘要字段。

进行中能力：
- A10 调度治理：QoS/公平性/DLQ/退避策略。

## 活跃变更

### A9（已完成，待归档）

目标：收敛跨会话恢复与确定性重放语义。

当前状态：
- OpenSpec 任务已完成。
- 需要在归档前维持文档、契约测试、索引口径一致。

收口检查项：
- `docs/runtime-config-diagnostics.md` 与 `docs/v1-acceptance.md` 字段与语义一致。
- `docs/mainline-contract-test-index.md` 与实际测试用例一致。
- `scripts/check-multi-agent-shared-contract.*` 保持阻断有效。

### A10（进行中）

目标：补齐 scheduler 治理能力，不改变 lib-first 定位。

已确认设计约束：
- 默认调度：`fifo`。
- 优先级来源：task 字段。
- 公平窗口：`3`。
- DLQ 默认：关闭。
- 退避策略：指数退避 + 抖动。

完成定义（DoD）：
- 配置、实现、诊断、timeline reason、契约测试、文档六项同步完成。
- Run/Stream 等价语义不回退。

## Lib-First Multi-Agent 差距收敛清单（2026-03-18）

P0（必须优先）：
- [x] A9 恢复能力主干落地。
- [ ] A9 文档与契约收口后归档。
- [ ] A10 调度治理落地并验收。
- [ ] recovery + qos 端到端合同测试矩阵补齐。
- [ ] 新增 reason taxonomy 与 run summary 解析语义统一（`additive + nullable + default`）。

P1（紧随其后）：
- [ ] workflow 图能力增强（subgraph/复用节点/条件模板）。
- [ ] multi-agent 协作原语增强（handoff/delegation/aggregation）。
- [ ] 长任务恢复边界收敛（resume 边界、in-flight 不回溯、超时重入策略）。
- [ ] 按 run/team/workflow/task 的统一检索 API（库接口优先）。
- [ ] 多代理主链路性能基线纳入 CI（吞吐/延迟/重试放大/recovery 时间）。

P2（可选，DX/生态）：
- [ ] 最小 replay CLI 工具链。
- [ ] 全链路示例（team + workflow + a2a + scheduler + recovery）。
- [ ] 外部适配样板与迁移映射文档。

推荐顺序：
1. A9 收口与归档。
2. A10 完成并归档。
3. P1 编排能力与协作原语。
4. P1 观测与性能回归门禁。
5. P2 DX/生态补充。

## 非近期范围（为保持简洁，明确延后）

以下能力不进入当前主线：
- 多租户控制面、RBAC、审计平台化。
- Web 控制台与运营平台。
- 全局调度控制面与跨租户负载均衡。

说明：相关方向保留在长期 `R4 platformization` 议题，不影响当前 lib-first 路线。

## 执行规则

- 每次只推进一个 OpenSpec change，避免并行大改导致口径漂移。
- 提案到实现必须遵循顺序：`proposal/design/spec/tasks -> code -> tests -> docs`。
- 合并前最少验证：
  - `go test ./...`
  - `go test -race ./...`
  - `pwsh -File scripts/check-docs-consistency.ps1`
  - `pwsh -File scripts/check-multi-agent-shared-contract.ps1`
