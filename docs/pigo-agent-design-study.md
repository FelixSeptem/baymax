# 《用 Go 编写 Pi Agent》学习版笔记

> 来源：`C:\Users\whenh\Downloads\write-pi-agent-in-go.pdf`（174 页，主题为用 Go 重写 `pi` CLI Agent 的 `pigo` 设计与源码拆解）。
>
> 本文是面向阅读和复习的精简版，不是原文翻译，也不替代原 PDF。它保留设计动机、关键协议、错误边界和安全约束，省略了逐行源码讲解、重复代码和特定项目的目录细节。
>
> 阅读方法：每章先看“要解决的问题”，再看“核心机制”和“可迁移原则”。文中标注“PDF 明确设计”的内容是原文直接或近直接陈述；“对 Baymax 的启发”是结合当前仓库架构得出的建议，不代表 Baymax 已经完全实现该语义。

## 0. 全书主线

Agent 不是一次模型调用，而是一个受状态和边界约束的闭环：

```text
用户输入 / 会话历史
        │
        ▼
上下文组装 ──▶ LLM 流式响应
        ▲             │
        │             ├── 文本回复
        │             └── Tool Call
        │                    │
        └──── Tool Result ◀──┘
```

全书反复出现的设计原则有：

1. **单一真相源**：协议、配置、事件和状态各有明确 owner，避免多套“差不多”的模型。
2. **边界隔离**：Provider 协议差异、工具执行、会话持久化和安全策略都关在各自边界内。
3. **错误尽早暴露、运行期结构化**：能在构建/校验阶段发现的错误不要拖到运行期；运行期错误要保留分类、原因和可恢复信息。
4. **默认保守、明确放行**：权限、网络、文件写入和插件加载默认拒绝或降级，只有明确策略才放行。
5. **一切有界**：输入、响应、缓冲、重试、超时、文件大小、并发量和上下文长度都必须有上限。
6. **确定性和幂等**：排序、ID 关联、重放、恢复和重复请求的结果要可预测。
7. **局部故障隔离**：一个工具、技能、Provider 或子 Agent 失败，不应无条件拖垮整个运行时。

## 1. CLI 入口与运行装配

### 要解决的问题

CLI Agent 很容易在不同命令中重复初始化 Provider、工具、会话和配置，导致行为漂移、测试困难和资源泄漏。pigo 把入口设计成一条统一装配链：

```text
main → dispatch → setupAgentEnv → RunConfig → Agent loop
```

### 核心机制

- `main` 只负责进程入口和退出码。
- `dispatch` 根据子命令选择动作，不在每个分支中重复创建运行环境。
- `setupAgentEnv` 集中加载配置、解析 Provider、创建工具注册表、恢复会话并建立上下文。
- `RunConfig` 是测试和运行时的依赖注入边界；循环本身不应偷偷读取全局配置或环境变量。
- 资源关闭（HTTP client、文件、子进程、事件流）由装配层明确负责。

### PDF 明确设计

统一装配的价值是把“怎么运行”与“Agent 如何循环”分离。这样可以替换 Provider、工具或会话存储，而不改动循环状态机。

### 对 Baymax 的启发

Baymax 是 library-first 项目，不需要复制 CLI 的 `main` 结构；更值得借鉴的是统一的运行装配快照和依赖注入边界。配置解析、Provider 选择、工具策略、上下文组装和诊断 recorder 应该在进入 Run/Stream 前完成，并且 Run 与 Stream 使用同一套解析结果。

## 2. `agentcore`：稳定的底层协议

### 要解决的问题

如果消息、内容块、事件和工具调用类型散落在各个实现包中，Provider 和 Agent loop 会互相耦合。pigo 建立一个只被上层依赖的 `agentcore` 叶子包：

```text
Agent loop ─────┐
Provider ───────┼──▶ agentcore（消息、内容、事件、结果协议）
Tools ──────────┘
```

### 核心机制

