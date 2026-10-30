# Development Roadmap

更新时间：2026-10-29

## 用法

本文件是 Baymax 的实施决策索引，不是设计文档。目标是快速回答三个问题：

1. 当前哪个缺口已经有代码级证据，可以起草 OpenSpec？
2. 第一阶段应该改哪个 source owner，交付什么可验证事实？
3. 哪些能力明确不能从相邻 form、DTO、callback 或应用层快照推断？

完整设计、任务、spec delta 和归档验证记录分别由 [OpenSpec changes](../openspec/changes/)、[Archive Index](../openspec/changes/archive/INDEX.md)、受影响 package README 和 contract tests 承载。本文件只保留导航、边界和启动闸门。

## 版本阶段口径（延续 0.x）

Baymax 仍处于 pre-`1.0.0` 阶段：不做 `1.0.0` / prod-ready 承诺，允许新增能力型提案，但每项能力都必须有 owner、contract、replay 和 gate 证据。新增提案准入规则（0.x 阶段）要求在 proposal、design、tasks 中写明 `Why now`、风险、回滚、文档影响和验证命令，并归入契约一致性、可靠性与安全、质量门禁回归治理或外部接入 DX 之一。

## 快速导航

状态含义：

- **进行中**：已有 active OpenSpec，代码、测试、文档和门禁必须在同一 change 内收敛。
- **可审计**：有明确的代码入口，但必须先提交最小 failing reproduction；复现成立后才起草。
- **需求触发**：没有当前排期承诺，只有真实部署或 fixture 触发后才进入审计。
- **已归档**：能力已进入主线，但只覆盖归档工件声明的 form、pause reason 和 deployment scope。

| 顺序 | 候选 / change | 状态 | 启动证据 | 第一实施方向 | 明确不做 |
| --- | --- | --- | --- | --- | --- |
| 1 | `composite-graph-restore-material` | **可审计** | 只有在 owner checkpoint 无法保存或消费 graph/node/dependency/completed-effect/aggregate/terminal lineage 时立项 | 只补缺失 owner 的 source-owned graph material、restore validation、atomic claim 和 successor lineage | application checkpoint、history replay、相同 run ID 再 `Run`、重新 enqueue |
| 2 | 多写者 / 跨机器 durable source store | **需求触发** | 真实部署提出并发 writer、跨机器 resume 或 cursor transaction，并有 stale-writer/崩溃复现 | source-owned CAS、writer fencing、atomic claim/cursor transaction、crash recovery、multi-instance evidence | 不把 FileStore 的同机单写顺序重开宣传为分布式恢复 |
| 3 | mailbox handler context safe point | **需求触发** | 宿主需要跨 handler pause/restore，且 mailbox owner 有可持久化 handler continuation material | mailbox owner safe point、checkpoint/projection binding、fresh resolver reauthorization；逐格更新 manifest | lifecycle history、host-side resume loop、planbridge 自建 queue/worker |
| 4 | Provider structured context / Prompt Cache | **需求触发** | SDK fixture 证明 role、tool-result correlation、ordering 丢失，或 cache 成本/P95 成为稳定瓶颈 | 在 `model/<provider>` 现有 projection/evidence owner 内补最小 conformance 或成本证据 | gateway、credential store、cache control plane、runtime selector |
| 5 | 通用 ExecutionGraph | **需求触发** | 多个独立宿主仍无法表达同一执行结构，且 fixture 证明既有 workflow/team/composer 不足 | 先审计既有定义和 validator，确认真实共享边界后再拆小 change | 因 Danny 模式名称新建统一 executor、拓扑或状态机 |

## 当前状态

- 进行中：无。
- 已归档：完整列表见 [Archive Index](../openspec/changes/archive/INDEX.md)。归档能力不能为相邻 form、pause reason 或 deployment scope 背书。

## 固定边界

以下规则适用于每个候选，除非新的 OpenSpec 明确改变并同步文档：

- source history 必须由 runner、child、team、workflow、mailbox 的 owner 显式注入。Memory owner 是 process-local；FileStore 目前只支持同机单写、已提交事实的顺序重开。
- native history 只负责只读事实恢复；它不能调用 Run、Stream、retry、resume、ack、nack、tool 或 model，也不能成为 execution resume 的替代。
- continuation 是 `execution form x pause reason x owner store` 的路径能力。direct 的 evidence 不能背书 child、team、workflow、mailbox 或 resumed planbridge。
- source append 与应用 projection 不天然原子。projection 提交前崩溃必须从同一 owner source replay 并幂等去重；source facts 不可用时返回 `recovery-required`，不得重跑不确定 effect。
- 未证明的能力必须在 capability manifest 中返回 typed `unsupported_capability` 或 `recovery-required`；不得用 DTO、package 存在、callback EventID、application pause 标记或相同 run ID 再运行伪造支持。
- 当前 deployment scope 默认是同机单写。多实例、跨机器、分布式 cursor 或统一 mailbox/runner 事务必须另有 CAS、fencing、atomic claim 和 crash evidence。

