# Development Roadmap

更新时间：2026-10-31

## 用法

这是 Baymax 的实施决策索引，不是设计文档。每个候选都必须能快速回答四个问题：

1. 什么运行时事实会触发它；
2. 先改哪个 owner 和代码入口；
3. 用什么最小复现证明缺口；
4. 何时可以归档，何时必须停止或回到 `unsupported_capability`。

详细方案、任务、spec delta 和历史证据分别见 [OpenSpec changes](../openspec/changes/)、[Archive Index](../openspec/changes/archive/INDEX.md) 与 [mainline contract test index](mainline-contract-test-index.md)。

## 版本阶段口径（延续 0.x）

Baymax 处于 pre-`1.0.0` 阶段，不做 `1.0.0` / prod-ready 承诺。允许新增能力型提案，但新增提案准入规则（0.x 阶段）要求先提交 owner、最小 failing reproduction、`Why now`、风险、回滚、文档影响和验证命令；目标必须落在契约一致性、可靠性与安全、质量门禁回归治理或外部接入 DX 之一。

## 当前队列

| 顺序 | 候选 | 状态 | 触发信号 | 第一入口 / owner | 最小复现 | 实施方向 | 完成 / 停止线 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `publish-workflow-successor-checkpoints-atomically` | **已归档** | workflow repeated pause 可能产生 orphan successor | `orchestration/workflow/continuation.go`、`engine.go` / workflow continuation owner | 在 successor save 与 predecessor commit 之间崩溃，fresh FileStore 重开 | 同一 owner transition 发布 predecessor projection 和 successor material；校验 lineage；duplicate resume 收敛 | 无 orphan；只继续 pending step；Run/Stream parity；明确不扩展为多写者 |
| 2 | 多写者 / 跨机器 durable source store | **需求触发** | 真实部署出现 concurrent writer、跨机器 resume/cursor transaction、stale writer 或崩溃 | source history/checkpoint owner；FileStore/CAS/claim seam | 两个 writer 同时 claim 或 advance cursor，确认 stale writer 被拒绝且不会重复恢复 | source-owned CAS、writer fencing、atomic claim/cursor、crash recovery、multi-instance evidence | 未有真实部署触发前不立项；证据不足保持 `需求触发` |
| 3 | mailbox handler context safe point | **需求触发** | 宿主需要跨 handler pause/restore，且 mailbox owner 能持久化 handler material | mailbox owner；handler safe point、checkpoint/projection binding | handler pause 后重启，确认 lease、attempt、terminal 仍由 mailbox owner 产生 | 保存 handler continuation material；fresh resolver reauthorization；保留 message/delivery/causation/authority identity | lifecycle history 不得冒充 execution continuation；任一 pause cell 无 owner evidence 就拒绝 |
| 4 | Provider structured context / Prompt Cache | **需求触发** | provider fixture 证明 role/tool-result/order 丢失，或 cache 成本/P95 稳定超标 | `model/<provider>` adapter owner；projection/conformance | 同一 logical request 对比结构化字段、cache 命中、成本和延迟 | 扩展 provider-native projection 与 conformance fixture；记录成本/P95 和 rollback evidence | 不建设 gateway、credential store 或 runtime selector；无稳定 fixture 不启动 |
| 5 | 通用 ExecutionGraph | **需求触发** | 多个宿主无法表达同一结构，fixture 证明既有 workflow/team/composer 不足 | graph definition/validator owner；先审计 definition、validator、restore seam | 用现有宿主表达同一 graph，证明 topology/dependency/terminal 语义不能共享 | 只抽取已验证的共享 definition/validation/restore 边界；按能力拆小 change | 不重建平行 executor、topology 或 state machine；无法形成共享边界就停止 |

## 统一准入

候选只有在以下证据齐备后才能进入 OpenSpec 实施：

- **代码事实**：已定位 source owner、store、claim/restore seam 和现有 negative path；不能只凭 DTO、package 存在或文档描述立项。
- **最小失败复现**：测试或 fixture 能稳定证明当前行为缺失、错误或会产生不可解释状态。
- **实施边界**：明确第一批 owner、回滚点、deployment scope，以及不会建设的平行 runtime。
- **可验证契约**：列出正向、负向、fresh-process/reopen、crash window、duplicate/conflict、Run/Stream parity 和 dedicated gate 证据。
- **OpenSpec 完整性**：`proposal.md`、`design.md`、`tasks.md` 和 spec delta 同步；三份工件都包含 Example Impact Assessment 与 Public Semantic Documentation Commitment。

