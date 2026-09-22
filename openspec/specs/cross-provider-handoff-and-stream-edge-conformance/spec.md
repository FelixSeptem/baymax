# cross-provider-handoff-and-stream-edge-conformance Specification

## Purpose
This capability defines a bounded, replayable contract for preserving equivalent handoff and stream-edge semantics across the supported OpenAI, Anthropic, and Gemini adapters without introducing a new provider owner or routing state machine.

## Requirements

### Requirement: Provider handoff SHALL normalize to one canonical contract

The supported provider adapters MUST normalize semantically equivalent tool-call requests, canonical tool-result feedback, thinking/reasoning projections, step correlation, and error classes into the existing runtime contract. The normalized projection MUST preserve `tool_call_id`, tool identity, canonical arguments/result shape, and causal step/run correlation without exposing provider-only fields as required runtime semantics.

#### Scenario: Equivalent tool call from each provider
- **WHEN** OpenAI, Anthropic, and Gemini produce semantically equivalent tool-call intents
- **THEN** normalization yields equivalent tool identity, arguments, call correlation, and next-step admission semantics

#### Scenario: Provider-only field is absent
- **WHEN** a provider does not expose an optional reasoning or usage field
- **THEN** the normalized contract uses its documented nullable/default representation and does not synthesize a misleading value

### Requirement: Tool-result handoff SHALL preserve correlation and feedback semantics

Adapters MUST accept canonical tool-result feedback and map it to provider-native input while preserving the original tool-call correlation, success/error meaning, and next model-step eligibility. Malformed feedback MUST fail with the canonical feedback-invalid classification before a provider request is issued.

#### Scenario: Canonical feedback round-trip
- **WHEN** the runner sends equivalent canonical tool results to each supported adapter
- **THEN** each adapter produces provider-native feedback that preserves call identity and equivalent continuation classification

#### Scenario: Malformed feedback is rejected
- **WHEN** canonical feedback lacks required correlation or violates bounded shape
- **THEN** the adapter rejects it deterministically as feedback-invalid without issuing a partial provider call

### Requirement: Stream edges SHALL be deterministic and provider-independent

Streaming normalization MUST distinguish start, partial content, terminal completion, abort, empty content, Unicode content, and overflow outcomes. Once the first semantic stream event is emitted, the runtime MUST NOT switch providers for that step. Usage and abort fields MUST remain nullable/defaultable when unsupported and MUST NOT be fabricated.

#### Scenario: Stream starts and completes
- **WHEN** equivalent providers emit a valid stream with zero or more partial events followed by completion
- **THEN** normalized events preserve start/partial/terminal semantics and equivalent final outcome classification

#### Scenario: Stream aborts after emission
- **WHEN** a provider aborts after the first semantic stream event
- **THEN** the current step ends with canonical abort/terminal classification and no provider fallback occurs mid-stream

#### Scenario: Empty or Unicode content is preserved
- **WHEN** a stream contains empty content or valid Unicode including U+2028/U+2029
- **THEN** normalization preserves content semantics without dropping, escaping into a different value, or changing event order

#### Scenario: Overflow is bounded
- **WHEN** provider content or a normalized event exceeds the configured contract bound
- **THEN** the step fails fast with a stable overflow classification and does not emit unbounded diagnostics or continue with truncated causal data

### Requirement: Provider fallback SHALL remain step-boundary deterministic

Capability and request-shape checks MUST occur before provider invocation. Fallback MAY select the next configured provider only before model-step execution or before the first semantic stream event; after that fence, the current step MUST terminate through existing error and terminal arbitration.

#### Scenario: Pre-step capability fallback
- **WHEN** the active provider cannot satisfy required capabilities before invocation and a later candidate can
- **THEN** the runtime selects the later candidate deterministically and preserves causal correlation

#### Scenario: Mid-stream provider failure
- **WHEN** the active provider fails after stream emission begins
- **THEN** the runtime reports canonical failure/abort semantics without concatenating output from another provider

### Requirement: Run and Stream SHALL remain semantically equivalent

Equivalent handoff and stream-edge workloads executed through Run and Stream MUST preserve normalized tool-call/result semantics, causal references, error classes, fallback fence, usage/abort classification, and terminal outcome after permitted event-order normalization. The contract MUST NOT introduce a Stream-only or Run-only provider state machine.

#### Scenario: Equivalent Run and Stream handoff
- **WHEN** equivalent Run and Stream executions perform the same provider handoff and tool-result feedback
- **THEN** their normalized causal, error, fallback, and terminal projections are equivalent

#### Scenario: Equivalent Run and Stream abort
- **WHEN** equivalent Run and Stream executions encounter the same post-start abort condition
- **THEN** both classify the abort and terminal outcome equivalently without provider switching

### Requirement: Conformance fixtures and replay SHALL be bounded and side-effect-free

The repository MUST provide versioned `provider_handoff_stream_edge.v1` fixtures covering valid handoff, malformed feedback, thinking/usage omission, stream completion, abort, empty/Unicode content, overflow, pre-step fallback, mid-stream failure, and Run/Stream parity. Replay MUST normalize deterministically, classify drift with stable reason codes, remain offline/read-only, and preserve compatibility with historical fixtures.

