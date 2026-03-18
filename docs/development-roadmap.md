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
- 已归档并稳定：R1-R3 主线能力、A4-A11。
- 已完成（未归档）：`introduce-async-agent-reporting-contract-a12`（23/23）。
- 进行中：`introduce-delayed-dispatch-not-before-contract-a13`（0/23）。

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
- A13 定时延后执行：`not_before` 延后领取语义（进行中）。

已收口能力（待归档）：
- A12 异步执行后汇报：`SubmitAsync + ReportSink`、幂等去重、回报重试与回放一致性。

## 近期变更进度

### A10（已完成，待归档）

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
- `check-multi-agent-shared-contract` 与 `contributioncheck` 通过。

### A12（已完成，待归档）

目标：补齐异步执行后独立回报能力，形成非阻塞协作闭环。

当前口径：
- OpenSpec change：`introduce-async-agent-reporting-contract-a12`（active）。
- 完成进度：`23/23`。
- 覆盖范围：`SubmitAsync + ReportSink`、回报重试与幂等去重、Run/Stream 等价与 recovery 回放一致性。

### A13（进行中）

目标：补齐业务级定时延后执行能力，完成通信能力三段式闭环。

当前口径：
- OpenSpec change：`introduce-delayed-dispatch-not-before-contract-a13`（active）。
- 范围聚焦：`scheduler.task.not_before`、claim 可领取判定、恢复后时间语义一致性。

### 通信能力主线进度（A11-A13）

1. A11：同步执行并等待结果（已实施）。
2. A12：异步执行后汇报（已实施，待归档）。
3. A13：定时延后执行（进行中）。

## Lib-First Multi-Agent 差距收敛清单（2026-03-18）

P0（必须优先）：
- [x] A9 恢复能力主干落地。
- [x] A9 文档与契约收口后归档。
- [x] A10 调度治理落地并验收。
- [ ] recovery + qos 端到端合同测试矩阵补齐。
- [x] 新增 reason taxonomy 与 run summary 解析语义统一（`additive + nullable + default`）。

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
1. A12 归档。
2. A13 收口并归档。
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