- 使用未导出方法实现 **sealed interface**，限制核心协议的合法实现集合。
- 消息和内容块通过 `role`、`type` 等判别字段区分变体。
- 对复杂协议提供自定义 JSON 编解码，保证未知类型、缺失字段和不兼容版本能够明确失败。
- 协议包不依赖 Provider SDK、CLI 命令或具体工具实现。

### 为什么 sealed interface 有用

Go 的普通接口允许任意外部类型实现。如果某个协议要求“只能出现规定的消息变体”，普通接口容易让外部实现破坏不变量。未导出方法可以让包内保留实现权，同时仍能对外暴露只读能力。

### 对 Baymax 的启发

Baymax 的 `core/types` 已经承担了相似的协议投影职责，但不能把“有 `core/types`”直接等同为“已经实现 pigo 的 `agentcore`”。应逐项检查：协议对象是否有唯一 owner、未知变体如何处理、Provider 细节是否保持在 `model/<provider>`、以及外部扩展是否会绕过校验。

## 3. 两层 Agent loop

### 内层循环：完成一次模型—工具交互

内层循环处理一次模型响应及其工具调用：

1. 组装当前上下文和工具描述。
2. 建立 Provider 流并消费增量事件。
3. 累积文本、Tool Call 和 usage。
4. 如果没有 Tool Call，则得到普通回复。
5. 如果有 Tool Call，则校验参数、执行工具、收集结果。
6. 把工具结果按 call id 回填到下一次模型请求。

### 外层循环：决定是否继续运行

外层循环处理 follow-up、用户追加输入和终止条件：

```text
外层 run
 ├─ 内层模型/工具循环
 ├─ 得到 StopReason
 ├─ 需要 follow-up？──是──▶ 追加输入并再次进入内层
 └─ 否 ───────────────▶ 终态
```

### `StopReason` 驱动状态机

停止原因不是普通字符串，而是循环状态机的输入。典型情况包括：

- 模型完成文本回复；
- 工具执行后需要继续；
- 用户要求继续或补充信息；
- 达到迭代/工具调用预算；
- 被取消、超时或策略拒绝；
- Provider 或工具发生不可恢复错误。

明确的停止原因可以避免“某个布尔值决定是否继续”造成的隐式分支，也让诊断、重放和测试拥有稳定断言。

### 对 Baymax 的启发

Baymax 已有 ReAct loop、Run/Stream 和终止分类。后续审查应关注所有终止路径是否共享同一 taxonomy，是否保持 Run/Stream 语义等价，以及 follow-up、HITL、realtime interrupt 和 scheduler recovery 是否错误地引入了第二套终止语义。

## 4. Provider、双协议和 SSE 传输

### Provider 的双失败模型

PDF 将错误按“流是否已经建立”分为两类：

```text
构建/连接阶段失败
  → Go error

流已建立后失败
  → 流内终止事件（保留已消费的有效事实）
```

原因是：流建立前没有可交付的增量事实，调用方需要立即知道请求无法开始；流建立后可能已经产生了部分文本或工具调用，简单返回一个 error 会丢失这些事实。

### 重试边界

- 连接前的构建、鉴权或瞬时网络失败可以按策略重试。
- 已经消费过响应内容的流不应盲目重试，否则可能重复工具调用或重复输出。
- `NewRequest` 工厂应能在每次重试时重新构造 request body，不能复用已读尽的 body。
- idle timeout（长时间没有任何事件）和 stall timeout（流未结束但进度停滞）应分别治理。

### 双协议解码

OpenAI 和 Anthropic 的流格式不同。pigo 使用有状态解码器把各自的 SSE/JSON 事件汇聚到统一事件模型：

```text
OpenAI SSE ───────┐
                  ├── Provider Event ──▶ Agent loop
Anthropic SSE ────┘
```

解码器必须记住当前内容块、tool call、usage 和终止原因，不能仅靠每个 SSE 行独立解析。

### Provider 注册和 preset

