## Context

Baymax 当前运行时在 `Run` 路径已具备工具调用闭环（工具分发后通过 `ToolResult` 回灌下一轮 model step），但 `Stream` 路径仅透传 tool-call 事件并标记 `stream_tool_dispatch_not_supported`，没有形成真正的 ReAct 闭环。该差异会导致：
- Run/Stream 行为不等价，影响 contract 可回归性；
- provider tool-calling 行为收敛不足，出现语义漂移风险；
- 观测/回放/门禁无法稳定覆盖 ReAct 主链路。

A55 正在实施 observability export + diagnostics bundle。本提案 A56 需要在不改 A54/A55 范围的前提下完成 ReAct 主合同一次性冻结，并复用既有治理链路（runtime config、single-writer diagnostics、readiness/admission、replay、quality gate），避免后续再次拆分 ReAct 子提案。

## Goals / Non-Goals

**Goals:**
- 冻结 Runner-first ReAct contract，统一 `Run/Stream` 的工具闭环语义。
- 为 `Stream` 补齐工具分发与回灌，消除 `stream_tool_dispatch_not_supported` 中间态。
- 增加 run-level `tool_call_limit` 并与 iteration limit 协同，形成确定性终止语义。
- 收敛 OpenAI/Anthropic/Gemini 的 tool-calling 归一 contract，避免 provider-specific 漂移。
- 打通 config/readiness/admission/diagnostics/replay/gate 一体化治理，新增 `react.v1` fixture 与独立 gate。
- 补齐 sandbox/memory 与 ReAct 多轮工具路径的契约接缝，确保安全语义与可观测语义不漂移。

**Non-Goals:**
- 不引入平台化控制面（UI/RBAC/多租户运维面板）。
- 不改变 A54 memory contract 与 A55 observability contract 的既有边界。
- 不承诺 provider SDK 底层行为一致，仅要求 canonical 合同输出一致。

## Decisions

### Decision 1: 坚持 Runner-first ReAct 编排，不把循环下沉到 model adapter

- 方案：ReAct 主循环继续由 `core/runner` 状态机统一编排；model adapter 只负责 provider 协议映射。
- 备选：将 ReAct 循环移入 `model/*` 适配层。
- 取舍：Runner-first 可保持 `library-first + contract-first` 边界与单点可观测；下沉到 adapter 会放大 provider 语义分叉并削弱主线契约门禁。

### Decision 2: 为 Run/Stream 引入共享 ReAct loop 核心，避免双实现漂移

- 方案：抽象共享 loop 核心（iteration 决策、tool dispatch、feedback merge、termination），Run/Stream 仅保留输入输出形态差异。
- 备选：在 Run/Stream 中分别补丁实现。
- 取舍：共享核心初期重构成本更高，但长期可显著降低“一个修了另一个漏掉”的回归概率。

### Decision 3: Stream 工具闭环采用“分步收敛 + 每步回灌”策略

- 方案：Stream 每个 model step 期间持续透传增量事件，同时收集该步 tool calls；步结束后执行工具并回灌 `ToolResult` 进入下一步。
- 备选：tool call 出现即即时打断并派发工具。
- 取舍：分步收敛更容易保证事件顺序稳定、实现可回放；即时打断可降低单次等待但更易产生跨 provider 事件顺序漂移。

### Decision 4: 新增 run-level tool-call budget，形成双限流终止策略

- 方案：新增 `runtime.react.tool_call_limit`（run-level）与现有 `MaxToolCallsPerIteration`（iteration-level）协同，任一超限均 fail-fast。
- 备选：仅依赖 iteration-level 限制。
- 取舍：仅 iteration 限制无法防止“多轮低频工具调用”导致长尾循环；run-level budget 更符合 ReAct 风险控制。

### Decision 5: provider tool-calling contract 以 canonical request/response 归一输出为准

- 方案：model adapters 对 tool-call request、tool result feedback、错误分类统一映射到 `core/types` canonical 结构。
- 备选：保持 provider 特性字段直透到 runner。
- 取舍：直透短期灵活但破坏跨 provider 契约一致性；归一映射更利于 replay 和 gate 稳定。

### Decision 6: A56 观测与回放一次性收敛

- 方案：新增 react additive diagnostics 字段（loop counters、termination reason、stream parity markers），并新增 `react.v1` replay fixtures 与 drift 分类。
- 备选：先功能后观测。
- 取舍：先功能后观测会重复提案；一次性收敛可直接纳入 required-check 阻断。

### Decision 7: ReAct 前置能力通过 readiness/admission 前置阻断

- 方案：将 ReAct 所需能力（tool dispatch 可用性、provider tool-calling 能力、sandbox required 依赖）纳入 readiness finding 与 admission 决策，保持 deny side-effect-free。
- 备选：执行期遇错再中断。
- 取舍：执行期中断会造成行为不确定与排障成本上升；前置阻断更符合 contract-first 与 fail-fast。

### Decision 8: A56 覆盖范围按“一次性可落地”冻结，不预留 ReAct 后续拆案

- 方案：A56 直接覆盖 Runner loop、provider mapping、readiness/admission、sandbox loop semantics、diagnostics/replay/gate、docs/examples 映射。
- 备选：先交付 loop，再后续补 provider/observability/gate。
- 取舍：拆案会增加多轮迁移与语义漂移风险；一次性冻结可降低长期总成本并提升实施确定性。

## Risks / Trade-offs

- [Risk] Stream 引入工具闭环后事件时序更复杂，可能产生兼容回归。  
  -> Mitigation: 固化 step-boundary 事件顺序 contract，并增加 Run/Stream parity + replay drift 套件。

- [Risk] provider SDK 差异导致 tool-calling 映射边界不一致。  
  -> Mitigation: 以 `core/types` canonical contract 为唯一输出口径，adapter 侧补齐 taxonomy 映射测试。

- [Risk] ReAct loop 扩展可能增加延迟与资源消耗。  
  -> Mitigation: run-level budget + iteration budget + timeout 组合治理，并输出 p95 与 budget 命中观测字段。

- [Risk] A55 在研期间字段变更引发交叉冲突。  
  -> Mitigation: A56 只复用 A55 已冻结字段集合；未冻结字段走 additive fallback，不阻塞 A56 合同实现。

## Migration Plan

1. 抽象 ReAct loop 核心并接入 Run 路径（保持现有行为不变）。
2. 为 Stream 接入同一 loop 核心，补齐工具分发与回灌，移除中间态 reason。
3. 增加 run-level `tool_call_limit` 配置与 fail-fast/热更新回滚校验。
4. 收敛 OpenAI/Anthropic/Gemini tool-calling request/response 映射与错误分类。
5. 扩展 diagnostics + RuntimeRecorder react 字段，保持 bounded-cardinality 与 idempotency。
6. 新增 `react.v1` replay fixtures 与 drift 分类，验证 mixed-fixture backward compatibility。
7. 新增 `check-react-contract.sh/.ps1` 并接入 `check-quality-gate.*` + CI required-check 候选。
8. 同步 README、roadmap、runtime config 文档与主线索引，执行 docs consistency gate。

## Open Questions

- None for A56 scope. 本提案按“一次性完整覆盖 ReAct 搭建主合同”收敛，不预留 ReAct 主题后续拆分里程碑。