没有 owner checkpoint、atomic claim、fresh-process evidence 或 source facts 的能力，必须返回 typed `unsupported_capability` 或 `recovery-required`，不得通过重跑 parent、history replay、相同 run ID 或 application pause 标记伪造恢复。

## 固定边界

- source history 与 checkpoint 由 runner、child、team、workflow、mailbox 的 owner 显式注入；Memory 只保证 process-local，FileStore 当前只承诺同机单写、顺序重开。
- history replay 是只读投影输入，不能调用 Run、Stream、retry、resume、ack、nack、tool 或 model，也不能替代 execution restore。
- continuation 是 `execution form × pause reason × owner store` 的路径能力；相邻 form、callback EventID、application snapshot 和重新 Run 都不能推断支持。
- 跨机器、多实例、distributed cursor 或统一 mailbox/runner transaction 必须另有 CAS、fencing、atomic claim 与 crash evidence；未满足前保持未支持。
- Run/Stream 必须保持同一 source terminal 语义；不允许通过平行终止或决策路径掩盖 owner 差异。

## 历史治理锚点

以下固定句是既有 contract tests 的机器可读闭包锚点，完整范围和归档编号见 [Archive Index](../openspec/changes/archive/INDEX.md)。`establish-embedded-host-command-response-and-event-correlation-contract` 保留为 embedded-host 治理引用，不代表新的 active change。

- Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Context organization 语义能力同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在 Context JIT Organization 增量吸收；生产可用治理同域需求（压缩质量门控、冷存检索/清理、一致性回放、强门禁）统一在 a69 吸收，不再新增平行 context 压缩提案。
- Context organization 同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在本提案内增量吸收，不再新增平行 context 组织提案。
- Hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- Realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在本提案内以增量任务吸收，不再新增平行 realtime 提案。
- Runtime 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。
- A64 的 harnessability scorecard 只衡量契约、回放、门禁和文档证据；按 harness ROI/depth 分层，坚持 computational-first, inferential-second，并受门禁耗时预算治理约束，不能借此放宽运行时或质量语义。
- State/session snapshot 必须复用现有 checkpoint/snapshot 语义与既有 memory lifecycle，不得重写存储层事实源。

## 已归档能力与明确不扩大

归档索引已经覆盖 direct continuation、owner history injection/recovery、workflow restore、composite pause matrix、mailbox async、host plan、context handoff、capability manifest，以及 workflow successor checkpoint 原子发布。具体支持矩阵、失败分类、rollback 和 evidence 以 [Archive Index](../openspec/changes/archive/INDEX.md) 与对应 package tests 为准。

以下边界在没有新提案和原始运行时证据前保持不变：

- mailbox lifecycle/history 不等于 handler execution continuation；
- direct context assembly 不等于 composite restore；
- FileStore reopen 不等于多写者、跨机器或 exactly-once；
- A64 harnessability、diagnostics、state/session snapshot 和同域 tracing/eval、hooks、realtime、runtime budget、context organization 需求只增量回收到既有 contract，不另开平行提案；
- hosted gateway、远程 event store、跨租户全局调度、平台化控制面和托管 adapter registry 不进入近期主线。

## 长期方向（不进入近期主线）

平台化控制面、跨租户全局调度与控制平面、市场化/托管化 adapter registry 能力，以及 hosted gateway 和远程 event store 均不进入近期主线；未来若启动，必须分别拆成独立 OpenSpec。

## 推进顺序

对任一候选，严格按以下顺序执行：

1. 从最新 `master` 审计代码并写最小 failing reproduction；
2. 证据满足准入后起草 OpenSpec，并切语义化功能分支；
3. 按 tasks 实施代码、测试、文档和 dedicated gate；
4. 运行全量质量、文档和 OpenSpec 校验；
5. 归档 change，更新本表，再合并到最新 `master` 并推送。

如果触发信号不存在、owner 无法注入、证据只能依赖应用层重跑，或 deployment scope 未声明，候选保持当前状态，不得为了推进队列而标记为 supported。

## 归档前验证

受影响 package 和 dedicated gate 必须先通过；归档前运行：

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

active change 必须出现在候选表；归档后从 active 列表移除，并在 Archive Index 保留链接和验证证据。
