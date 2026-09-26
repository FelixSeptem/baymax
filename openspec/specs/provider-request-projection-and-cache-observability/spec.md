# provider-request-projection-and-cache-observability Specification

## Purpose
本能力定义一套版本化、有界、可离线回放的**请求侧**投影契约：把 `ModelRequest`（`Input`、`Messages`、`ToolResult`、`Capabilities`）到 provider SDK request 的投影归一化为可比较的 provider-neutral 事实，从而让 role 语义、part 顺序、tool-result 归属、工具顺序、稳定前缀与 usage/cache 可观测性的丢失或漂移成为可被 fixture 捕获、可被 gate 阻断的显式证据，而不是适配器内部的隐式行为。

## Requirements

### Requirement: 请求侧投影 SHALL 由版本化有界契约表达

系统 MUST 提供版本化、provider-neutral 的请求投影契约 `provider_request_projection.v1`，用于表达一次模型调用的请求侧事实。契约 MUST 同时包含 `source`（runtime 送入 `ModelRequest` 的 provider-neutral 事实）与 `observed`（实际到达 provider SDK 边界的归一化事实），并 MUST 为二者提供确定性 canonical 序列化与 digest。

契约 MUST NOT 引入 provider SDK 依赖，MUST NOT 记录 raw prompt、raw reasoning、完整消息正文、credentials 或任何无界 payload；文本类事实 MUST 只以 digest 与长度表达。

#### Scenario: 契约对同一输入产生稳定投影
- **WHEN** 同一 `ModelRequest` 被同一适配器投影两次
- **THEN** canonical 序列化与 digest 完全一致，且不产生任何副作用

#### Scenario: 契约超界时确定性失败
- **WHEN** cases、parts、roles、tools 或单个 digest 超出契约边界
- **THEN** 契约以稳定的 overflow 分类 fail-fast，且不返回截断后的部分成功结果

#### Scenario: 未知版本被拒绝
- **WHEN** 解析到的 fixture 版本不是 `provider_request_projection.v1`
- **THEN** 解析失败并返回 schema drift 分类，不尝试兼容未知版本

### Requirement: 投影缺口 SHALL 被显式申报且不可静默漂移

契约 MUST 允许每个用例声明 `declared_gap`，并 MUST 依据 `source` 与 `observed` 计算出确定性 gap 分类。系统 MUST 在下列任一情况下失败：

- 未声明缺口但计算出缺口（未申报的语义漂移）；
- 声明缺口但计算不出缺口（投影行为已改变，需显式更新契约）。

`declared_gap` MUST 被当作待修复项的证据锚点；契约 MUST NOT 为它引入任何「容忍」「允许降级」或「等价语义」的解释。

#### Scenario: 未申报的角色投影丢失被发现
- **WHEN** `source` 声明了 system/assistant 角色而 `observed` 只包含 user 角色，且用例未声明对应缺口
- **THEN** 校验失败并返回 role projection drift 分类

#### Scenario: 投影行为改变必须显式更新契约
- **WHEN** 某次改动使此前被申报的缺口不再成立
- **THEN** 校验失败并返回 contract drift 分类，要求显式更新契约与 fixture

### Requirement: Adapter 请求投影 SHALL 可审计且不得泄漏 provider-only 语义

supported adapter（OpenAI、Anthropic、Gemini）的请求投影 MUST 可被同一 conformance 家族通过既有注入缝隙观测，且 MUST NOT 需要新增运行时 API 才能审计。观测到的投影 MUST 归一化为 provider-neutral 事实，provider-only 字段 MUST NOT 成为必需语义，MUST NOT 进入 runtime 契约或诊断输出。

同一 `ModelRequest` 的 Run 与 Stream 投影 MUST 语义等价，仅允许由调用形状决定的差异。

#### Scenario: 三家适配器投影可比较
- **WHEN** 三家适配器接收等价的 `ModelRequest`
- **THEN** 归一化后的投影结构可直接比较，且差异只以声明过的缺口或稳定 drift 分类表达

#### Scenario: provider-only 字段不进入契约
- **WHEN** 适配器观测到 provider 特有字段
- **THEN** 归一化投影忽略该字段，不新增必需 runtime 语义，也不写入 raw payload 诊断