Provider 名称、preset、鉴权方式和协议类型应分离：同一 Provider 可能有多个模型/preset；鉴权来源也不应硬编码在协议解码器里。

### 对 Baymax 的启发

Baymax 已有多 Provider 适配和错误分类，但应继续确认：连接失败与流中失败是否可观察地区分，重试是否只发生在未消费流，partial output 是否保留，以及 idle/stall 是否有独立上限。不要把 pigo 的错误策略无条件推广到 scheduler、admission 或 recovery 等非 Provider 运行域。

## 5. 工具系统：从 Schema 到安全执行

### 注册阶段

- 工具名称需要稳定、可归属到 namespace。
- 注册时校验并编译 JSON Schema，尽早发现格式错误。
- 工具描述和 Schema 是发送给模型的协议，不应由具体执行器临时拼接。
- 重名注册必须失败或按明确的优先级处理，不能静默覆盖。

### 执行阶段

PDF 给出的生命周期可以概括为：

```text
prepare
  → validate
    → authorize / before
      → execute
        → after / finalize
```

各阶段职责不同：

- `prepare`：解析调用、补齐运行上下文和资源预算。
- `validate`：Schema、大小、类型和业务参数校验。
- `authorize/before`：策略、信任、sandbox 和人工审批闸门。
- `execute`：实际副作用或只读操作。
- `after/finalize`：记录结果、释放资源、生成 diff/provenance。

### 错误和 panic

工具错误、参数错误、权限拒绝、sandbox 启动失败、超时和 panic 都应该被隔离并转换成可关联的结构化结果；但“结构化”不等于“成功”。结果必须带有状态、错误分类、原因和是否可重试等信息。

### 并行调用

当模型一次返回多个工具调用时，执行器可以按策略串行或并行：

- 只读、互不依赖的调用可以并行；
- 有副作用或存在依赖时应串行；
- 并行执行完成后，结果仍按原始调用顺序回填；
- 每个结果必须通过 call id 关联，不能依赖完成先后顺序。

### 文件和 Web 工具

PDF 特别强调：

- 文件路径必须位于 workspace boundary 内；
- 读取和输出有大小上限，超限时截断并明确提示；
- 写入前检查目标唯一性、冲突和 diff；
- WebFetch 只允许 HTTPS，阻断危险的跨域重定向，并设置响应大小上限。

### 对 Baymax 的启发

Baymax 已有 dispatcher、JSON Schema、middleware、sandbox、allowlist 和 policy。当前更适合做“生命周期合同审计”，而不是新增一套工具系统：确认 prepare/validate/authorize/execute/finalize 是否在所有路径存在，panic/取消/超时是否统一收口，并验证并行结果顺序和 Run/Stream 对等。

## 6. 上下文压缩

### 为什么不能只截断

简单保留最近 N 条消息会丢失任务目标、文件变更、工具结果和未完成事项。pigo 把压缩视为“重写上下文但保留运行事实”的操作。

### 核心约束

- 以 token 预算和保留区为锚点，而不是只用字符数。
- system/developer 等受保护内容不能被普通压缩删除。
- 在合法消息边界切分，不能截断工具调用结构。
- 保留最近上下文，同时生成结构化摘要。
- 摘要应包含任务目标、已完成工作、未完成事项、关键决策、文件操作和工具结果。
- 压缩失败应回退到安全的截断/原上下文路径，不影响主 run。
- 压缩本身也必须有时间、大小和重试上限。

### 结构化摘要的意义

摘要不是给人看的日志，而是下一轮模型可执行的“交接单”。它应区分事实与推断，并尽量保留可验证引用，例如文件路径、工具 call id、artifact 或 checkpoint digest。

### 对 Baymax 的启发

Baymax 已有 truncate/semantic compaction、质量评分、embedding/reranker 和 fallback，这比 pigo 的最小实现更丰富。可借鉴的重点是检查压缩产物是否显式保留运行交接事实，而不是照搬某个摘要模板或粗略的 token 换算公式。

