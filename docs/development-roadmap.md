# Development Roadmap

更新时间：2026-09-12

## 定位

Baymax 主线保持 `library-first + contract-first`：

- 交付可嵌入的 Go runtime，不建设平台化控制面。
- 以 OpenSpec、契约测试、回放和质量门禁驱动行为变更。
- 代码、测试、文档与 spec delta 在同一变更中收敛。

本文件是运行态决策入口，不重复归档提案的设计和任务细节。历史范围、设计理由和完整任务以 `openspec/changes/archive/` 为准。

## 当前状态

状态权威来源：

1. 活跃变更：`openspec list --json`。
2. 已归档变更：`openspec/changes/archive/INDEX.md`。
3. 示例交付状态：`examples/agent-modes/MATRIX.md`。

截至 2026-09-12：

- 已归档：
  - `introduce-runtime-steering-and-follow-up-input-contract`
  - `standardize-runtime-failure-taxonomy-and-terminal-outcome-contract`
  - `harden-runtime-event-stream-terminal-recovery-contract`
  - `harden-tool-lifecycle-and-failure-isolation-contract`
  - `establish-session-history-checkpoint-replay-contract`
  - `context-compression-runtime-handoff-contract`
  - `extension-lifecycle-governance-resource-resolution-contract`
  - `introduce-provider-model-capability-and-credential-preflight-contract`
  - `establish-embedded-host-command-response-and-event-correlation-contract`
- 候选：
  - 当前没有默认启动的 P0/P1 change。新 change 必须满足本文件的准入规则，并由明确的风险信号或宿主需求触发。

最近归档的变更完成了运行终态、事件恢复、工具失败隔离、会话/回放、上下文交接、扩展治理、provider/model 准入以及嵌入式宿主命令/事件关联的主线收口。较早的已完成能力请直接查阅 [Archive Index](../openspec/changes/archive/INDEX.md)。

## 版本阶段口径（延续 0.x）

当前仓库不做 `1.0.0` / prod-ready 承诺，继续沿用 `0.x` 治理口径（见 `docs/versioning-and-compatibility.md`）。在 `0.x` 阶段，版本号用于表达变更范围，不构成稳定兼容承诺；主线目标是持续收敛、可回归迭代。

`0.x` 阶段允许新增能力型提案，但新增能力必须有明确的宿主价值、可验证的 contract 边界和回滚路径。

## 已交付基线

以下能力已稳定并由归档提案、主干测试和门禁覆盖。后续需求必须作为增量扩展，复用既有术语和 source-of-truth，不得重新定义平行语义。

| 域 | 已交付基线 | 后续约束 |
| --- | --- | --- |
| Runtime 与 Protocol | Run/Stream、Agent Runtime Protocol、capability/context/admission、checkpoint/workspace provenance、权威终态 | Run/Stream 保持语义等价；执行、恢复和排队仍由 source runtime 拥有。 |
| Realtime 与 HITL | interrupt/resume、durable stream binding、cursor、catch-up/live-tail、terminal recovery | 不创建第二套 event ordering、cursor 或终态状态机。 |
| Tool、MCP 与 Security | 本地工具生命周期、MCP profiles、sandbox isolation/egress、allowlist、policy precedence | 工具和扩展不得绕过 policy、sandbox 或 `RuntimeRecorder`。 |
| Context 与 Memory | Context Assembler、压缩生产治理、压缩 handoff、memory SPI、scope/search/lifecycle | `context/*` 不直连 provider SDK；snapshot 保持唯一事实源。 |

State/session snapshot 必须复用现有 checkpoint/snapshot 语义与既有 memory lifecycle，不得重写存储层事实源。
| Orchestration | Workflow、Teams、A2A、Scheduler、Mailbox、task board、recovery | 不以新 change 建立平台化调度或统一多代理拓扑。 |
| Config、Readiness 与 Diagnostics | `env > file > default`、fail-fast、热更新原子回滚、readiness/admission、diagnostics replay | QueryRuns 和诊断 schema 仅 additive + nullable + default。 |
| Extension 与 Provider | extension lifecycle/resource resolution、adapter manifest/capability、静态 provider/model catalog、credential preflight | Provider 细节在 `model/<provider>`；不引入远程目录、credential store 或扩展市场。 |
| Evaluation 与 Gates | OTel/eval、corpus/badcase/experiment、semantic/performance/docs/contract gates | 评测和观测不演化为托管控制面。 |