#### Scenario: Run 与 Stream 投影等价
- **WHEN** 同一 `ModelRequest` 分别经 Run 与 Stream 路径投影
- **THEN** 归一化投影语义等价，不等价时返回 Run/Stream parity drift 分类

### Requirement: Supported adapter request projection SHALL preserve native semantic facts

For OpenAI, Anthropic, and Gemini, a `ModelRequest` whose `Messages` contains system, user, or assistant entries MUST preserve the same ordered role sequence at the provider SDK boundary whenever that provider SDK can express the role. The adapter MUST preserve the source-relative order of messages and the final user instruction, and MUST NOT silently replace a non-empty structured request with a single user text input.

A canonical tool result with a non-empty call identity and tool name MUST reach the provider SDK boundary using that provider's native tool-result or function-response shape when expressible. The provider request MUST preserve the result-to-call association and tool identity. Text-envelope projection is not a conforming success path for a supported native shape.

#### Scenario: Structured roles remain observable at the SDK boundary
- **WHEN** a request contains ordered system, user, and assistant messages together with a final user instruction
- **THEN** the normalized SDK-boundary projection retains the corresponding ordered role facts and does not collapse them into one user-text fact

#### Scenario: Tool result reaches a native request part
- **WHEN** a request contains a canonical tool result with valid call identity and tool name
- **THEN** the normalized SDK-boundary projection marks the result as native, retains the call/tool association, and does not mark it as a text-envelope result

#### Scenario: Unsupported native representation fails before invocation
- **WHEN** an adapter cannot safely express a canonical role or valid tool result in its provider-native request shape
- **THEN** it returns a deterministic request-shape failure before the provider request is sent and does not emit a partial text-envelope fallback

### Requirement: Request projection parity SHALL cover Run Stream and token accounting

For the same canonical request facts and equivalent provider response usage, each supported adapter's Run and Stream paths MUST normalize cache usage with semantically equivalent availability and read/write/create classification. A provider SDK may report usage at different lifecycle events, but normalization MUST be deterministic and MUST NOT double-count partial and terminal usage. CountTokens paths MAY omit cache usage when the provider API cannot report it; such omission MUST resolve to nullable/default unavailable semantics rather than fabricated values.

#### Scenario: Run and Stream expose equivalent cache usage
- **WHEN** equivalent provider responses are delivered through Run and Stream for the same request
- **THEN** replay observes equivalent normalized cache availability and token classification, regardless of lifecycle event placement

#### Scenario: Run and Stream preserve equivalent request semantics
- **WHEN** the same request is projected through Run and Stream
- **THEN** normalized provider-boundary roles, final-input placement, and tool-result association are equivalent

#### Scenario: Stream usage arrives in multiple events
- **WHEN** a stream emits interim and terminal usage records
- **THEN** the adapter selects the contract-defined authoritative record deterministically, avoids double counting, and preserves Run/Stream parity

#### Scenario: CountTokens cannot report cache usage
- **WHEN** a provider token-count API has no cache accounting fields
- **THEN** the count projection leaves cache usage unavailable/default and does not treat counted input tokens as cache reads or writes

#### Scenario: Token accounting matches generation projection facts
- **WHEN** the same request is counted and executed through Run or Stream on one supported adapter
- **THEN** token-count and generation requests normalize to equivalent role and tool-result projection facts, apart from fields unavailable to the token-count API

### Requirement: Resolved request projection gaps SHALL be migrated explicitly

When a runtime repair resolves a previously declared role, tool-result-native, ordering, or Run/Stream projection gap, the versioned fixture MUST replace that `declared_gap` with an explicit no-gap native expectation in the same change. Replay and gate verification MUST fail if a resolved gap remains declared, if a newly observed gap is undeclared, or if the updated native expectation is not deterministic.

#### Scenario: Resolved gap cannot remain silently declared
- **WHEN** a fixture still declares a gap that is no longer observed after native request projection is implemented
- **THEN** replay fails with the existing request-projection contract drift classification until the fixture expectation is explicitly updated

### Requirement: Tool-result 投影 SHALL 保持关联且可判定归属

