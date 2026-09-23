## Purpose

本能力为已通过宿主准入的工具集合建立有界、provider-neutral、可离线回放的 schema pressure 与 selection quality audit，使是否需要后续按需投影由可复现证据决定，而不是由运行时猜测或隐式 selector 决定。

## ADDED Requirements

### Requirement: Audit input SHALL be versioned, bounded, and admission-scoped

审计 MUST 接收版本化 `tool_schema_pressure_selection_audit.v1` 输入，其中包含 snapshot identity、已准入工具、canonical schema facts、task gold set、pressure budgets、strategy descriptors 和 optional corpus advisory metadata。输入 MUST 只包含 provider-neutral bounded facts，不得包含 raw schema、prompt、model output、tool result body、endpoint、credential 或 provider SDK 类型。

#### Scenario: Valid admitted snapshot is accepted
- **WHEN** host supplies a supported version with bounded admitted tools, canonical schema facts, gold labels and strategy parameters
- **THEN** audit produces a normalized result with no provider, network, filesystem or execution dependency

#### Scenario: Unbounded or non-admitted input is rejected
- **WHEN** tool count, identity length, schema bytes, label count, strategy count or serialized input exceeds its declared bound, or a tool lacks an admission fact
- **THEN** audit fails fast with a stable overflow/admission classification and returns no partial result

### Requirement: Schema normalization SHALL be deterministic and privacy-preserving

等价 schema 输入 MUST 产生相同 canonical digest、byte length 和 estimator version。token estimate MUST 使用版本化 provider-neutral estimator；审计输出只能保留 digest、长度、计数、有限标签和 bounded reason，不得持久化 raw schema 或内容正文。

#### Scenario: Equivalent schema order normalizes identically
- **WHEN** two inputs contain equivalent JSON schema objects with different object-key or tool ordering
- **THEN** canonical digest, bytes, estimate and normalized tool order are identical

#### Scenario: Privacy material is supplied
- **WHEN** input contains endpoint, credential-like field, prompt, raw response or oversized schema body
- **THEN** validation rejects or redacts it with a stable privacy/overflow classification before scoring

### Requirement: Pressure metrics SHALL be reproducible and bounded

审计 MUST 计算 admitted tool count、projected count、canonical schema bytes、deterministic token estimate、per-tool size summary、pressure level 和 budget comparison。Pressure level MUST be one of the versioned bounded values `within_budget`、`elevated`、`exceeds_budget` 或 `insufficient_measurement`。

#### Scenario: Full admitted set exceeds pressure budget
- **WHEN** canonical bytes or deterministic token estimate exceeds the fixture budget
- **THEN** baseline result records `exceeds_budget` with bounded metric values and no runtime projection

#### Scenario: Missing measurement is not treated as zero
- **WHEN** a tool has no valid canonical schema fact or estimator version
- **THEN** audit records `insufficient_measurement` and does not fabricate a zero-size metric

### Requirement: Selection quality SHALL use synthetic gold sets as mandatory evidence

每个强制 synthetic case MUST declare task intent, expected tools, allowed tools, forbidden tools and optional fallback tools. Audit MUST compute deterministic precision、recall、F1、expected-hit、forbidden-hit 和 fallback coverage；gold 集合冲突、空集合策略或越界标签 MUST fail with stable fixture classification。

#### Scenario: Strategy improves quality without forbidden hits
- **WHEN** a strategy selects tools with higher expected-hit and F1, no forbidden tools, and valid allowed/fallback membership
- **THEN** the normalized result records the strategy metrics and preserves deterministic ordering

#### Scenario: Gold set is contradictory
- **WHEN** a tool is present in both forbidden and expected sets, or an expected tool is absent from the admitted snapshot
- **THEN** audit fails with `gold_set_conflict` and does not emit a projection candidate

### Requirement: Strategy comparison SHALL be finite, deterministic, and advisory-only

Audit MUST support the versioned strategies `full_admitted_set`、`capability_filtered_set`、`priority_top_k`、`source_partitioned_set` and bounded `fixture_declared_strategy` parameters. Equivalent inputs MUST produce the same candidate set, tie-break order, metrics and skip reasons. Strategy output MUST NOT alter tool admission, registry, ModelRequest, provider projection or execution.

