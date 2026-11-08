# Development Roadmap

更新时间：2026-11-03

Baymax 提案发现与实施入口。按状态区确认当前工作，再按候选表判断是否有证据可启动；详细契约、任务和历史证据见 [OpenSpec changes](../openspec/changes/)、[归档索引](../openspec/changes/archive/INDEX.md) 与 [mainline contract test index](mainline-contract-test-index.md)。

## 版本阶段口径（延续 0.x）

Baymax 处于 pre-`1.0.0` 阶段，不做 `1.0.0` / prod-ready 承诺。允许新增能力型提案；新增提案准入规则（0.x 阶段）要求先确定 owner、`Why now`、最小失败复现、部署范围、风险/回滚、文档影响与验证命令。提案范围限于契约一致性、可靠性与安全、质量门禁回归治理或外部接入 DX。

## 当前状态

已归档 196：`deployable-source-history-store-contract` 已交付显式 source-owner store binding、same-host multi-instance fencing/CAS、稳定事件 identity 的幂等 append、有界 replay 与 projection reconciliation；跨机器协调和 source/projection 原子提交继续明确为 unsupported。范围和验证证据见 [归档索引](../openspec/changes/archive/INDEX.md)。此前归档 195 的 `team-native-continuation-checkpoint-restore` 已完成 parallel/vote team 的 source-owned checkpoint/restore 证据与 same-host 能力边界。

- 已归档：
  - `child-composer-continuation-contract`（F；归档 194；child continuation 仅在显式 File owner + native history + executor 下声明，验证 profile 仍为 same-host/single-writer/sequential reopen）。
  - `refresh-upstream-findings-current-pin`（归档 197；以当前 pin 固化 upstream findings、Baymax contract evidence、部署边界和 provider live-smoke 的 `skipped-unconfigured` 规则；不提升 unsupported capability）。

- 进行中：无。

I 仍是根据现有 capability evidence 登记的条件候选，不代表已声明支持或已创建 change。

状态唯一来源：active change 以当前分支 `openspec list --json` 为准，已归档 change 以 [archive index](../openspec/changes/archive/INDEX.md) 为准。状态含义：需求触发=缺稳定失败复现；起草中=OpenSpec 尚未满足 `applyRequires`；可实施=owner、复现、部署范围、验证和回滚齐备；已归档=实现、门禁和索引完成。

## 待实施方向与已交付基线