### Harnessability scorecard 与门禁耗时预算治理

A64 的 harnessability scorecard 用于衡量契约覆盖、回放漂移、门禁接线、文档一致性和验证开销；它是门禁可审计性的计算型指标，不改变运行时语义。评估遵循 **harness ROI/depth** 分层，并坚持 **computational-first, inferential-second**：客观测试与结构化证据先于主观评估结论。门禁执行同时受门禁耗时预算治理约束，超出预算时记录并按既有回滚/降级路径处理，不自动放宽质量阈值。

完整的 proposal、design、spec、fixture 和 gate 映射见：

- `openspec/changes/archive/INDEX.md`
- `docs/mainline-contract-test-index.md`
- `docs/runtime-config-diagnostics.md`
- `docs/runtime-module-boundaries.md`
- `docs/pi-agent-comparison-and-adoption-study.md`（外部项目对照与后续提案筛选依据）

## 可启动候选

候选不是承诺排期。启动前必须先完成现状审计，并在 OpenSpec proposal 中记录 `Why now`、风险、回滚点、文档影响、Example Impact Assessment 和验证命令。

### P2：嵌入式宿主事件与请求响应接缝

**触发信号**：出现 IDE、桌面宿主、headless UI 或 HITL 客户端需要统一接入；或者现有 embedding 方无法把命令响应、异步事件与恢复操作安全关联。

**目标**：在已有 realtime、interrupt/resume 与 durable stream binding 之上定义 transport-neutral 的宿主请求/响应和事件关联接缝。首个 binding 可评估严格 JSONL framing，但协议不得依赖某种远程网关。

**必须复用**：`run_id`、cursor、sequence、dedupe、interrupt/resume、readiness admission、policy/sandbox、`RuntimeRecorder` 和现有终态 taxonomy。

**验证方向**：request correlation、重复响应、超时、断连 catch-up、过期 cursor、host action authorization、权威终态查询、Run/Stream parity、replay idempotency 和 framing drift gate。

**明确不做**：REST/SSE/WebSocket/gRPC gateway、托管连接、远程 Session/Artifact store、RBAC、多租户、平台化 UI 或第二套实时状态机。

**Example Impact Assessment（立项时）**：`修改示例`。先完成 `examples/agent-modes/realtime-interrupt-resume` 的文档基线，再增加可回归运行态场景。

完整对照、证据基线和不吸收项见 `docs/pi-agent-comparison-and-adoption-study.md`。该候选借鉴 Pi 的成熟 RPC 接入经验，但不复制其 AgentSession 或实验性远程 Session Server。

## 后续提案备选池（Pi 对照增量）

同域收口声明：Realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在本提案内以增量任务吸收，不再新增平行 realtime 提案。Tracing+eval 同域增量需求（语义映射、指标汇总、执行治理、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。Context organization 同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在本提案内增量吸收，不再新增平行 context 组织提案。Context organization 语义能力同域需求（reference-first、isolate handoff、edit gate、relevance swap-back、lifecycle tiering、task-aware recap）优先在 Context JIT Organization 增量吸收；生产可用治理同域需求（压缩质量门控、冷存检索/清理、一致性回放、强门禁）统一在 a69 吸收，不再新增平行 context 压缩提案。Hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。Runtime 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。

备选池用于记录可验证方向，不代表承诺排期。候选必须由真实宿主需求或可复现风险触发；没有触发证据时保持观察。提案启动后，其状态只进入“当前状态”，不在本表维护第二份进度。

