## MODIFIED Requirements

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