| 候选 / 触发信号 | Owner / 第一检查 | 最小复现与首批实施方向 | 停止线 / 未成立时 | OpenSpec |
| --- | --- | --- | --- | --- |
| **A / 已归档 189**<br>恢复授权 provenance 与 revalidation | 宿主需要在 continuation restore 前说明原授权依据、当前 authority/scope/revision/effect binding 是否仍有效，且现有 source checkpoint 证据不能完整回答该问题。 | `core/types`、`core/runner`、host resume admission owner；构造已归档 checkpoint，在 authority、write scope、plan revision 或 source listen scope 变化后请求恢复，断言副作用前拒绝、checkpoint 未消费且返回稳定分类。已交付可选 admission provenance、fresh revalidation 与 replay/gate，作为后续方向的恢复授权基线。 | 不新建全局授权服务、业务账本或跨 source 身份映射；不能由 callback、DTO、复用 RunID 或 observer event 推断授权。 | [host-authorization-provenance-and-resume-revalidation](../openspec/changes/archive/189-host-authorization-provenance-and-resume-revalidation/) |
| **B / 已归档 190**<br>source adapter 生命周期契约 | 新 source/integration adapter 需要保留外部事实的稳定 identity、cursor、source metadata、投递幂等和故障归属；现有 adapter 接入若无法证明这些事实则不得扩展。 | `host/sourceadapter`、integration adapter owner；已交付 source-bound metadata/identity、pending receipt CAS、cursor-after-commit、`sent/gone/retryable`、health、dispose/abort、同机单写文件重开、只读 replay 与离线示例；集成证据证明 Observer 不推进 cursor、确认失败重投经 host CAS 不重复 effect。 | 不把 callback EventID、host observation sequence 或 adapter cache 当 durable cursor；不引入全局 source store、平行 terminal owner 或 replay 侧 effect。 | [source-adapter-lifecycle-contract](../openspec/changes/archive/190-source-adapter-lifecycle-contract/) |
| **C / 已归档 191**<br>attention-oriented runtime diagnostics projection | 宿主需要从现有 RuntimeRecorder/diagnostics 中直接找到需要人工决定的恢复、授权、source、effect 或 gate 问题，而不是自行拼接大量原始字段。 | `runtime/diagnostics` owner；已交付只读快照投影、有限 next action/owner/reference、原子校验、稳定排序/去重/限量、严格 replay 与真实 Composer Run/Stream admission 证据；healthy 和自动 retry 不生成待处理项。 | 不建设 Baymax panel、远程控制面、全局 notification service 或新的诊断写入口；RuntimeRecorder 继续是唯一诊断写入路径。 | [attention-oriented-runtime-diagnostics-projection](../openspec/changes/archive/191-attention-oriented-runtime-diagnostics-projection/) |
| **D / 已归档 192**<br>model surface revision 与 bounded context | host resolver 返回材料增长，或宿主需要判断配置、prompt、tool schema 哪个安全版本进入 provider-neutral request projection。 | `context/hostprojection`、`model/conformance` owner；固定有效 UTF-8 role/content/aggregate byte bounds，非法或超限整组拒绝；显式 opt-in 三槽安全 facts 进入 canonical projection、expected/observed drift、offline replay 和 Run/Stream parity。 | 不截断或部分注入；不增加配置、诊断 writer、QueryRuns 字段或 prompt registry；不从角色、tool order、capability、cache control 推断 model surface；不让该证据拥有授权或 effect。 | [model-surface-revision-and-bounded-context-contract](../openspec/changes/archive/192-model-surface-revision-and-bounded-context-contract/) |
| **E / 已归档 193**<br>role-scoped runtime ports 与 owner index | 现有证据索引无法为 resolver material、model surface 和 source adapter 分别定位唯一 owner 及完整验证链。 | `tool/contributioncheck` 与 documentation owner；已交付可校验的 owner → contract → replay → gate → docs 索引，分别登记三个独立语义，并通过阻断式离线 validator 和成对质量门禁验证。 | 没有两个不同 caller 通过同一宽接口越权的复现，不收窄 runtime port；不建立全局 service locator 或从证据推断执行/恢复能力。 | [role-scoped-runtime-ports-and-owner-index](../openspec/changes/archive/193-role-scoped-runtime-ports-and-owner-index/) |
| **F / 已归档 194**<br>child/composer continuation public contract | child/subagent/fanout 的执行 wiring、身份和失败传播已有基础；已补齐 child pause-specific continuation evidence，但支持范围仍受 owner/deployment profile 限制。 | `orchestration/composer` 与 child continuation owner；已交付 source-position checkpoint binding、fresh-process evidence、三类 pause cell、stable cursor/retention floor、expired/gap typed error、重复恢复与已完成 effect 不重跑，并将 capability promotion 绑定到显式 File owner + native history + executor。 | 不把 child 执行能力写成跨机器或多写者恢复能力；不得借用 workflow/team/mailbox history 或重新 Run parent 模拟恢复；任一 owner、history 或 executor 证据缺失时 planner 继续返回 `unsupported_capability`，不得降级 direct。 | [child-composer-continuation-contract](../openspec/changes/archive/194-child-composer-continuation-contract/) |
| **G / 已归档 195**<br>parallel/vote team source-owned checkpoint/restore | team parallel/vote 的并发、成员身份、聚合/投票和失败传播已有基础；已完成恢复摘要绑定完整聚合策略、vote provenance 和部署能力证据。 | `orchestration/teams` owner；已补齐 team policy digest/checkpoint metadata，验证 completed/pending member、投票参与边界、tie-break、duplicate claim、fresh-process restore 与 Run/Stream parity；能力条件明确暴露 same-host 限制。 | 未配置 team source owner/store 时继续 terminal-only/unsupported；不能从 child/workflow 支持推导 team 支持，不能以 mailbox lifecycle 或 history replay 替代 team checkpoint；ambiguous in-flight member 必须 recovery-required/unsupported。 | [team-native-continuation-checkpoint-restore](../openspec/changes/archive/195-team-native-continuation-checkpoint-restore/) |
| **H / 已归档 196**<br>可部署的 source-owned durable history store contract | 现有 native history 能表达 source identity、sequence、opaque cursor、retention floor 与 bounded replay；FileStore/既有证据仍不足以声明跨机器、多写者、多实例生产恢复。 | `core/history`、runner、child、team、workflow、mailbox source owner；已交付显式 owner-store factory/deployment binding、same-host multi-instance CAS/fencing、stable event identity、幂等 append、bounded `ReadAfter`、retention/expired/gap typed errors 与 crash-before-projection replay/reconciliation；source append 与 Danny projection 不做跨项目事务，改以 source identity/cursor + projection idempotency 明确对账语义。 | 不把 callback EventID、application snapshot 或静态 manifest 当 source history；不声称跨机器协调或 source/projection 原子提交；source store 不可用必须 `recovery-required`，不能从头重跑不确定 effect。 | [deployable-source-history-store-contract](../openspec/changes/archive/196-deployable-source-history-store-contract/) |
| **I / 条件候选**<br>mailbox lifecycle 与 Agent pause restore 边界 | mailbox 已具备 admission、lease、ack/requeue/DLQ、lifecycle replay 和一次逻辑 admission，但不是 Agent continuation owner；是否需要 mailbox 承载 Agent 任意暂停点恢复尚未定案。 | `orchestration/mailbox` 与真实 execution owner；先由产品明确“只要可靠投递/重试”还是“跨进程 Agent pause restore”。前者仅补齐边界文档和拒绝证据；后者另立 API，绑定 mailbox item/native checkpoint，定义 lease authority 转移、completion/ack/retry/DLQ 与 terminal evidence 关系，禁止 redelivery 重跑 uncertain effect，并保持 mailbox lifecycle cursor 与 Agent event cursor 独立。 | 在产品选择前不得暗中宣称 mailbox continuation；不能用 mailbox replay 模拟 runner/child/team/workflow restore，也不能把 lease reclaim 当 checkpoint claim。 | `mailbox-agent-pause-restore-boundary`（待立项） |
| **J / 已归档 197 证据刷新**<br>upstream findings 与当前 Baymax pin 对齐 | upstream finding 仍记录旧 pin，不能把旧版本结论直接作为当前版本正式复现；当前代码/contract evidence pin 为 `v0.0.0-20261103091945-d2e08cd6dc8c`。 | docs/upstream owner；已以当前 pin 固化 child/team/source-history/mailbox/cancellation/retry 的证据矩阵、部署前提和未验证项；旧 pin 仅作为历史背景，不关闭或迁移结论。 | 不因旧 finding 推断当前版本 bug；无真实 endpoint/model/key 时 live-smoke 保持 `skipped-unconfigured`，不写成 Baymax 代码失败或成功。 | [refresh-upstream-findings-current-pin](../openspec/changes/archive/197-refresh-upstream-findings-current-pin/) |

