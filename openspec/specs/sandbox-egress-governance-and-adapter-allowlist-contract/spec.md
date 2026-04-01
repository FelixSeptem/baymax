# sandbox-egress-governance-and-adapter-allowlist-contract Specification

## Purpose
TBD - created by archiving change introduce-sandbox-egress-governance-and-adapter-allowlist-contract-a57. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL enforce sandbox egress governance with deny-first default
Runtime MUST evaluate sandbox egress policy before tool execution that may access network resources.

Egress policy MUST support canonical actions:
- `deny`
- `allow`
- `allow_and_record`

Default behavior MUST be deny-first when no explicit allowlist rule is matched.

#### Scenario: Tool network request without explicit allowlist rule
- **WHEN** tool execution attempts network egress and no matching egress allow rule exists
- **THEN** runtime denies egress with canonical deny classification

#### Scenario: Tool network request with explicit allowlist rule
- **WHEN** tool execution targets host or domain explicitly allowed by policy
- **THEN** runtime allows egress and records canonical decision metadata

### Requirement: Adapter activation SHALL be governed by allowlist contract
Runtime MUST validate adapter allowlist metadata before adapter activation.

Minimum allowlist identity dimensions for this milestone:
- adapter id
- publisher
- version
- signature state

Adapters failing allowlist validation MUST be blocked before runtime activation.

#### Scenario: Adapter is not in allowlist
- **WHEN** runtime resolves adapter metadata and no matching allowlist entry is found
- **THEN** activation fails fast and adapter is not loaded

#### Scenario: Adapter signature state is invalid
- **WHEN** adapter metadata indicates invalid signature state under enforce policy
- **THEN** activation fails fast with deterministic allowlist classification

### Requirement: Egress and allowlist findings SHALL be consumable by readiness and admission
Egress and allowlist violations MUST produce canonical findings for readiness preflight and deterministic admission mapping.

#### Scenario: Readiness observes egress policy conflict
- **WHEN** preflight detects invalid or unsafe effective egress policy
- **THEN** readiness emits canonical `sandbox.egress.*` finding with machine-readable metadata

#### Scenario: Admission consumes allowlist blocking finding
- **WHEN** readiness outputs blocking `adapter.allowlist.*` finding
- **THEN** admission denies managed execution with side-effect-free semantics