## 7. 会话持久化与历史树

### JSONL 和 parent tree

pigo 使用追加式 JSONL 保存会话 entry。每个 entry 指向 parent，当前 leaf 决定当前对话上下文：

```text
root
 ├── user-1
 │    └── assistant-1
 │          ├── tool-branch-a
 │          └── tool-branch-b
```

这种结构支持：

- 从任意历史点 fork/branch；
- 只读取某个 leaf 的祖先链作为上下文；
- 在不覆盖原历史的情况下尝试另一条路径；
- 用追加写降低损坏风险。

### 原子性和迁移

- 写入应使用临时文件、flush/fsync、rename 等原子替换策略（具体实现依宿主而定）。
- entry schema 必须带版本，旧版本需要显式迁移或拒绝。
- 读取损坏记录时要保留可诊断信息，不应静默修复成错误历史。

### HTML 回放

回放页面应是离线、无脚本、无外部资源的静态输出；所有消息和属性全文转义，防止会话内容变成 HTML/脚本注入点。

### 对 Baymax 的启发

Baymax 已有 checkpoint lineage、branch/replay reference 和 snapshot import/export，但这些概念不自动等同于消息级 JSONL session tree。后续若需要借鉴，应先明确“消息历史”“运行 checkpoint”“诊断回放”三者的 owner 和关联，而不是用一个存储结构同时承担所有职责。

## 8. 项目信任与安全闸门

### 三态信任

PDF 使用三态 trust：

```text
Undecided ──明确确认──▶ Trusted
    │                       │
    └────拒绝/不可信──────▶ Untrusted
```

- `Undecided`：没有明确授权，不应执行高风险副作用。
- `Trusted`：在定义范围内允许执行，但仍受工具和 sandbox 策略限制。
- `Untrusted`：默认拒绝或只允许只读安全操作。

信任可以按就近目录继承，但 session trust（本次会话决定）和持久 trust（跨会话保存）要分开，避免一次临时批准永久扩大权限。

### BeforeToolCall 闸门

副作用工具在执行前经过统一闸门，闸门可以：

- 允许；
- 拒绝并给出原因；
- 要求人工确认；
- 降级为只读或 dry-run。

安全决策不能只写进 prompt；必须由运行时强制执行并记录。

### 对 Baymax 的启发

Baymax 已有 policy precedence、sandbox、allowlist、action gate 和 security diagnostics。pigo 的目录 trust 继承是可研究的补充，不应直接替换现有策略栈，也不能假定当前 action gate 已经等价于 pigo 的 `BeforeToolCall`。

## 9. 子 Agent 与进程隔离

### 两种隔离级别

- **goroutine 隔离**：启动成本低、共享内存方便，适合可信且轻量的协作者；故障隔离主要靠 context、recover 和协议边界。
- **process 隔离**：地址空间、资源和权限边界更强，适合不可信扩展或需要独立生命周期的 Agent。

### stdio JSON-RPC

子进程可以通过 stdin/stdout 传输 JSON-RPC：

```text
父 Agent ──request(id)──▶ 子进程
父 Agent ◀─response(id)── 子进程
```

实现要点：

- pending map 以 request id 关联并发请求；
- 读循环遇到 EOF 或进程崩溃时，统一 failAll 未完成请求；
- 关闭时先发送优雅停止信号，超时后再强杀；
- 所有请求、响应、超时和退出原因都可观测；
- 子 Agent 故障不应让父 Agent 的其他任务失去终态。

### 对 Baymax 的启发

Baymax 当前的 teams/mailbox/workflow/orchestration 是多代理协调基础，但不等于已有 pigo 的跨进程 Agent isolation。若未来跨进程，建议沿用 Baymax 的 reference/digest/bounded metadata 边界，另行定义 transport、权限、恢复和版本合同，不直接复制 `os.Executable()` 自启动模型。

## 10. Skills、Plugins 与包管理

### 三层扩展模型

