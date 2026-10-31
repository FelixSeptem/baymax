# Development Roadmap

更新时间：2026-10-31

这是 Baymax 的实施决策索引，不是设计文档。先用下方候选表定位状态、触发证据、代码入口和实施方向，再进入对应的 OpenSpec；详细契约、任务和证据以 [OpenSpec changes](../openspec/changes/)、[归档索引](../openspec/changes/archive/INDEX.md) 和 [mainline contract test index](mainline-contract-test-index.md) 为准。

## 版本阶段口径（延续 0.x）

Baymax 处于 pre-`1.0.0` 阶段，不做 `1.0.0` / prod-ready 承诺。允许新增能力型提案；新增提案准入规则（0.x 阶段）要求明确 owner、最小 failing reproduction、`Why now`、风险、回滚、文档影响和验证命令，目标限于契约一致性、可靠性与安全、质量门禁回归治理或外部接入 DX。

## 当前状态

- 进行中：
  - 暂无。

### 已归档状态

- 已归档：
  - `establish-provider-structured-context-cache-runtime`（provider-native prompt-cache controls 与安全 projection）。
  - `publish-workflow-successor-checkpoints-atomically`（workflow successor checkpoint 原子发布）。
- 候选：
  - 需求触发：多写者/跨机器 source store、mailbox handler context safe point、通用 ExecutionGraph；当前只有边界或需求信号，没有足够的原始运行时失败证据。

## 候选索引

| 优先级 / 状态 | 候选 | 触发证据 | 第一入口 / owner | 实施方向 | 停止线 |
| --- | --- | --- | --- | --- | --- |
| B / 需求触发 | 多写者 / 跨机器 durable source store | 真实 concurrent writer、跨机器 resume/cursor、stale writer 或 crash 触发；当前 FileStore 只证明同机单写、顺序重开 | source history/checkpoint owner；FileStore、CAS、claim seam | source-owned CAS、writer fencing、atomic checkpoint claim/cursor advancement、crash recovery、多实例 duplicate-resume 测试；或明确限制部署范围 | 无真实部署触发不立项；不得把单写 FileStore 宣称为分布式恢复 |
| C / 需求触发 | mailbox handler context safe point | 需要宿主 handler pause/restore；当前只有 mailbox lifecycle/host-plan continuation | mailbox owner；handler safe point、checkpoint/projection binding | 持久化 handler continuation；恢复前重新授权 resolver；保留 message、delivery、lease、attempt、causation、authority、terminal identity | lifecycle history 不等于 handler continuation；无 owner evidence 的 pause cell 返回 typed unsupported/recovery-required |
| D / 需求触发 | 通用 ExecutionGraph | 多个宿主 fixture 证明 workflow/team/composer 无法表达共享 topology、dependency 或 terminal 语义 | graph definition/validator owner；definition、validator、restore seam | 只抽取已验证的共享 definition/validation/restore 边界；按能力拆 change，不建平行 executor/state machine | 无共享 owner 边界则停止；不得由统一 DTO 推断统一恢复能力 |

状态口径：**起草中**表示 active OpenSpec 未满足 `applyRequires`；**需求触发**表示缺少可重复的原始失败证据；**可实施**要求 owner、最小复现、部署范围、回滚点、契约测试和文档影响齐备；**已归档**要求实现、证据、门禁通过并写入归档索引。

## 统一准入与实施方向

每个候选必须按同一条路径推进：

1. **审计代码事实**：定位 source owner、store、claim/restore seam 和现有 negative path；不能只凭 DTO、package 存在或 README 立项。
2. **写最小失败复现**：测试或 fixture 必须稳定展示缺失、错误状态或不可解释恢复；包含 positive、negative、fresh-process/reopen、crash window、duplicate/conflict 和 Run/Stream parity（适用时）。
3. **确定第一批改动**：列出 owner、第一入口、deployment scope、回滚点和“不建设”的平行 runtime。
4. **起草 OpenSpec**：`proposal.md`、`design.md`、`tasks.md`、spec delta 同步完成；三份工件都包含 Example Impact Assessment 与 Public Semantic Documentation Commitment。
5. **实施并验证**：代码、测试、文档、dedicated gate 同步提交；完成后归档 change，更新本表，再合并最新 `master` 并推送。

没有 owner checkpoint、atomic claim、fresh-process evidence 或 source facts 的能力，必须返回 typed `unsupported_capability` 或 `recovery-required`，不得通过重跑 parent、history replay、相同 run ID 或 application pause 标记伪造恢复。

## 固定运行时边界

- source history 与 checkpoint 由 runner、child、team、workflow、mailbox 的 owner 显式注入；Memory 只保证 process-local，FileStore 当前只承诺同机单写、顺序重开。
- history replay 只能作为只读投影输入，不能调用 Run、Stream、retry、resume、ack、nack、tool 或 model，也不能替代 execution restore。
- continuation 是 `execution form × pause reason × owner store` 的路径能力；相邻 form、callback EventID、application snapshot 和重新 Run 都不能推断支持。
- 跨机器、多实例、distributed cursor 或统一 mailbox/runner transaction 必须另有 CAS、fencing、atomic claim 与 crash evidence；未满足前保持 unsupported。
- Run/Stream 必须保持同一 source terminal 语义；不允许用平行终止或决策路径掩盖 owner 差异。

## 已归档能力与不扩大范围

归档索引已覆盖 direct continuation、owner history injection/recovery、workflow restore、composite pause matrix、mailbox async、host plan、context handoff、capability manifest 以及 workflow successor checkpoint 原子发布。支持矩阵、失败分类、rollback 和 evidence 以归档索引与 package tests 为准。

以下需求只允许增量回收到已有 contract，不另开平行提案：tracing/eval、context organization、hooks/middleware、realtime、runtime budget、A64 harnessability、diagnostics、state/session snapshot。它们不得改变 source ownership、恢复边界或质量门禁语义。

## 历史治理锚点

以下固定句由既有 contract gate 校验，完整范围和历史证据见 [归档索引](../openspec/changes/archive/INDEX.md)。

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

平台化控制面、跨租户全局调度与控制平面、市场化/托管化 adapter registry 能力，以及 hosted gateway 和远程 event store 不进入近期主线；未来启动时必须分别拆成独立 OpenSpec。

## 验证命令

受影响 package 和 dedicated gate 先通过；归档前至少运行：

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

状态来源固定为 `openspec list --json` 的 active changes 与 `openspec/changes/archive/INDEX.md`；active change 必须出现在本页，归档 change 必须从 active 候选移出并保留归档索引链接。
