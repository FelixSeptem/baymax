## ADDED Requirements

### Requirement: Runtime SHALL expose CA2 external capability-hint and template-pack config with deterministic precedence
Runtime configuration MUST expose CA2 external retriever capability-hint and template-pack fields with precedence `env > file > default`.

The minimum template-pack profile set for this milestone MUST include:
- `graphrag_like`
- `ragflow_like`
- `elasticsearch_like`

Runtime MUST support deterministic resolution semantics aligned with Stage2 execution:
- profile defaults are resolved first,
- explicit mapping/auth/header fields override profile defaults,
- explicit-only mode is valid when no template profile is selected.

Invalid template-pack profile values or malformed hint structure MUST fail fast during startup and hot reload.

#### Scenario: Startup with template-pack profile override
- **WHEN** CA2 template-pack profile is configured in both YAML and environment variables
- **THEN** effective profile resolves by `env > file > default` and participates in deterministic template resolution

#### Scenario: Startup with explicit-only external mapping
- **WHEN** runtime starts with no template-pack profile and valid explicit external mapping fields
- **THEN** runtime accepts configuration and Stage2 can execute explicit-only mapping path

#### Scenario: Invalid template-pack profile value
- **WHEN** runtime receives unsupported template-pack profile value
- **THEN** startup or hot reload is rejected with fail-fast validation error

#### Scenario: Malformed capability-hint config
- **WHEN** runtime receives malformed capability-hint structure
- **THEN** startup or hot reload is rejected with fail-fast validation error

### Requirement: Runtime diagnostics SHALL expose additive CA2 hint and template resolution fields
Runtime diagnostics MUST expose additive CA2 Stage2 fields for hint/template observability without breaking existing consumer semantics.

The minimum additive fields MUST include:
- `stage2_template_profile`
- `stage2_template_resolution_source`
- `stage2_hint_applied`
- `stage2_hint_mismatch_reason`

Hint mismatch and template anomalies MUST remain observational only and MUST NOT imply automatic strategy actions.

#### Scenario: Consumer inspects successful hint and template resolution
- **WHEN** Stage2 executes with valid template profile and applied capability hints
- **THEN** diagnostics include resolved template profile, resolution source, and hint-applied marker

#### Scenario: Consumer inspects hint mismatch
- **WHEN** Stage2 executes with unsupported or invalid capability hints
- **THEN** diagnostics include normalized `stage2_hint_mismatch_reason` while preserving existing stage-policy outcomes

#### Scenario: Existing diagnostics consumer reads legacy fields only
- **WHEN** consumer parses only pre-existing CA2 diagnostic fields
- **THEN** diagnostics remain backward-compatible and existing field semantics are unchanged

### Requirement: Runtime SHALL preserve CA2 layered error semantics while extending hint and template fields
Runtime diagnostics and event mappings for CA2 Stage2 MUST preserve baseline layered error semantics (`transport`, `protocol`, `semantic`) and MAY include forward-compatible enum extension values.

Hint/template-related diagnostics MUST be additive extensions and MUST NOT redefine baseline error-layer meanings.

#### Scenario: Baseline error layer with template profile
- **WHEN** Stage2 retrieval fails with baseline layered error under active template profile
- **THEN** diagnostics preserve baseline error-layer semantics and include additive template context fields

#### Scenario: Extended error layer value with hint mismatch
- **WHEN** implementation emits an extended layer enum value while Stage2 also records hint mismatch
- **THEN** diagnostics preserve extended value and additive mismatch fields without schema conflict

### Requirement: Repository SHALL protect CA2 hint-template contract with contract tests and benchmark baseline
The repository MUST include contract tests that validate CA2 hint/template configuration, deterministic resolution behavior, observational-only mismatch semantics, and Run/Stream semantic equivalence.

The repository MUST additionally include benchmark coverage for hint/template resolution overhead and maintain compatibility with existing CA2 trend benchmark baselines.

#### Scenario: Contract test suite for hint and template semantics is executed
- **WHEN** CI or local validation runs CA2 hint/template contract tests
- **THEN** semantic mismatches in precedence, mismatch policy, or Run/Stream equivalence fail the suite

#### Scenario: Benchmark suite for hint and template resolution is executed
- **WHEN** CI or local performance validation runs CA2 benchmarks
- **THEN** benchmark outputs include hint/template resolution baseline and remain comparable with existing CA2 trend baselines