请求投影 MUST 保留 tool result 与 tool call 的关联标识与工具身份，并 MUST 显式表达该结果是以 provider 原生 tool 归属形式还是以文本信封形式进入请求。缺失关联的 tool result MUST 在 provider 调用前以既有 `feedback_invalid` 分类失败。

#### Scenario: 关联标识在投影中可判定
- **WHEN** `ModelRequest` 携带带关联的 tool result
- **THEN** 归一化投影保留 call/name 关联并标注其归属形式（原生或文本信封）

#### Scenario: 缺失关联在 provider 调用前失败
- **WHEN** tool result 缺少 `call_id` 或工具名
- **THEN** 投影在 provider 调用前以 `feedback_invalid` 失败，且不发出部分请求

### Requirement: Usage 与 cache 可观测性 SHALL 遵守 additive + nullable + default

usage 投影 MUST 显式表达 cache 计量是否可用，并在来源可验证时表达 provider-neutral 的 cache read、cache write/create 与 bounded total 语义。`total_tokens` 在该结构中表示 cache read + write/create 的合计，不表示 provider 的全部 input tokens。规范来源由有界 `source_kind` 与 `source_version` 表达；不得将 SDK 原始字段名或其他 provider-only 元数据提升为通用字段。OpenAI cached input、Anthropic cache read/cache creation input、Gemini cached content token 等 provider-native 字段 MUST 只能由对应 adapter 在其 SDK 边界解释；`model/conformance`、replay 和 runtime MUST NOT 引入 provider SDK 依赖或根据普通 input token 推导 cache 数值。

适配器 MUST 将可用 projection 放入独立的 `ModelResponse.CacheUsage` carrier；Stream MUST 仅在权威终态事件的 `Meta["cache_usage"]` 中暴露同一归一化快照。该载体 MUST NOT 修改既有 `TokenUsage` 字段语义或 RuntimeRecorder schema。

当适配器没有 cache 计量来源、来源字段缺失、来源语义不能安全归一化，或 Run/Stream 两条路径无法给出等价事实时，投影 MUST 表达 `available=false`，并 MUST NOT 填充 read/write/total 数值。任何新增 usage/cache 字段 MUST 遵守 `additive + nullable + default`：历史 fixture 缺失字段 MUST 按 unavailable/default 处理且解析不得失败，未知字段 MUST 被安全忽略。非负性、总量一致性和 provider-native 语义不确定性 MUST 以稳定 drift 分类 fail-fast，不得静默修正。

#### Scenario: Provider cache usage is projected from an observed source
- **WHEN** an adapter receives a provider response whose official SDK exposes a non-negative cache read or cache creation/write count
- **THEN** the provider-neutral projection marks cache usage available, preserves the bounded source classification, and exposes only the normalized nullable/default fields defined by the contract

#### Scenario: Cache usage is unavailable without a reliable source
- **WHEN** an adapter response lacks cache accounting or the provider-native fields cannot be safely mapped
- **THEN** the projection marks `available=false`, leaves cache read/write/total values at their defaults, and does not infer values from ordinary input tokens

#### Scenario: 当前适配器表达 cache 不可用
- **WHEN** a supported adapter does not return cache accounting
- **THEN** the projection expresses `available=false` and does not populate read/write values

#### Scenario: Historical fixture omits cache fields
- **WHEN** a fixture written before cache usage support contains no cache fields
- **THEN** replay succeeds, resolves the fields to unavailable/default values, and preserves the historical projection semantics

#### Scenario: 历史 fixture 缺少 cache 字段仍可解析
- **WHEN** a historical fixture omits all cache usage fields
- **THEN** parsing succeeds and applies the documented default semantics without drift

#### Scenario: Unknown future usage fields are present
- **WHEN** a fixture contains additional provider usage fields not defined by the current contract
- **THEN** replay ignores those fields safely and does not change the canonical digest or drift classification

#### Scenario: 未知字段被安全忽略
- **WHEN** a fixture contains a field unknown to the current contract
- **THEN** parsing succeeds and the unknown field does not affect projection, digest, or drift classification

#### Scenario: Cache counts are invalid or inconsistent
- **WHEN** a cache read/write/total value is negative, overflows a bounded field, or violates the contract's declared total relationship
- **THEN** replay fails fast with a stable cache usage projection drift classification and does not return a partially normalized projection