这些方向来自对 [deepseek-ai/dsh-superagent-prototype](https://github.com/deepseek-ai/dsh-superagent-prototype) 九章设计文档的架构审阅；它们是 Baymax 可借鉴的 owner、恢复、adapter、诊断和模型可见面治理方向，不表示采用其 DeepSeek Harness 控制面、SQLite application runtime、`danger-full-access` worker 或跨 source 身份 prompt 映射。F–J 是当前 Baymax 能力证据中明确暴露的后续候选：执行 wiring 已存在不等于 continuation restore 已支持，source history 可 replay 不等于生产级恢复，mailbox lifecycle replay 不等于 Agent checkpoint owner。后续候选按同一行动卡格式加入本表：触发信号、owner/入口、最小失败复现、首批实施方向、停止线和 OpenSpec slug 缺一不可。新信号先判断能否作为已有 contract 的增量，不为 DTO/API 已存在或 README 描述单独立项。

### 后续启动顺序与验收重点

下表给出实施顺序和最小验收目标；候选 OpenSpec slug 作为后续立项参考，是否已启动以对应分支的 OpenSpec 查询结果为准。每项满足准入后单独起草、实施、验证和归档，前项完成后再评估下一项；无需为进入下一阶段而扩大当前提案。

| 顺序 | 待实施方向 | 首批可评审交付 | 最小验收重点 |
| --- | --- | --- | --- |
| 已交付 / B | Source adapter 生命周期（归档 190） | source-bound metadata、external identity、cursor 与 `sent/gone/retryable` 契约；health、dispose/abort 边界及离线示例。 | `sent/gone` 确认后才推进 cursor；retryable、取消和持久化前中断保留原 cursor；重复 identity 经 host effect CAS 保持幂等，冲突 identity 拒绝；重开后从已确认位置继续。 |
| 已交付 / C | 面向需处理事项的诊断投影（归档 191） | 从既有 diagnostics 生成只读、有界的 actionable items，包含稳定分类、next action、owner/reference。 | healthy 记录不产生待处理项；unsupported、recovery-required、effect-pending、gate-failed 各有确定归属与行动；不增加诊断 writer 或触发执行。 |
| 3 / D | 模型可见面版本与有界上下文 | resolver material all-or-nothing byte bounds；显式 opt-in configuration/prompt/tool-schema safe facts；canonical request projection、replay、drift 与 provider audit gate。 | 单项与累计超限均整组拒绝且无 partial material；缺省 model surface unavailable；历史 fixture 兼容、未知字段不改变 canonical digest；Run/Stream 一致；外部内容不成为授权依据。 |
| 4 / E | 按角色收窄 runtime port 与 owner 索引 | owner 无法追踪的复现已成立，先维护 owner → contract → replay → gate → docs 索引。 | 已登记语义可定位唯一 owner 和验证证据；port 收窄另需至少两个 caller 证明越权问题，不在本次实施。 |
| 5 / F | child/composer continuation public contract | bounded child checkpoint、fresh-process restore、三类 pause cell、cursor retention/gap、effect 不重跑和 manifest promotion evidence。 | 任一 pause cell 缺新进程证据则保持 child/fanout unsupported，不降级 direct。 |
| 6 / G | parallel/vote team native checkpoint/restore | team/member/aggregate-vote 统一 checkpoint、atomic claim、pending-only restore、vote provenance/tie-break、same-host 条件和 Run/Stream parity。 | 无 team owner/store 或存在 ambiguous in-flight member 时返回 unsupported/recovery-required。 |
| 7 / H | deployable source history store | owner factory 绑定、多写者 CAS/fencing、幂等 event identity、bounded replay、crash reconciliation 和不可用时 recovery-required。 | 未证明跨机器/多实例部署前提时，只声明已验证的同机 profile。 |
| 8 / I | mailbox 边界决策 | 可靠投递目标只维护 lifecycle owner；若要求 Agent pause restore，另行定义 checkpoint/lease/terminal binding API。 | 产品目标未确定前不新增隐式 continuation 语义。 |
| 9 / J | current-pin upstream finding refresh | 以当前 Baymax pin 重做正式复现，更新未验证矩阵和 live-smoke 配置前提。 | 旧 pin 仅作历史记录；无真实 provider 配置时保持 skipped-unconfigured。 |

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
