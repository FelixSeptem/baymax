# Development Roadmap

更新时间：2026-11-02

Baymax 提案发现与实施入口。按状态区确认当前工作，再按候选表判断是否有证据可启动；详细契约、任务和历史证据见 [OpenSpec changes](../openspec/changes/)、[归档索引](../openspec/changes/archive/INDEX.md) 与 [mainline contract test index](mainline-contract-test-index.md)。

## 版本阶段口径（延续 0.x）

Baymax 处于 pre-`1.0.0` 阶段，不做 `1.0.0` / prod-ready 承诺。允许新增能力型提案；新增提案准入规则（0.x 阶段）要求先确定 owner、`Why now`、最小失败复现、部署范围、风险/回滚、文档影响与验证命令。提案范围限于契约一致性、可靠性与安全、质量门禁回归治理或外部接入 DX。

## 当前状态

已归档 187：`extract-common-execution-graph-boundaries` 提供有界、source-neutral 的拓扑投影，同时保留 workflow、team 和 composer child 各自的执行、恢复与摘要 owner。范围见 [执行图指南](guides/execution-graph.md) 和 [归档索引](../openspec/changes/archive/INDEX.md)。

- 进行中：
  - 暂无进行中的 OpenSpec change。
- 候选：
  - 暂无其他候选。

状态唯一来源：active change 以当前分支 `openspec list --json` 为准，已归档 change 以 [archive index](../openspec/changes/archive/INDEX.md) 为准。状态含义：需求触发=缺稳定失败复现；起草中=OpenSpec 尚未满足 `applyRequires`；可实施=owner、复现、部署范围、验证和回滚齐备；已归档=实现、门禁和索引完成。

## 候选提案

| 候选 / 触发信号 | Owner / 第一检查 | 最小复现与首批实施方向 | 停止线 / 未成立时 | OpenSpec |
| --- | --- | --- | --- | --- |
| 暂无候选。新的 trigger 只有在具备 owner、最小失败复现和停止线时才进入本表。 | — | — | 不以 DTO/API 存在或 README 描述作为单独立项依据。 | — |

后续候选按同一行动卡格式加入本表：触发信号、owner/入口、最小失败复现、首批实施方向、停止线和 OpenSpec slug 缺一不可。新信号先判断能否作为已有 contract 的增量，不为 DTO/API 已存在或 README 描述单独立项。

## 实施准入与固定边界

1. 先查代码中的 source owner、store、claim/restore seam 和 negative path；不能仅凭类型或 package 存在推断能力。
2. 先提交可重复的正向/负向失败证据；按适用性覆盖 fresh-process/reopen、crash window、duplicate/conflict 和 Run/Stream parity。
3. OpenSpec 的 proposal/design/tasks/spec delta 同步明确部署范围、首批改动、风险、回滚和不建设项；三份主工件均声明 Example Impact Assessment 与 Public Semantic Documentation Commitment。
4. 实施同步更新代码、测试、公开文档、示例和 dedicated gate；全量验证通过后归档、更新本页/索引，再合并并推送。

- source history/checkpoint 由 runner、child、team、workflow 或 mailbox owner 显式注入；Memory 仅 process-local，FileStore 默认仅同机单写、顺序重开。
- history replay 只读，不能触发 Run、Stream、retry、resume、ack、nack、tool 或 model，也不能替代 execution restore。
- continuation 能力按 `execution form × pause reason × owner store` 分别证明；callback EventID、application snapshot 或重新 Run 不构成支持证据。
- 缺少 owner checkpoint、atomic claim、fresh-process evidence 或 source facts 时，返回 typed `unsupported_capability` 或 `recovery-required`；不得伪造恢复。
- Run/Stream 必须保持同一 source terminal 语义，不得引入平行终止或决策路径。

## 既有范围

已归档能力（包括 `extract-common-execution-graph-boundaries` 的 source-neutral topology projection，以及 `coordinate-durable-source-stores-across-writers` 的 same-host multi-instance CAS/fencing）支持矩阵、失败分类、rollback 和证据以 [archive index](../openspec/changes/archive/INDEX.md) 与 package tests 为准。Tracing/eval、context organization、hooks/middleware、realtime、runtime budget、A64 harnessability、diagnostics、state/session snapshot 优先增量回收到现有 contract；不得借此改变 source ownership、恢复边界或质量门禁语义。

以下历史治理锚点由既有 contract gate 校验；完整范围见 [archive index](../openspec/changes/archive/INDEX.md)。

`establish-embedded-host-command-response-and-event-correlation-contract` 保留为 embedded-host 治理引用，不代表新的 active change。

- Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Context organization 语义能力同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在 Context JIT Organization 增量吸收；生产可用治理同域需求（压缩质量门控、冷存检索/清理、一致性回放、强门禁）统一在 a69 吸收，不再新增平行 context 压缩提案。
- Context organization 同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在本提案内增量吸收，不再新增平行 context 组织提案。
- Hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在本提案内以增量任务吸收，不再新增平行 realtime 提案。
- Runtime 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- A64 的 harnessability scorecard 只衡量契约、回放、门禁和文档证据；按 harness ROI/depth 分层，坚持 computational-first, inferential-second，并受门禁耗时预算治理约束，不能借此放宽运行时或质量语义。
- State/session snapshot 必须复用现有 checkpoint/snapshot 语义与既有 memory lifecycle，不得重写存储层事实源。

## 长期方向（不进入近期主线）

平台化控制面、跨租户全局调度与控制平面、市场化/托管化 adapter registry 能力、hosted gateway 和远程 event store 不进入近期主线；未来分别以 owner-boundary evidence 单独立项。

## 验证命令

候选起草前运行相关 package 与 reproduction；归档前至少执行：

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

```powershell
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
openspec validate --all
pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1
```
