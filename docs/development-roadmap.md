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
- 已归档并稳定：R1-R3 主线能力、A4-A14。
- 进行中：`enhance-workflow-graph-composability-a15`。
- 待规划/探索：A16+。

## 当前代码能力（简版）

已稳定主干：
- Runner Run/Stream 统一语义（含并发与背压基线）。
- 多 Provider（OpenAI/Anthropic/Gemini）与能力探测/降级。
- Context Assembler CA1-CA4。
- 安全治理 S1-S4。
- 多代理基础编排：`teams` / `workflow` / `a2a` / `scheduler` / `composer`。

已落代码但默认关闭：
- Composer 恢复域 `recovery.*`（A9）：`memory|file` 后端、冲突策略 `fail_fast`、恢复摘要字段。

已收口能力（A12/A13）：
- A12 异步回报：`a2a.async_*` reason taxonomy、回报重试与幂等去重。
- A13 延后调度：`scheduler.delayed_*` reason taxonomy、`not_before` 语义与延后摘要字段。

## 近期变更进度

### A12（已归档）

目标：补齐异步执行后独立回报能力，形成非阻塞协作闭环。

口径：
- OpenSpec change：`introduce-async-agent-reporting-contract-a12`（archived）。
- 覆盖范围：`SubmitAsync + ReportSink`、回报重试与幂等去重、Run/Stream 等价与 recovery 回放一致性。

### A13（已归档）

目标：补齐业务级定时延后执行能力，完成通信能力三段式闭环。

口径：
- OpenSpec change：`introduce-delayed-dispatch-not-before-contract-a13`（archived）。
- 范围聚焦：`scheduler.task.not_before`、claim 双 gate、恢复后时间语义一致性、delayed timeline reason 与 run diagnostics additive 字段。

### A14（已归档）

目标：不新增运行时功能，完成 A12/A13 的契约冻结、门禁矩阵与文档收敛。

口径：
- OpenSpec change：`close-a12-a13-tail-contract-and-compatibility-governance-a14`（archived）。
- 范围聚焦：shared contract reason completeness、`sync/async/delayed × Run/Stream × qos/recovery` 矩阵、A12/A13 兼容窗口 parser 语义、roadmap/index/doc 一致性。

### A15（进行中）

目标：在保持 workflow 执行状态机不变的前提下，引入图编译复用能力（subgraph + condition template）。

口径：
- OpenSpec change：`enhance-workflow-graph-composability-a15`（active）。
- 范围聚焦：compile-before-plan 展开、深度上限 `3`、canonical ID `<alias>/<step_id>`、override 策略（允许 `retry/timeout`、禁止 `kind`）、Run/Stream fail-fast 一致性、diagnostics additive 字段与 gate/doc 收口。

### 通信能力主线进度（A11-A14）

1. A11：同步执行并等待结果（已实施）。
2. A12：异步执行后汇报（已归档）。
3. A13：定时延后执行（已归档）。
4. A14：尾项收口治理（已归档）。
5. A15：workflow 图可组合能力（进行中）。

## Lib-First Multi-Agent 差距收敛清单（2026-03-19）

P0（必须优先）：
- [x] A9 恢复能力主干落地。
- [x] A9 文档与契约收口后归档。
- [x] A10 调度治理落地并验收。
- [x] A14 尾项收口（shared gate + matrix + docs/index）完成并归档。
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
1. 完成 A15（workflow graph composability）并归档。
2. 推进 P1 协作原语与恢复边界增强（A16+）。
3. 推进 P1 观测与性能回归门禁。
4. 补齐 P2 DX/生态能力。

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