## 启动闸门

每个候选按以下顺序推进，缺一项就停在“可审计”或“需求触发”：

1. **代码审计**：定位 source owner、store、claim/restore seam 和现有 negative path。
2. **最小复现**：测试必须证明运行时事实缺失，而不是文档、DTO 或调用方尚未接入。
3. **OpenSpec**：从最新 `master` 创建语义化分支；维护 `proposal.md`、`design.md`、`tasks.md`、spec delta；三份工件都声明 Example Impact Assessment 和 Public Semantic Documentation Commitment。
4. **实现证据**：正负单测、fresh-process/reopen、crash window、duplicate/conflict、Run/Stream parity、replay fixture 和 dedicated gate 同步完成。
5. **文档与门禁**：更新受影响 README、integration guide、roadmap 和 contract-test index；通过质量、文档和 OpenSpec 校验后才归档。

## 第一候选审计清单

审计 `composite-graph-restore-material` 时只看真实 owner 实现和测试：

- `orchestration/workflow/continuation.go`
- `orchestration/teams/continuation.go`
- `orchestration/composer/continuation.go`
- 各 owner 的 continuation tests、capability manifest 和 restore gate

依次确认：

1. checkpoint 是否保存并 restore 消费 graph definition/digest、native node ID、dependency completion、pending/completed state、effect outcome、aggregate、budget/deadline、terminal arbitration、predecessor/successor lineage；
2. fresh-process restore 是否只 dispatch pending node，且 settled member/step/effect 不重跑；
3. claim 前是否校验 graph、capability subset、authority、plan、write scope、effect namespace、occurrence、causation、revision 和 budget；
4. 最小失败用例是否能证明缺口会造成错误终态、断链或重复 effect。

若四项均有 owner evidence，则不重复起草该候选，转到下一项。

## 已归档能力边界

近期归档已建立 direct continuation、source history injection/recovery、workflow owner restore、composite pause matrix、mailbox async、host plan、context handoff 和 capability manifest 基线。具体支持矩阵、证据引用、失败分类和 rollback 只以 archive index、对应 spec delta、package README 与 tests 为准；roadmap 不复制这些长文。

当前仍明确受限：

- mailbox continuation 只有 owner 声明的 lifecycle/history 能力时才能使用；没有 handler safe point 就必须保持 unsupported；
- child/team/workflow 只有拥有 source-owned checkpoint material、atomic claim 和 fresh-process restore evidence 的格子才可提升为 supported；
- context assembly/handoff 不能把 direct projection 推广到 composite；必须绑定实际 native node/checkpoint 并重新授权；
- FileStore 的“可重开”不等于多写者、跨机器或 exactly-once。

### 历史治理锚点

以下是既有 contract tests 的压缩闭包锚点，完整范围和归档编号见 [Archive Index](../openspec/changes/archive/INDEX.md)。`establish-embedded-host-command-response-and-event-correlation-contract` 保留为 embedded-host 治理引用，不代表新的 active change。

- Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Context organization 语义能力同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在 Context JIT Organization 增量吸收；生产可用治理同域需求（压缩质量门控、冷存检索/清理、一致性回放、强门禁）统一在 a69 吸收，不再新增平行 context 压缩提案。
- Context organization 同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在本提案内增量吸收，不再新增平行 context 组织提案。
- Hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在本提案内以增量任务吸收，不再新增平行 realtime 提案。
- Runtime 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- A64 的 harnessability scorecard 只衡量契约、回放、门禁和文档证据；按 harness ROI/depth 分层，坚持 computational-first, inferential-second，并受门禁耗时预算治理约束，不能借此放宽运行时或质量语义。
- State/session snapshot 必须复用现有 checkpoint/snapshot 语义与既有 memory lifecycle，不得重写存储层事实源。

## 长期方向（不进入近期主线）

平台化控制面、跨租户全局调度与控制平面、市场化/托管化 adapter registry 能力，以及 hosted gateway、远程 event store 等方向不进入近期候选；若未来启动，必须分别拆成独立 OpenSpec。

## 验证命令

局部实施至少运行受影响包和对应 gate；归档前运行：

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

```powershell
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
openspec validate --all
```

切换 active change 或归档后，额外运行：

```powershell
pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1
```

### 文档维护规则

- roadmap 只维护候选导航、当前状态和启动闸门；详细设计不要复制到这里。
- active change 必须在“当前状态/进行中”列出；归档 change 只从 archive index 获取完整列表。
- 示例相关变更先更新 `examples/agent-modes/MATRIX.md` 和对应 README；无示例变更必须在三份 OpenSpec 工件中说明理由。
- 公开语义变化必须在同一 change 中维护文档 owner、路径和验证命令；否则不得进入实施或归档。