| 成熟度 | 候选方向 | 可吸收点 | 必须复用的 Baymax owner | 触发信号与首要边界 |
| --- | --- | --- | --- | --- |
| 首选候选（现有 P2 细化） | 嵌入式宿主事件与请求响应接缝 | 严格 JSONL、request correlation、command response 与 async event 分离、pending 收口、stdout 保护 | Runner/Composer、Agent Runtime Protocol、Realtime、terminal arbiter、readiness/policy/sandbox、`RuntimeRecorder` | IDE、桌面宿主、headless UI 或外部 HITL 客户端需要运行中控制。不得只做 framing；必须先明确 source-owned active Run control。 |
| 与首选合并优先 | 宿主介导 HITL adapter | 将 confirm/select/input 投影为反向请求并按 ID 返回响应 | `ClarificationResolver`、`ActionGateResolver`、现有 timeout 和 HITL timeline | 首个宿主需要跨进程确认或输入。只适配既有 HITL，不建立新状态机。 |
| 条件候选 | 跨 Provider handoff 与 stream edge conformance | 跨 provider tool-call/thinking 转换、abort usage、overflow、Unicode/空内容 fixture | `model/<provider>`、provider admission、context handoff、failure taxonomy | 出现跨 provider 恢复或边界兼容回归。优先补测试、fixture 和 gate，不扩张 provider SDK 边界。 |
| 条件候选 | 外部 Extension authoring conformance | extension authoring eval、真实 workflow fixture、失败反馈 | extension lifecycle/resource resolution、manifest/capability、allowlist、sandbox | 出现新的真实扩展来源。不得建设 package manager/market，也不得无准入动态执行扩展。 |
| 观察候选 | Model catalog 与本地模型路由增量 | runtime model discovery、本地模型 router、明确 auth preflight | provider/model catalog、credential preflight、readiness、host injection | 静态或宿主注入 catalog 无法满足明确路由需求。不引入 credential store，不在 `context/*` 引入 provider SDK。 |
| 观察候选 | Eval transcript/artifact comparison | baseline/candidate harness、transcript 与 snapshot artifact | OTel/eval/corpus、checkpoint/artifact refs、diagnostics replay | 现有 eval 无法定位可复现质量回归。只增加有界引用与比较，不建立 artifact service。 |
| 审计项 | Durable operation 与状态所有权审计 | entries/registers/usage ledger/lane 作为 owner 检查框架 | Session history、checkpoint、snapshot、scheduler/mailbox/recovery、diagnostics | 先证明现有 checkpoint 无法恢复真实进行中 operation，再转为提案；不得先创建第二套 Session 状态模型。 |

备选池合并与排序规则：

1. active Run control、命令关联、JSONL framing、output integrity、event backpressure 和基础 HITL bridge 优先在“嵌入式宿主事件与请求响应接缝”内收敛，避免相互循环依赖。
2. steering/follow-up 会新增输入队列与时序语义，必须晚于基础宿主接缝并独立评审。
3. Provider、Extension、Eval 方向继续采用需求触发，不因外部项目存在同名能力而自动立项。
4. Durable operation 当前只做 owner 审计；Pi 的 lane/register/ledger 不成为 Baymax 默认公共模型。
5. Pi 的 experimental CBOR protocol、remote Session Server、attachment/lease、SQLite hosted backend 保持长期延后，不进入近期备选。

### 需求触发观察项

以下方向不预设提案顺序，只有出现相应信号才进入候选审计：

| 方向 | 触发信号 | 首要边界 |
| --- | --- | --- |
| 远程 model catalog 或本地模型路由 | 静态/宿主注入目录不足以支持明确的路由需求 | 以本页“Model catalog 与本地模型路由增量”为同一候选，不创建平行提案；不在 `context/*` 引入 provider SDK，不接入 credential store。 |
| 运行时成本或延迟治理增量 | 成本或 P95 抖动成为稳定主线瓶颈 | 复用 operation profile、timeout resolution、budget admission 与既有诊断字段。 |
| 评测或 tracing 增量 | 跨后端字段解释不一致，或出现可复现质量回归缺口 | 复用 OTel/eval/corpus 基线，不引入评测控制面。 |
| 外部 extension 生态增量 | 新的真实扩展来源、供应链审计或隔离需求超出现有 lifecycle contract | 复用 manifest/capability/allowlist；不建设 package manager 或市场。 |

## 示例状态

`examples/agent-modes/MATRIX.md` 是 agent-mode 示例范围、文档基线、实现状态、gate 和 replay 映射的唯一来源。旧的 numbered examples backlog 已完成迁移，不在本文件重复维护。

