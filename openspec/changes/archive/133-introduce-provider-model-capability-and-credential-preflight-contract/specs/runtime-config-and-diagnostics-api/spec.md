## ADDED Requirements

### Requirement: Provider catalog configuration has deterministic precedence and validation
The runtime SHALL expose provider-model catalog controls under `runtime.provider_catalog.*` and resolve them with `env > file > default` precedence. Startup with an enabled invalid catalog MUST fail fast. Validation MUST reject duplicate normalized descriptors, malformed capabilities, nonpositive context windows, invalid declared fallbacks, and invalid credential-preflight policy values with bounded canonical reasons.

#### Scenario: Environment catalog setting overrides file setting
- **WHEN** a valid `runtime.provider_catalog.*` value is supplied by both environment and file configuration
- **THEN** the runtime admits the environment value and records its normalized catalog version without changing unrelated configuration precedence

#### Scenario: Invalid startup catalog fails before provider dispatch
- **WHEN** enabled startup configuration contains a descriptor with a nonpositive context window
- **THEN** initialization fails fast with the canonical validation reason and does not create a provider dispatch

### Requirement: Catalog hot reload is atomic
The runtime SHALL validate an entire replacement catalog and credential-preflight policy before publication. A valid candidate MUST become visible only to later admissions. An invalid candidate MUST atomically retain the previous complete runtime snapshot and surface a canonical rollback finding.

#### Scenario: Invalid replacement preserves prior active catalog
- **WHEN** a runtime hot reload proposes a catalog with an invalid fallback target
- **THEN** the manager retains the preceding valid catalog for later admissions and emits the canonical reload-rollback finding

### Requirement: Provider-model diagnostics are additive and redacted
Runtime diagnostics SHALL add nullable fields with compatibility defaults for catalog version, normalized provider/model identity, selected fallback identity, required and optional capability outcome, credential evidence status, and canonical reason codes. These fields MUST be bounded, normalized, and free of credential material, endpoints, raw provider responses, and validation payloads. Only `observability/event.RuntimeRecorder` SHALL write the diagnostic projection.

#### Scenario: Legacy diagnostic reader receives compatibility defaults
- **WHEN** a diagnostic record predates provider-model admission fields
- **THEN** a reader receives the documented nullable or default values without a schema or decoding failure

#### Scenario: Credential material is excluded from diagnostics
- **WHEN** a host supplies credential evidence with status and reason code
- **THEN** the persisted diagnostic record contains only the redacted status and canonical reason and contains no secret or endpoint value

### Requirement: Diagnostic replay preserves admission projection parity
The runtime SHALL replay provider-model admission diagnostics idempotently from versioned, redacted fixtures. The replay contract MUST detect drift in catalog identity, capability outcome, credential status, fallback identity, canonical reason order, and equivalent Run/Stream projection.

#### Scenario: Replaying the same admission event is idempotent
- **WHEN** the same valid provider-model admission event is replayed more than once
- **THEN** the diagnostic projection remains equivalent for Run and Stream and does not duplicate or reorder canonical findings
