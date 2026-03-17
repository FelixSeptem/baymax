## ADDED Requirements

### Requirement: Runtime config SHALL expose CA3 reranker controls and threshold profile settings
Runtime config MUST expose CA3 reranker controls with deterministic precedence `env > file > default`, including:
- reranker enablement,
- reranker timeout and bounded retry policy,
- provider/model threshold profile map.

Invalid reranker or threshold profile configuration MUST fail fast at startup and hot reload.

#### Scenario: Startup with valid reranker config
- **WHEN** runtime starts with valid CA3 reranker controls and threshold profiles
- **THEN** effective config includes reranker settings and deterministic threshold precedence behavior

#### Scenario: Hot reload with invalid reranker profile
- **WHEN** config update includes malformed threshold profile or invalid timeout
- **THEN** runtime rejects update and preserves previous valid snapshot

#### Scenario: Reranker enabled without provider/model profile
- **WHEN** reranker is enabled and selected provider/model has no configured threshold profile
- **THEN** runtime fails fast with missing-profile validation error

### Requirement: Runtime diagnostics SHALL expose provider/model-scoped CA3 reranker quality fields
Runtime diagnostics MUST expose additive CA3 reranker fields sufficient for tuning and incident triage, including:
- reranker enabled/used marker,
- provider/model identity,
- threshold source,
- threshold-hit status,
- reranker fallback reason.

These fields MUST NOT break existing diagnostics consumers.

#### Scenario: Reranker path succeeds
- **WHEN** CA3 reranker executes successfully
- **THEN** diagnostics include reranker usage marker, provider/model identity, and threshold source

#### Scenario: Reranker path falls back
- **WHEN** reranker is bypassed or fails under `best_effort`
- **THEN** diagnostics include fallback reason and effective decision path marker

#### Scenario: Existing consumer reads legacy fields only
- **WHEN** diagnostics consumer does not parse new reranker fields
- **THEN** existing diagnostics semantics remain backward-compatible

### Requirement: Runtime SHALL expose threshold tuning toolkit integration contract
Runtime-adjacent tooling contract MUST define stable input/output schema for CA3 threshold tuning toolkit, including corpus metadata fields and recommendation artifact schema versioning.

#### Scenario: Toolkit runs with supported schema version
- **WHEN** tuning toolkit receives input matching supported schema version
- **THEN** toolkit produces recommendation artifacts with declared output schema version

#### Scenario: Toolkit receives unsupported schema version
- **WHEN** tuning toolkit input schema version is unsupported
- **THEN** toolkit fails fast with explicit schema-version error and no partial output

#### Scenario: Toolkit minimal output mode
- **WHEN** tuning toolkit run succeeds in configured minimal mode
- **THEN** output contract requires markdown artifact and does not require JSON artifact

#### Scenario: Corpus readiness guidance reported
- **WHEN** tuning toolkit evaluates a corpus for selected provider+model segment
- **THEN** output includes corpus readiness and confidence guidance fields without enforcing fixed hard-gate constants

### Requirement: Runtime SHALL expose reranker extension registration contract
Runtime MUST expose a stable extension registration contract for provider-specific reranker implementations.

The contract MUST preserve existing fail-fast and best-effort policy semantics regardless of built-in or custom implementation path.

#### Scenario: Valid custom reranker registration
- **WHEN** application registers a valid provider-specific reranker implementation
- **THEN** runtime accepts registration and executes custom implementation for matching provider/model

#### Scenario: Invalid custom reranker registration
- **WHEN** application registers incompatible reranker implementation
- **THEN** runtime rejects registration with explicit validation error and preserves built-in path