示例变更必须遵循：

- `examples/agent-modes` 先完成 `MATRIX.md` 与对应模式 README 的文档基线，再进入代码实现。
- 每项模式必须列出 semantic anchor、runtime path、expected markers、rollback notes、contracts、gates 和 replay。
- 示例只演示既有主线 contract，不得在 examples 中定义平行的配置、终态或观测语义。

## 新增提案准入规则（0.x 阶段）

从本文件生效起，`0.x` 阶段新增提案必须满足以下要求：

1. 直接服务于至少一类目标：
   - 契约一致性（Run/Stream、reason taxonomy、错误分层、兼容语义）。
   - 可靠性与安全（fail-fast、回滚、幂等、恢复边界、安全治理）。
   - 质量门禁回归治理（contract/perf/docs gate regression）。
   - 外部接入 DX（模板、迁移、脚手架、conformance）且可被 gate 验证。
2. 保持 library-first 边界，不引入平台化控制面能力。
3. 在 proposal、design 与 tasks 中显式说明：
   - `Why now`
   - 风险与非目标
   - 回滚点
   - 文档影响
   - Example Impact Assessment：`新增示例`、`修改示例`，或 `无需示例变更（附理由）`
   - 验证命令
4. 对行为、配置、诊断 schema 或 contract 的变更，必须同步 OpenSpec spec delta、正负边界测试、必要 integration、replay fixture 与 shell/PowerShell parity gate。
5. 非法配置和非法热更新必须 fail-fast，并原子回滚；新增诊断字段必须遵守 additive + nullable + default。

## 长期方向（不进入近期主线）

以下方向明确延后，除非有独立产品决策和拆分后的 OpenSpec：

- 平台化控制面（多租户、RBAC、审计和运营面板）。
- 跨租户全局调度与控制平面。
- 市场化/托管化 adapter registry 能力。
- Remote Runtime Gateway、托管 Session/Artifact persistence、独立 session server 与远程 event store。
- Pi experimental protocol 的 CBOR wire、session attachment/lease 与 SQLite hosted backend。

这些方向不能以单个大提案混合交付；未来若启动，必须分别定义 gateway、persistence profile、artifact resolver 与 authorization/governance profile。

## 通用架构约束

- `runtime/*` 禁止依赖 `mcp/http` 或 `mcp/stdio`。
- 非 `mcp/*` 包禁止依赖 `mcp/internal/*`。
- `context/*` 禁止直接引入 provider 官方 SDK。
- Provider 协议和模型适配细节必须落在 `model/<provider>`。
- 诊断写入必须走 `observability/event.RuntimeRecorder` 单写入口。
- 配置优先级固定为 `env > file > default`。
- QueryRuns/诊断字段变更保持 additive + nullable + default。
- Run/Stream 对等场景不得引入平行终止或决策语义。

## 执行与归档

单变更优先；并行变更必须在 proposal 中声明依赖和不重叠边界。

标准顺序：

1. `openspec list --json` 与现状审计。
2. `proposal.md`、`design.md`、`tasks.md`、spec delta，包含 Example Impact Assessment。
3. 代码、测试、回放、独立 gate 与文档同步实施。
4. 执行门禁：

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

```powershell
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
```

5. 使用归档脚本完成重命名和归档，禁止手工移动目录：

```powershell
pwsh -File scripts/openspec-archive-seq.ps1 -ChangeName "<change-name>"
```

6. 归档后同步更新本文件的“当前状态”、README 里程碑快照、`docs/mainline-contract-test-index.md`（如 gate 映射变化）与 `openspec/changes/archive/INDEX.md`。

## 维护检查表

每次归档或切换 active change 后：

1. 运行 `openspec list --json`，以输出更新“进行中”条目。
2. 以 archive index 更新“已归档”条目；本文件只列最近一组，完整列表不复制。
3. 检查候选是否仍有未满足的触发信号；无触发信号的候选保持观察状态。
4. 涉及示例时只更新 `examples/agent-modes/MATRIX.md` 及对应 README，不恢复 numbered backlog。
5. 执行 `pwsh -File scripts/check-docs-consistency.ps1`，确保不存在 `roadmap-status-drift`。