```text
Skill：提示词、触发条件、工具声明
   ↓
Plugin：更完整的代码/资源扩展
   ↓
Package Manager：安装、版本、锁定、更新和回滚
```

Skill 适合表达领域知识和工具使用约束；Plugin 可以带实现；包管理器负责可重复安装和供应链安全。

### 故障隔离与供应链约束

- 一个坏 Skill/Plugin 不应拖垮整个 Agent。
- 加载前校验 manifest、能力、版本和路径。
- 安装尽量禁用任意安装脚本（PDF 以 `npm pack --ignore-scripts` 为例）。
- 使用 lockfile 或等价机制保证版本可复现。
- 扩展加载、执行和卸载都应有超时和资源上限。

### 对 Baymax 的启发

Baymax 已有 `skill/loader` 的 Discover/Compile、触发评分、预算、SkillBundle 和 tool whitelist，但当前证据不足以称为完整 Plugin/Package Manager 生态。若未来需要扩展治理，应先定义 manifest、capability、digest、兼容性、隔离和回滚合同，不直接引入 npm 作为运行时依赖。

## 11. 后记：边界与后续方向

PDF 最终强调，工程质量不只取决于“能否调用模型”，还取决于：

- 失败是否可解释、可恢复；
- 数据和工具边界是否明确；
- 长任务是否可压缩、可持久化和可回放；
- 扩展是否被隔离；
- 协议是否能在 Provider 和传输变化时保持稳定。

它没有要求所有 Agent 都采用同一种 loop、存储或拓扑。相反，值得稳定的是外部边界和生命周期：请求、事件、工具调用、结果、取消、重试、恢复和权限决策之间的关联。

## 12. 一页式复习清单

读完 PDF 后，可以用下面的问题检查一个 Agent runtime：

### 协议

- 核心消息/事件是否有唯一事实源？
- 未知变体和版本不兼容是否 fail-fast？
- Provider 差异是否被关在适配层？

### 循环与终态

- 内层工具循环和外层 follow-up 是否职责清楚？
- StopReason/terminal outcome 是否有稳定 taxonomy？
- Stream 断开、观察者退出或取消后，是否仍能得到权威终态？

### 工具与安全

- Schema 是否在注册或调用前校验？
- authorize、execute、finalize 是否可观测？
- panic、超时、拒绝和运行错误是否区分？
- 文件、网络、并发和响应大小是否有界？

### 上下文与持久化

- 压缩是否保留任务交接事实？
- 会话历史、checkpoint 和诊断回放是否职责分离？
- branch、replay、restore 是否幂等且版本化？

### 扩展与隔离

- Skill、Plugin 和包版本是否可复现？
- 坏扩展是否局部失败？
- 跨进程边界是否只传有限、可验证的数据？

## 13. 与 Baymax 的关系：不要混淆的三层结论

| 证据层级 | 可以说什么 | 不能说什么 |
| --- | --- | --- |
| PDF 明确设计 | pigo 采用了双层 loop、sealed interface、SSE 状态解码、JSONL parent tree 等 | Baymax 必须复制这些具体 API/目录/存储 |
| Baymax 相邻基础设施 | Baymax 有 `core/types`、Run/Stream、dispatcher、compaction、snapshot、policy、skill loader 等构件 | Baymax 已经具备 pigo 的完整等价语义 |
| Baymax 适配建议 | 可以审计 failure taxonomy、终态、tool lifecycle、压缩交接和扩展治理 | 这些建议都是 PDF 原文结论，或已经被 roadmap 承诺实施 |

## 14. 建议的学习顺序

1. 先读本笔记第 0、3、4、5 章，理解 Agent loop、Provider 和工具的主运行路径。
2. 再读第 6、7、8 章，理解长任务、恢复和安全边界。
3. 最后读第 9、10 章，评估多 Agent 和扩展生态是否值得引入。
4. 回到第 12 章清单，对照 Baymax 的具体 contract、replay fixture 和 gate，而不是只看目录名称。
