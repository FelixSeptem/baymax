# Development Roadmap

更新时间：2026-11-03

Baymax 提案发现与实施入口。按状态区确认当前工作，再按候选表判断是否有证据可启动；详细契约、任务和历史证据见 [OpenSpec changes](../openspec/changes/)、[归档索引](../openspec/changes/archive/INDEX.md) 与 [mainline contract test index](mainline-contract-test-index.md)。

## 版本阶段口径（延续 0.x）

Baymax 处于 pre-`1.0.0` 阶段，不做 `1.0.0` / prod-ready 承诺。允许新增能力型提案；新增提案准入规则（0.x 阶段）要求先确定 owner、`Why now`、最小失败复现、部署范围、风险/回滚、文档影响与验证命令。提案范围限于契约一致性、可靠性与安全、质量门禁回归治理或外部接入 DX。

## 当前状态

已归档 191：`attention-oriented-runtime-diagnostics-projection` 提供只读、有界且确定性的待处理事项投影、稳定 next action/owner/reference 与只读 replay。189 的恢复授权和 190 的 source adapter 生命周期为既有基线；RuntimeRecorder 和各 source/effect owner 保持独立。范围和验证证据见 [归档索引](../openspec/changes/archive/INDEX.md)。

- 进行中：
  - 暂无进行中的 OpenSpec change。
- 候选：
  - 下方实施方向表记录五项方向；A、B、C 已归档，后续按 D→E 评估准入并推进，E 保留条件候选状态。

状态唯一来源：active change 以当前分支 `openspec list --json` 为准，已归档 change 以 [archive index](../openspec/changes/archive/INDEX.md) 为准。状态含义：需求触发=缺稳定失败复现；起草中=OpenSpec 尚未满足 `applyRequires`；可实施=owner、复现、部署范围、验证和回滚齐备；已归档=实现、门禁和索引完成。

## 待实施方向与已交付基线

| 候选 / 触发信号 | Owner / 第一检查 | 最小复现与首批实施方向 | 停止线 / 未成立时 | OpenSpec |
| --- | --- | --- | --- | --- |
| **A / 已归档 189**<br>恢复授权 provenance 与 revalidation | 宿主需要在 continuation restore 前说明原授权依据、当前 authority/scope/revision/effect binding 是否仍有效，且现有 source checkpoint 证据不能完整回答该问题。 | `core/types`、`core/runner`、host resume admission owner；构造已归档 checkpoint，在 authority、write scope、plan revision 或 source listen scope 变化后请求恢复，断言副作用前拒绝、checkpoint 未消费且返回稳定分类。已交付可选 admission provenance、fresh revalidation 与 replay/gate，作为后续方向的恢复授权基线。 | 不新建全局授权服务、业务账本或跨 source 身份映射；不能由 callback、DTO、复用 RunID 或 observer event 推断授权。 | [host-authorization-provenance-and-resume-revalidation](../openspec/changes/archive/189-host-authorization-provenance-and-resume-revalidation/) |
| **B / 已归档 190**<br>source adapter 生命周期契约 | 新 source/integration adapter 需要保留外部事实的稳定 identity、cursor、source metadata、投递幂等和故障归属；现有 adapter 接入若无法证明这些事实则不得扩展。 | `host/sourceadapter`、integration adapter owner；已交付 source-bound metadata/identity、pending receipt CAS、cursor-after-commit、`sent/gone/retryable`、health、dispose/abort、同机单写文件重开、只读 replay 与离线示例；集成证据证明 Observer 不推进 cursor、确认失败重投经 host CAS 不重复 effect。 | 不把 callback EventID、host observation sequence 或 adapter cache 当 durable cursor；不引入全局 source store、平行 terminal owner 或 replay 侧 effect。 | [source-adapter-lifecycle-contract](../openspec/changes/archive/190-source-adapter-lifecycle-contract/) |
| **C / 已归档 191**<br>attention-oriented runtime diagnostics projection | 宿主需要从现有 RuntimeRecorder/diagnostics 中直接找到需要人工决定的恢复、授权、source、effect 或 gate 问题，而不是自行拼接大量原始字段。 | `runtime/diagnostics` owner；已交付只读快照投影、有限 next action/owner/reference、原子校验、稳定排序/去重/限量、严格 replay 与真实 Composer Run/Stream admission 证据；healthy 和自动 retry 不生成待处理项。 | 不建设 Baymax panel、远程控制面、全局 notification service 或新的诊断写入口；RuntimeRecorder 继续是唯一诊断写入路径。 | [attention-oriented-runtime-diagnostics-projection](../openspec/changes/archive/191-attention-oriented-runtime-diagnostics-projection/) |
| **D / 需求触发**<br>model surface revision 与 bounded context | provider request projection、tool schema、context budget、capability evidence 或 runtime config 变化后，集成方需要确定某次模型请求实际看到了什么版本和有界内容。 | `context`、`runtime/config`、`model/*`、`tool/*` owner；以同一 run 的配置/prompt/tool schema revision 变更和上下文超限 fixture，断言请求快照、截断/拒绝分类、Run/Stream parity 与 replay 证据一致。首批仅建立 revision/reference contract、bounded resource read model 与 drift gate。 | 不把 prompt 或外部文本提升为 authorization、terminal 或 effect owner；不引入隐式全局 prompt registry 或 provider SDK 到 `context/*`。 | `model-surface-revision-and-bounded-context-contract` |
| **E / 条件候选**<br>role-scoped runtime ports 与 owner index | 新 integration 或 orchestration 能力需要调用过宽接口，或文档无法从 owner 追溯到 contract、replay、gate、测试和公开语义维护路径。 | `core/*`、`orchestration/*` 与 architecture/documentation owner；选择至少两个不同 caller（如 source adapter 与 host integration），证明它们可通过同一宽接口越过各自职责边界，或 evidence index 无法定位唯一 writer。首批按角色定义窄 port，并扩展 owner → contract → replay → gate → docs 索引。 | 不以接口重命名或 package 数量为立项依据；不建立平行 runtime、全局 service locator 或由 topology digest 推断执行/恢复能力。 | `role-scoped-runtime-ports-and-owner-index` |

