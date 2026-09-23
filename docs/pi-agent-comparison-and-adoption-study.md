# Baymax 与 Pi Agent 对比及吸收建议

更新时间：2026-09-10

## 1. 研究范围与结论

本文比较 Baymax 当前主线与 `earendil-works/pi`，用于筛选后续 OpenSpec 提案方向，不构成排期或兼容性承诺。

Pi 研究基线：

- 仓库：<https://github.com/earendil-works/pi>
- DeepWiki 索引时间：2026-09-08
- 索引提交：[`96617628`](https://github.com/earendil-works/pi/commit/96617628)
- 重点源码：`packages/agent`、`packages/ai`、`packages/coding-agent`、`packages/protocol`、`packages/server`、`packages/client`、`packages/session-backends`

Baymax 研究基线：

- 活跃提案以 `openspec list --json` 为准；调研时无 active change。
- 已交付能力以 `openspec/specs/`、`openspec/changes/archive/INDEX.md` 和 `docs/mainline-contract-test-index.md` 为准。
- 关键实现入口是 `core/runner`、`core/types`、`orchestration/composer`、`runtime/config`、`runtime/diagnostics` 与 `observability/event`。

2026-09-12 校准：当前实施方向是跨 Provider handoff 与 stream edge conformance。Provider-native 投影仍由 `model/openai|anthropic|gemini` 各自拥有，Runner 保持 step/terminal/fallback fence 所有权；不吸收远程 model catalog、credential store、全局 router、hosted gateway 或 Pi 的 provider-neutral wire abstraction。该方向通过 `provider_handoff_stream_edge.v1` 离线回放及双平台 gate 回滚，无配置或数据迁移。

核心结论：

1. 两个项目不是同一种产品。Pi 首先是可直接使用的终端 coding agent 与 SDK；Baymax 首先是 `library-first + contract-first` 的 Go runtime。
2. Baymax 在协议投影、Run/Stream 等价、失败分类、策略/沙箱、诊断回放和质量门禁上更系统；Pi 在交互产品、宿主控制、运行中输入、Session 使用体验和扩展开发体验上更完整。
3. Pi 最值得吸收的是“同一 Agent 内核之上的多种宿主适配模式”，尤其是严格 JSONL RPC、命令/异步事件分离、反向 UI 请求、pending request 收口和输出通道保护。
4. Baymax 不应复制 Pi 的 AgentSession、lane/register/ledger 或实验性远程 Session Server；这些做法会与现有 Runner、Session history、checkpoint、snapshot、scheduler 和 diagnostics owner 重叠。
5. 下一项优先候选应补齐“运行中的嵌入式宿主接缝”，而不是再增加一个投影类型或直接建设远程网关。

## 2. 定位与架构比较

| 维度 | Pi Agent | Baymax | 比较结论 |
| --- | --- | --- | --- |
| 产品定位 | 面向开发者的终端 coding agent、SDK 和可扩展应用 | 可嵌入的 Go agent runtime 与治理合同 | Pi 更贴近最终用户；Baymax 更适合作为受控运行内核。 |
| 主要语言与分发 | TypeScript/Node.js monorepo、CLI/TUI/npm packages | Go modules、library-first packages、examples 与 contract gates | 不迁移语言或包结构，只吸收边界设计。 |
| 核心执行 | `Agent`/agent loop，由 `AgentSession` 组合会话和产品能力 | `core/runner.Engine` 提供 Run/Stream，Composer 组合 readiness 与 orchestration | 两者都应保持单一执行内核；Baymax 不需要第二个 Agent loop。 |
| 运行模式 | Interactive、print、JSON events、RPC 共用同一 Session | 主要由调用方直接调用 Run/Stream，并传入 EventHandler | Baymax 的明显缺口是标准宿主适配模式，而非执行能力。 |
| 状态模型 | Agent state、Session tree；新 Harness 使用 entries/registers/usage ledger 与 lanes | Run/Session 协议投影、history/checkpoint、snapshot、scheduler/mailbox、diagnostics 分域持有 | Pi 可作为 owner 审计参考，但不可整体移植。 |
| 治理方式 | 单测、集成测试、provider E2E、behavioral eval | OpenSpec、单测/integration、replay fixture、shell/PowerShell gate、文档一致性 | Baymax 治理更强；Pi 的真实使用路径测试值得补充。 |

## 3. 核心执行与生命周期

### 3.1 已经对齐的部分

Pi 与 Baymax 都将模型调用、工具执行和下一轮决策维持在一个核心循环内。Baymax 的 `core/types.Runner` 已定义 Run/Stream 对等入口，`core/runner.Engine` 负责主循环、工具闭环和终态输出；Agent Runtime Protocol 只做引用与生命周期投影，不取代 source runtime。

Baymax 已具备 Pi 中下列基础能力的同域实现：

- 多轮 model/tool loop 与并行工具执行。
- provider abstraction、工具调用规范化和错误分类。
- context assembly、compression、handoff、checkpoint 和 replay。
- lifecycle hooks、tool middleware、skills 和 extension lifecycle。
- cancel/timeout 的 context 传播以及规范化 terminal outcome。

因此，不建议以“借鉴 Pi”为理由重写 Runner、增加 `AgentSession` facade，或创建平行终态状态机。

### 3.2 Baymax 的实际缺口

`core/runner.Engine` 当前没有按 `run_id` 暴露 active Run registry 或统一 Run control handle。调用者通常只能取消最初传入的 `context.Context`。Agent Runtime Protocol 虽声明 `cancel|resume|retry` action，但 `ValidateProtocolAction` 只验证 action availability，不执行 source Runtime 操作。

Realtime interrupt/resume 也主要通过 `RunRequest.Realtime.Events` 在 Run/Stream 入口批量摄入。它已经验证 envelope、sequence、dedupe 和 cursor，却不是一个宿主可在 Run 进行中持续写入的 command ingress。

这决定了宿主方向不能只实现 JSONL framing。必须先明确：

- 谁持有 active Run 的取消函数和生命周期快照。
- command admission response 与业务 terminal outcome 如何分离。
- 中途 interrupt、HITL response 和未来 steering 如何进入 source Runtime。
- late/duplicate command 如何保持幂等且不覆盖首个终态。

## 4. 宿主模式、RPC 与事件流

Pi 的成熟 RPC mode 是当前最直接的参考对象：

- 以 LF 为唯一分隔符的严格 JSONL over stdin/stdout。
- command 可携带 `id`，response 回显该标识。
- command response 与异步 Agent/Session event 使用不同 envelope。
- `RpcClient` 用 pending map 关联响应，并在子进程退出时拒绝所有未完成请求。
- extension 的 `select|confirm|input` 被投影成反向 UI request，等待宿主返回对应 response。
- stdout 仅用于协议；普通日志重定向 stderr。
- raw stdout 写入显式等待 backpressure，避免协议帧丢失或交错。
- 行解析不使用会错误识别 `U+2028/U+2029` 的通用 readline 分隔行为。

Baymax 已有可复用基线：Realtime envelope、Agent Runtime Protocol Event、durable stream binding、terminal recovery、HITL RequestID、policy/readiness/sandbox 和 RuntimeRecorder。缺的是把这些 owner 连接成可嵌入宿主协议的 adapter。

建议的吸收原则：

1. transport-neutral contract 先于 JSONL binding。
2. Host adapter 只负责 correlation、framing 和 delivery，不拥有 Run、cursor、terminal 或 Session 状态。
3. 命令“已接收/被拒绝”与命令触发的业务结果必须分开。
4. 宿主断开只影响观察和 pending request，不得默认取消业务 Run。
5. 传输 backpressure 不得通过现有无返回值的 `EventHandler.OnEvent` 隐式阻塞或改变 Runner 状态。

## 5. HITL、Steering 与 Follow-up

Pi 将 extension UI 调用桥接到 RPC 宿主，并区分两类运行中输入：steering 影响当前运行的下一步，follow-up 在当前运行空闲后排队执行。

Baymax 已有 `ClarificationResolver` 和 `ActionGateResolver`，Runner 会发出 clarification/timeline 事件并同步等待 resolver。因此，第一步应把现有 HITL 语义适配为 host-mediated request/response，而不是创建新 HITL 状态。

Steering/follow-up 已由独立 OpenSpec 提案收口为 source-owned bounded lanes：steering 在既有 safe point 应用，follow-up 在 idle/terminal 边界通过现有 Run/Stream path 晋升为 distinct causal Run，并明确 cancel/terminal/disconnect 优先级与 parity。该实现不复制 Pi 的 AgentSession、lane/register/ledger 或远程 Session Server。

## 6. Session、持久化与 Harness

Pi 的传统 Session 使用 append-only JSONL tree 支持 branch/fork/compaction；新 AgentHarness 将 durable state 分为 append-only entries、mutable registers 和 usage ledger，并以 lane 表示 conversation tree 上的命名 cursor，每个 lane 同时最多运行一个 operation。

Baymax 已经把相关责任拆给多个 owner：

- Session history/checkpoint replay 持有历史关联、branch lineage 和 restore/replay 幂等。
- Unified snapshot 只聚合 source-owned segments，不重写模块存储。
- Context handoff 引用既有 artifact/checkpoint/history/snapshot owner。
- Scheduler、mailbox 和 recovery store 持有排队、租约、异步恢复等编排事实。
- Runtime diagnostics 只保存有界观测记录，不是 Session 或 Run 事实源。

Pi 的 entries/registers/ledger/lane 适合用作“状态所有权检查表”，但不适合作为 Baymax 新公共模型。只有当真实场景证明现有 checkpoint 无法恢复进行中的 operation，才考虑增量的 durable-operation contract，并且必须引用现有 Session/checkpoint owner。

## 7. Provider、模型与凭据

Pi 通过统一 AI package、Models runtime、动态 model discovery、CredentialStore、认证命令和 llama.cpp router 提供完整的终端产品体验，并对跨 provider handoff、tool-call ID、thinking blocks、context overflow、abort token usage、Unicode 和空消息做专项测试。

Baymax 已有 provider adapters、静态/宿主注入 catalog、capability negotiation、credential preflight、fallback、readiness admission 和统一错误分类。近期不应引入 credential store，也不应把 provider SDK 移入 `context/*`。

最值得吸收的是测试与准入方法：

- 跨 provider 的 message/tool-call/thinking block handoff conformance。
- stream abort 后 partial facts 与 token usage 的兼容矩阵。
- context overflow、空内容、Unicode 和孤立 tool call 的边界 fixture。
- 当静态目录不足时，再评估本地模型路由或远程 catalog；凭据仍由宿主提供。

响应侧（Provider → Runtime）的上述方法已由归档 `harden-cross-provider-handoff-and-stream-edge-conformance` 落地为 `provider_handoff_stream_edge.v1` fixture、离线 replay 与双平台 gate。

请求侧（Runtime → Provider）此前没有对应的合同；该缺口已由归档 142 `establish-provider-request-projection-and-cache-observability-contract` 固化为审计基线，并由归档 144 `preserve-provider-native-request-projection-parity` 修复已验证的 native request projection drift。其当前基线（可复现，非推测）为：

- `core/runner` 唯一构造 `types.ModelRequest`，把 `Input`、`Messages`、`ToolResult`、`Capabilities` 原样转发；语义丢失发生在适配器，不在 runner。
- 归档 142 的审计曾证明三个适配器会把请求压平为单段文本；归档 144 已将该 declared gap 迁移为各官方 SDK 可表达的 native role、稳定顺序、assistant history、tool-result correlation 与 capability projection。`toolcontract` 仍只提供 SDK-neutral canonical facts，不成为新的共享 wire protocol。
- 同一请求在 `Generate`、`Stream` 与可用的 `CountTokens` 路径下保持 canonical request facts 和 Run/Stream parity；各 provider 的 SDK capability exception 必须显式记录。
- 已知边界（归档 144 已按 provider 能力显式收口）：`CountTokens` 与 `Generate`/`Stream` 采用同一组 canonical request facts；OpenAI 官方 SDK 无 token-count API 的能力例外仍被显式记录，不伪造 cache 或 token 结果。
- `TokenUsage` 无 cache 字段，cache 会计只能按 additive + nullable + default 演进；当前适配器无 cache 来源，投影固定 `available=false` 且不伪造 read/write 值。

这些事实先被归一化为 `provider_request_projection.v1`：`source`/`observed` 双投影、canonical digest、被钉住的 `declared_gap`（缺口只能被显式修复，不能被静默引入或静默修复），以及离线 replay 与双平台 gate；归档 144 已在同一合同中显式迁移已修复 gap，后续只观察 cache usage 的真实成本/P95 证据。

## 8. Tool、Extension 与安全

Pi 的 extension 系统对终端用户很灵活：动态加载 TypeScript，注册 tools、commands、providers、UI 和 lifecycle hooks，并配有 extension authoring eval。

Baymax 已有更严格的 tool lifecycle、middleware、extension discovery/lifecycle/resource resolution、manifest/capability、allowlist、sandbox 和 failure isolation。两者目标不同：Pi 优先本地可塑性，Baymax 优先可审计准入。

可借鉴方向是增强外部扩展的开发者体验与 conformance：最小 fixture、authoring eval、失败隔离场景和宿主 UI bridge。不可借鉴的是无准入地动态执行任意扩展、建设 package manager/market，或绕过现有 policy/sandbox owner。

## 9. 可观测性、测试与评测

Pi 的测试覆盖 unit、AgentSession integration、provider E2E 和 model-backed eval，并保存 session snapshot、transcript 和生成源码 artifact，支持 baseline/candidate 对比。

Baymax 已有 OTel、RuntimeRecorder、QueryRuns、diagnostics replay、corpus/badcase/experiment 和大量 contract gates。Baymax 的优势是确定性漂移阻断；Pi 的优势是围绕真实 CLI/Session workflow 的产品路径验证。

可吸收的增量包括：

- 为宿主协议建立 subprocess conformance harness，验证 stdout purity、framing、退出码和 pending 收口。
- 将现有 eval run 与 transcript/artifact/checkpoint 引用关联，避免复制内容或建立新的 artifact store。
- 对 provider edge cases 建立小型跨后端矩阵，而不是只扩大通用集成测试。

## 10. Pi 实验性远程协议的处理结论

Pi 的 `packages/protocol`、`packages/server` 和 `packages/client` 当前明确标为 experimental，采用 versioned ClientHello、length-prefixed CBOR、Request/Cancel/Response/Event/Attachment envelopes、session target fencing、attachment 和 lease。

其中可以作为未来合同设计检查项的只有：

- 显式 protocol version negotiation。
- frame size limit 和 malformed frame fail-fast。
- Request/Cancel/Response/Event 分离。
- server/session/attachment target fencing。
- connection loss 时 pending operation 的确定性收口。

当前明确不吸收：CBOR wire format、Unix socket server、remote Session host、attachment/lease ownership、SQLite hosted backend 和多客户端 Session 服务。这些内容既不稳定，也属于 Baymax roadmap 明确延后的 Remote Runtime Gateway/hosted persistence 范围。

## 11. 后续提案备选结论

| 成熟度 | 候选方向 | Pi 启发 | Baymax 增量边界 | 启动条件 |
| --- | --- | --- | --- | --- |
| 首选候选 | 嵌入式宿主事件与请求响应接缝 | RPC correlation、异步 event、pending map、output guard | 补 source-owned active Run control 和 transport-neutral adapter；首个 binding 可为 strict JSONL | IDE、桌面宿主、headless UI 或外部 HITL 客户端出现明确需求。 |
| 与首选合并优先 | 宿主介导 HITL adapter | extension UI reverse request | 只适配现有 clarification/action gate RequestID 和 timeout，不建新 HITL state | 首个宿主需要跨进程确认、选择或输入。 |
| 条件候选 | 运行中 steering/follow-up 输入语义 | steer 与 follow-up 分离、queue regression | 新增有界输入 ingress 和明确时序；不修改历史消息 owner | 宿主确实需要在 active Run 中追加或排队用户输入。 |
| 条件候选 | 跨 Provider handoff 与 stream edge conformance | cross-provider、abort usage、overflow、Unicode fixtures | 优先扩展测试/fixture/gate，复用现有 provider admission 和 error taxonomy | 已归档为 `harden-cross-provider-handoff-and-stream-edge-conformance`，触发条件已满足。 |
| 已归档 | Provider 请求侧结构化投影与 Prompt Cache 可观测性 | 统一 provider projection、cache/usage 字段兼容 | 归档 142 建立 `source`/`observed` 双投影、canonical digest、declared gap 与离线 replay/gate；归档 144 修复已验证的 native role/tool-result/ordering parity；cache schema 仍不伪造 | 已归档为 142 + 144；后续只有 cache 成本/P95 与 provider usage 证据成立时才另行立项。 |
| 条件候选 | 外部 Extension authoring conformance | extension authoring eval、真实 workflow 测试 | 复用 lifecycle/manifest/capability/allowlist/sandbox，改善开发者反馈 | 出现新的外部扩展来源或集成方。 |
| 观察候选 | Model catalog 与本地模型路由增量 | Models runtime、动态 discovery、llama.cpp router | catalog 仍由 source/host 提供，不引入 credential store | 静态或宿主注入 catalog 无法满足明确路由需求。 |
| 观察候选 | Eval transcript/artifact comparison | eval harness、snapshot/transcript artifact | 只增加引用和 baseline/candidate 对比，不建 artifact service | 现有 corpus/eval 无法定位可复现质量回归。 |
| 审计项 | Durable operation 与状态所有权审计 | Harness entries/registers/ledger/lanes | 先审计现有 Session/checkpoint/snapshot/scheduler owner，不预设新模型 | checkpoint 无法恢复真实进行中 operation 时才转提案。 |

提案合并规则：

- active Run control、JSONL framing、output integrity、event backpressure 和基础 HITL bridge 优先作为同一宿主接缝的内聚设计，不拆成互相循环依赖的提案。
- steering/follow-up 必须独立于基础宿主接缝，避免第一项提案同时引入新输入队列语义。
- Provider、Extension、Eval 方向只能在对应触发信号出现时启动，并复用各自已归档 contract。
- Durable operation 在发现 owner 缺口前保持审计项，不以抽象完整性为理由立项。

## 12. 参考入口

Pi：

- [Overview](https://deepwiki.com/earendil-works/pi/1-overview)
- [Agent loop and AgentHarness](https://deepwiki.com/earendil-works/pi/2.1-agent-loop-and-agentharness-(pi-agent-core))
- [AgentSession lifecycle](https://deepwiki.com/earendil-works/pi/2.2-agentsession-and-session-lifecycle)
- [Session management and compaction](https://deepwiki.com/earendil-works/pi/2.3-session-management-and-compaction)
- [Print, JSON and RPC modes](https://deepwiki.com/earendil-works/pi/5.4-alternative-execution-modes:-print-json-and-rpc)
- [Extension lifecycle](https://deepwiki.com/earendil-works/pi/6.1-extension-api-and-lifecycle-events)
- [Experimental session protocol](https://deepwiki.com/earendil-works/pi/7.3-session-server-client-and-protocol-(experimental))
- [Testing infrastructure](https://deepwiki.com/earendil-works/pi/10.3-testing-infrastructure)
- [RPC source documentation](https://github.com/earendil-works/pi/blob/96617628/packages/coding-agent/docs/rpc.md)

Baymax：

- `docs/runtime-harness-architecture.md`
- `docs/runtime-module-boundaries.md`
- `docs/runtime-config-diagnostics.md`
- `docs/mainline-contract-test-index.md`
- `openspec/specs/agent-runtime-protocol-contract/spec.md`
- `openspec/specs/realtime-event-protocol-and-interrupt-resume-contract/spec.md`
- `openspec/specs/durable-runtime-event-stream-binding/spec.md`
- `openspec/specs/runtime-event-stream-terminal-recovery/spec.md`
- `openspec/specs/session-history-checkpoint-replay/spec.md`
- `openspec/specs/extension-lifecycle-governance/spec.md`

## 13. 研究限制与复核规则

- Pi 结论基于上述固定提交与 DeepWiki 索引；其后续行为变化必须重新核对源码，不能把本文当作 Pi 永久合同。
- DeepWiki 用于导航和交叉验证，关键协议判断应回到固定提交下的源码与官方文档。
- 本文只筛选可借鉴方向，不证明 Baymax 已出现对应产品需求。
- 每个候选启动前仍必须执行现状审计、OpenSpec proposal/design/spec/tasks、Example Impact Assessment、正负边界测试、必要 integration、replay 和 gate。
