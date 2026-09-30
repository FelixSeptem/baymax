## Purpose

为已通过既有宿主准入的工具集合提供有界、provider-neutral、可离线回放的 readiness 证据，判断按需 Tool Schema 投影是否达到进入后续运行时设计的稳定触发条件，同时不改变当前工具执行或请求投影语义。

## ADDED Requirements

### Requirement: Readiness input SHALL be versioned, bounded, and admission-scoped

Readiness evaluation MUST accept a versioned `tool_schema_projection_readiness.v1` input containing a bounded snapshot identity, already-admitted tool identities, canonical schema facts, pressure budgets, task quality cases, and an optional sample window. Every tool MUST carry an admission fact inherited from the existing admission owners. Raw schema bodies, prompts, model output, tool results, endpoints, credentials, provider SDK values, and unbounded text MUST be rejected.

#### Scenario: Valid admitted readiness snapshot is accepted
- **WHEN** a host supplies a supported version with bounded admitted tools, canonical facts, budgets, and quality cases
- **THEN** evaluation returns a normalized provider-neutral result without network, filesystem, model, or tool execution

#### Scenario: Non-admitted or oversized input is rejected
- **WHEN** a tool lacks an admission fact or a declared count, identity, schema, window, or serialized input bound is exceeded
- **THEN** evaluation fails fast with a stable admission or overflow classification and emits no partial readiness result

### Requirement: Sample windows SHALL distinguish stable pressure from isolated peaks

The evaluator MUST normalize a finite ordered sample window and classify pressure and selection signals as `stable`, `transient`, `within_budget`, or `insufficient_measurement` using versioned thresholds supplied by the fixture. A single outlier MUST NOT satisfy a stable-bottleneck condition. Equivalent samples supplied in equivalent ordering MUST produce the same window digest and classification.

#### Scenario: Repeated pressure satisfies the stable trigger
- **WHEN** the required number of bounded samples repeatedly exceeds the declared schema-token or budget threshold
- **THEN** the result records stable pressure evidence with the contributing sample ordinals and deterministic window digest

#### Scenario: One-time peak remains transient
- **WHEN** only one sample exceeds the threshold while the remaining valid samples remain within budget
- **THEN** the result records transient pressure and the overall conclusion cannot be `ready_for_runtime_design`

### Requirement: Projection opportunity SHALL be advisory and admission-preserving

The evaluator MUST compute an advisory projection opportunity only from the supplied admitted set and task quality cases. Any candidate projected set MUST be a deterministic subset of the admitted identities, preserve required schema fields and capability semantics, and report omitted identities with bounded reasons. The evaluator MUST NOT mutate admission, registry, lifecycle, ModelRequest, provider projection, or execution state.

#### Scenario: Safe subset exposes measurable opportunity
- **WHEN** an admitted subset reduces pressure across stable samples while retaining required task tools and schema semantics
- **THEN** the result records an opportunity with deterministic selected identities, retained fields, and bounded omission reasons

#### Scenario: Candidate attempts to add or bypass a tool
- **WHEN** an opportunity contains an identity absent from the admitted snapshot or bypasses allowlist, manifest, capability, or sandbox facts
- **THEN** evaluation returns `blocked` with an admission-integrity classification and no candidate is emitted

### Requirement: Readiness verdict SHALL expose independent evidence axes

The normalized result MUST expose nullable-or-default verdicts for pressure signal, projection opportunity, selection quality, admission integrity, semantic completeness, Run/Stream parity, and replay determinism. Each axis MUST be independently inspectable and MUST NOT be inferred from a single aggregate score.

#### Scenario: Evidence axes are complete
- **WHEN** a bounded fixture contains valid pressure, quality, opportunity, parity, and replay evidence
- **THEN** the result contains all seven axes with stable enum values and bounded metrics or reasons

#### Scenario: Missing optional evidence is represented explicitly
- **WHEN** a fixture omits an optional sample, quality, or opportunity section
- **THEN** the corresponding axis uses its documented nullable/default value and the evaluator does not fabricate a positive signal

### Requirement: Overall conclusion SHALL require stable pressure and quality evidence

The conclusion MUST be one of `not_ready`, `ready_for_runtime_design`, or `blocked`. `ready_for_runtime_design` MUST require stable pressure, a deterministic subset opportunity, non-degrading selection quality, admission integrity, semantic completeness, Run/Stream parity, and replay determinism. Missing or transient evidence MUST produce `not_ready`; any admission, privacy, semantic, parity, or replay violation MUST produce `blocked`.

#### Scenario: Stable evidence activates readiness
- **WHEN** pressure is stable, the subset opportunity is measurable, quality is preserved or improved, and all integrity/parity axes pass
- **THEN** the conclusion is `ready_for_runtime_design` and explicitly states that no runtime projection is enabled

#### Scenario: Stable pressure without quality evidence is insufficient
- **WHEN** stable pressure exists but task quality is missing, degraded, or contradictory
- **THEN** the conclusion is `not_ready` with a bounded insufficiency reason

### Requirement: Replay SHALL be deterministic, compatible, and Run/Stream equivalent

Replay MUST accept unknown additive fields, apply documented defaults for omitted optional fields, and classify canonical digest, window, metrics, selected identities, conclusion, and reason-order drift. Equivalent Run and Stream fixtures MUST produce semantically equivalent normalized results. Replaying an unchanged fixture twice MUST produce identical output and classifications.

#### Scenario: Historical fixture remains readable
- **WHEN** a fixture omits optional window or opportunity fields and contains only the required v1 facts
- **THEN** replay succeeds using documented defaults and retains a `not_ready` or evidence-limited conclusion rather than inventing readiness

#### Scenario: Run and Stream parity drifts
- **WHEN** equivalent Run and Stream inputs differ in pressure, quality, selected identities, conclusion, or reason sequence
- **THEN** replay returns stable `run_stream_parity_drift` and the overall conclusion is `blocked`

### Requirement: Readiness contract SHALL remain offline and library-first

The capability MUST NOT perform remote discovery, dynamic download, marketplace resolution, credential storage or probing, provider SDK/tokenizer calls, runtime diagnostics writes, global mutable selector/router creation, or a second admission/readiness/terminal state machine. It MUST retain only bounded digests, counts, metrics, identities, and reasons rather than raw payloads.

#### Scenario: Remote or runtime-owned source is requested
- **WHEN** an input asks the evaluator to resolve tools remotely or consume runtime mutation callbacks
- **THEN** validation rejects the source with a stable unsupported-control-plane classification and performs no I/O

#### Scenario: Raw payload persistence is detected
- **WHEN** fixture, replay, or diagnostics code attempts to persist raw schema, prompt, model output, or tool-result content
- **THEN** the contribution boundary rejects the change before merge

### Requirement: Governance gates SHALL verify the readiness boundary and documentation status

The change MUST provide deterministic contract/replay tests and shell/PowerShell gates covering bounded input, stable-window classification, admission subset integrity, semantic completeness, parity, compatibility, privacy, and offline boundaries. Documentation gates MUST verify that archived changes are not listed as active and that the new readiness capability is mapped to its owning tests and scripts.

#### Scenario: Boundary and documentation gates pass
- **WHEN** valid fixtures, negative fixtures, replay drift cases, Run/Stream parity cases, and roadmap/status mappings are evaluated
- **THEN** both platform gate variants pass and identify the readiness contract as the source of truth

#### Scenario: Archived status drift is present
- **WHEN** Roadmap or README marks an archived OpenSpec change as in progress
- **THEN** the documentation consistency gate fails with a stable status-parity classification until the source is corrected