#### Scenario: Multiple strategies produce stable comparison
- **WHEN** one fixture declares two or more supported strategies with equivalent input ordering
- **THEN** replay produces identical strategy order, metrics and selected identities across repeated runs

#### Scenario: Strategy attempts runtime wiring
- **WHEN** a contribution adds selector/router state, ModelRequest tool-subset wiring, provider tokenizer use, or registry mutation to consume audit output
- **THEN** the boundary gate rejects the change with a library-first/runtime-wiring classification

### Requirement: Projection candidate conclusion SHALL require pressure and quality evidence

Audit MUST emit one conclusion from the versioned set `baseline_sufficient`、`pressure_only`、`quality_only`、`pressure_and_quality_gap`、`projection_candidate` 或 `insufficient_evidence`。`projection_candidate` MUST require baseline pressure exceedance, baseline quality degradation, and deterministic improvement by at least one strategy on both pressure and quality metrics. The conclusion MUST remain offline advisory data.

#### Scenario: Stable pressure and quality gap activates candidate evidence
- **WHEN** baseline exceeds pressure budget, baseline quality degrades, and one strategy improves both pressure and quality without forbidden hits
- **THEN** audit emits `projection_candidate` with bounded evidence and no runtime side effect

#### Scenario: Pressure alone is insufficient
- **WHEN** schema pressure exceeds budget but baseline selection quality remains within threshold
- **THEN** audit emits `pressure_only` and does not recommend runtime projection

### Requirement: Eval corpus and Badcase input SHALL remain optional advisory evidence

Replay MAY accept versioned Eval corpus/Badcase metadata containing bounded coverage, label completeness and aggregate trend facts. Missing corpus, unknown fields, unlabeled cases or insufficient coverage MUST NOT fail the synthetic gate or invent quality metrics. Advisory evidence MUST NOT write back to Eval, tool registry, Skill, MCP, policy or runtime config.

#### Scenario: Corpus is absent
- **WHEN** a valid synthetic fixture omits optional corpus advisory input
- **THEN** audit succeeds with nullable/default advisory fields and preserves the synthetic conclusion

#### Scenario: Corpus coverage is insufficient
- **WHEN** corpus metadata reports insufficient labeled cases or unsupported fields
- **THEN** audit records advisory insufficiency without changing the mandatory synthetic result

### Requirement: Replay SHALL preserve compatibility, parity, and deterministic drift classes

Diagnostics replay MUST validate canonical digest、pressure metrics、quality metrics、strategy order、conclusion、unknown-field tolerance、historical defaults、overflow、gold conflict、strategy drift 和 Run/Stream semantic parity. Replaying the same fixture twice MUST produce identical normalized output and classifications.

#### Scenario: Historical fixture omits optional strategy and corpus fields
- **WHEN** a pre-advisory fixture contains only baseline pressure and synthetic quality facts
- **THEN** replay succeeds using documented defaults without inventing strategy or corpus drift

#### Scenario: Run and Stream differ semantically
- **WHEN** equivalent Run and Stream audit inputs produce different pressure, quality, strategy, conclusion or reason facts
- **THEN** replay fails with a stable `run_stream_parity_drift` classification

### Requirement: Audit contract SHALL not create a parallel control plane

The capability MUST remain offline, read-only and library-first. It MUST NOT perform remote discovery, dynamic download, marketplace resolution, credential storage/probing, provider SDK/tokenizer calls, global mutable selector/router creation, second admission/readiness/terminal state machine, or raw payload persistence.

#### Scenario: Remote or runtime-owned source is requested
- **WHEN** an audit input attempts to resolve tools from a remote source or consume runtime mutation callbacks
- **THEN** validation fails with a stable unsupported-source/control-plane classification and performs no I/O

#### Scenario: Raw payload persistence is detected
- **WHEN** fixture, replay or diagnostics code stores raw schema, prompt, model output or tool result body
- **THEN** contribution gate rejects the change before merge