这些方向来自对 [deepseek-ai/dsh-superagent-prototype](https://github.com/deepseek-ai/dsh-superagent-prototype) 九章设计文档的架构审阅；它们是 Baymax 可借鉴的 owner、恢复、adapter、诊断和模型可见面治理方向，不表示采用其 DeepSeek Harness 控制面、SQLite application runtime、`danger-full-access` worker 或跨 source 身份 prompt 映射。后续候选按同一行动卡格式加入本表：触发信号、owner/入口、最小失败复现、首批实施方向、停止线和 OpenSpec slug 缺一不可。新信号先判断能否作为已有 contract 的增量，不为 DTO/API 已存在或 README 描述单独立项。

### 后续启动顺序与验收重点

下表给出实施顺序和最小验收目标；候选 OpenSpec slug 作为后续立项参考，是否已启动以对应分支的 OpenSpec 查询结果为准。每项满足准入后单独起草、实施、验证和归档，前项完成后再评估下一项；无需为进入下一阶段而扩大当前提案。

| 顺序 | 待实施方向 | 首批可评审交付 | 最小验收重点 |
| --- | --- | --- | --- |
| 已交付 / B | Source adapter 生命周期（归档 190） | source-bound metadata、external identity、cursor 与 `sent/gone/retryable` 契约；health、dispose/abort 边界及离线示例。 | `sent/gone` 确认后才推进 cursor；retryable、取消和持久化前中断保留原 cursor；重复 identity 经 host effect CAS 保持幂等，冲突 identity 拒绝；重开后从已确认位置继续。 |
| 已交付 / C | 面向需处理事项的诊断投影（归档 191） | 从既有 diagnostics 生成只读、有界的 actionable items，包含稳定分类、next action、owner/reference。 | healthy 记录不产生待处理项；unsupported、recovery-required、effect-pending、gate-failed 各有确定归属与行动；不增加诊断 writer 或触发执行。 |
| 3 / D | 模型可见面版本与有界上下文 | 请求可见面的 revision/reference、资源读取边界、超限分类和 drift fixture。 | 配置、prompt、tool schema 改变可追溯；截断或拒绝行为有界且可回放；Run/Stream 一致；外部内容不成为授权依据。 |
| 4 / E | 按角色收窄 runtime port 与 owner 索引 | 在越权调用或 owner 无法追踪的复现成立后，定义窄 port 并维护 owner → contract → replay → gate → docs 索引。 | 至少两个 caller 证明现有边界问题；收窄后调用无法越过各自职责；每项公开语义可定位唯一 owner 和验证证据。无失败证据则继续保留为条件候选。 |

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