#### Scenario: Canonical fixture replays successfully
- **WHEN** replay processes a valid `provider_handoff_stream_edge.v1` fixture whose normalized digest matches expectation
- **THEN** replay succeeds deterministically without invoking providers, tools, or mutating runtime state

#### Scenario: Fixture drift is classified
- **WHEN** normalized output differs in correlation, event boundary, usage/abort, fallback fence, or parity
- **THEN** replay fails with the corresponding stable drift classification and no partial success

#### Scenario: Historical fixtures remain compatible
- **WHEN** new fixtures run alongside archived provider and replay fixtures
- **THEN** historical fixtures continue to parse and normalize with documented defaults

### Requirement: Conformance gates SHALL enforce ownership and diagnostics boundaries

Shell and PowerShell gates MUST execute equivalent checks for adapter ownership, no mid-stream fallback, bounded fixtures, canonical error taxonomy, Run/Stream parity, and replay idempotency. Provider raw payloads, reasoning bodies, credentials, and unbounded stream content MUST NOT be written to diagnostics or OTel attributes.

#### Scenario: Gate detects provider-specific leakage
- **WHEN** implementation requires provider-only fields, stores raw payloads, or emits unbounded diagnostic content
- **THEN** the conformance gate fails with a deterministic boundary classification

#### Scenario: Shell and PowerShell results agree
- **WHEN** both gate implementations run against the same repository state
- **THEN** they produce equivalent pass/fail classifications for the conformance matrix

### Requirement: Cross-provider conformance SHALL cover request-side projection

已归档的跨 Provider conformance 覆盖响应侧 handoff 与 stream edge；本能力 MUST 扩展至请求侧投影，使 supported adapter 的 `ModelRequest` → provider SDK request 投影属于同一 conformance 家族，可被同一套 fixture、离线 replay 与 gate 覆盖。请求侧与响应侧 MUST 共享 provider-neutral 归一化与稳定 drift 分类口径，MUST NOT 各自定义平行语义。

#### Scenario: 请求侧与响应侧共享归一化家族
- **WHEN** 同一 provider 同时产生请求侧与响应侧投影证据
- **THEN** 二者使用同一 provider-neutral 归一化层与同一 drift 词表，不存在第二套分类

#### Scenario: 请求侧 provider-only 字段被隔离
- **WHEN** provider SDK 请求形状包含 provider 特有字段
- **THEN** conformance 归一化不把该字段提升为必需 runtime 语义，也不写入 raw payload 诊断

### Requirement: Request-side conformance SHALL be observable through existing adapter seams

请求侧审计 MUST 复用既有注入缝隙（OpenAI 的 SDK 参数级缝隙、Anthropic/Gemini 的适配器→SDK 边界注入），MUST NOT 新增导出 API、MUST NOT 新增运行时配置键。门禁 MUST 静态校验三家适配器的 SDK 请求构造仍归属各自 `model/<provider>` 包，且 canonical request-facts 归一化与校验不依赖 provider SDK。

三家 supported adapter MUST 使用各自官方 SDK 可表达的原生请求形状保留 `ModelRequest.Messages` 的 system/user/assistant 角色、稳定顺序、最终用户输入和有效 tool-result 的 call/name 关联。Run 与 Stream MUST 产生语义等价的归一化请求投影；CountTokens 若可用，MUST 采用相同的 canonical request facts。门禁与离线 replay MUST 使用版本化 fixture 证明归档 142 中已修复的 declared gap 已被显式迁移，且不得记录 raw prompt、reasoning、credentials 或无界 provider payload。

#### Scenario: SDK-boundary capture proves native projection
- **WHEN** 三家 adapter 接收包含角色消息与有效 tool result 的同一 canonical request facts
- **THEN** 各自注入缝隙捕获到的 SDK-boundary request 都保留可比较的角色、顺序和原生结果关联事实

#### Scenario: Run Stream parity regression is blocked
- **WHEN** 同一 adapter 的 Run 与 Stream 路径在角色、最终输入位置或 tool-result 关联上产生不同归一化投影
- **THEN** conformance suite 以稳定的 Run/Stream parity drift 分类失败

#### Scenario: Provider-owned construction boundary is preserved
- **WHEN** 实现新增或修改请求映射代码
- **THEN** provider SDK 类型仅存在于相应的 `model/<provider>` 包，SDK-neutral canonical facts 不形成共享 provider wire protocol

#### Scenario: 审计不新增运行时 API
- **WHEN** 请求侧 conformance 审计被执行
- **THEN** 审计仅依赖既有 SDK-boundary 注入缝隙完成，未新增导出 API、配置键或平行运行时分支

#### Scenario: 适配器所有权被门禁保护
- **WHEN** SDK 请求构造逻辑被移动到 `model/<provider>` 之外的包
- **THEN** 对等 gate 失败并给出稳定失败分类
