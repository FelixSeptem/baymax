## ADDED Requirements

### Requirement: Runtime SHALL expose library-level readiness preflight API
Runtime MUST expose a library-level readiness preflight API that returns deterministic readiness result without requiring platform control-plane dependencies.

The API MUST return:
- overall status (`ready|degraded|blocked`),
- structured findings list,
- evaluation timestamp.

#### Scenario: Host invokes readiness preflight before startup
- **WHEN** application calls readiness preflight on runtime manager with valid config snapshot
- **THEN** runtime returns deterministic readiness result with status and structured findings

#### Scenario: Host invokes readiness preflight repeatedly with unchanged snapshot
- **WHEN** application calls readiness preflight multiple times without configuration or dependency changes
- **THEN** runtime returns semantically equivalent readiness status and finding classifications

### Requirement: Readiness classification SHALL support strict degradation policy
Readiness classification MUST support `strict` policy:
- when `strict=false`, degraded conditions MUST map to `degraded`,
- when `strict=true`, degraded conditions MUST escalate to `blocked`.

#### Scenario: Degraded condition under non-strict policy
- **WHEN** runtime detects recoverable degraded finding and `strict=false`
- **THEN** readiness status is `degraded` and findings remain observable

#### Scenario: Degraded condition under strict policy
- **WHEN** runtime detects same degraded finding and `strict=true`
- **THEN** readiness status escalates to `blocked`

### Requirement: Readiness findings SHALL use canonical structured schema
Each readiness finding MUST use canonical fields:
- `code`
- `domain`
- `severity`
- `message`
- `metadata`

Finding `code` values MUST be stable and machine-assertable for contract tests.

#### Scenario: Consumer inspects readiness findings
- **WHEN** readiness preflight returns one or more findings
- **THEN** each finding includes canonical fields and stable machine-readable code

#### Scenario: Regression introduces non-canonical finding shape
- **WHEN** implementation omits required finding field or renames canonical key
- **THEN** contract validation fails and blocks merge

### Requirement: Preflight SHALL include fallback and backend-activation visibility
Readiness preflight MUST include checks for scheduler/mailbox/recovery activation and fallback outcomes in effective runtime snapshot.

Fallback-aware findings MUST classify deterministic degraded conditions when configured persistent backend falls back to memory.

#### Scenario: Persistent backend falls back to memory
- **WHEN** effective runtime path uses fallback-to-memory for scheduler or mailbox
- **THEN** readiness returns degraded finding with deterministic fallback reason metadata

#### Scenario: Invalid mandatory backend activation
- **WHEN** effective runtime configuration requires backend activation and activation fails without valid fallback
- **THEN** readiness returns blocked finding and overall status is `blocked`
