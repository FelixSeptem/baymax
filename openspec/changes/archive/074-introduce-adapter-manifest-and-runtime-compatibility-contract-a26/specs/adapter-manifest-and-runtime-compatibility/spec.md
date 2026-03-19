## ADDED Requirements

### Requirement: Repository SHALL define adapter manifest schema for runtime compatibility checks
The repository MUST define a machine-readable adapter manifest contract for external adapters.

The manifest MUST include at least:
- `type`,
- `name`,
- `version`,
- `baymax_compat`,
- `capabilities.required`,
- `capabilities.optional`,
- `conformance_profile`.

#### Scenario: Contributor validates manifest structure
- **WHEN** contributor runs manifest validation for an adapter project
- **THEN** validation succeeds only when required manifest fields are present and syntactically valid

#### Scenario: Contributor omits mandatory manifest field
- **WHEN** manifest is missing one or more mandatory fields
- **THEN** validation fails fast with deterministic field-level error classification

### Requirement: Runtime SHALL enforce manifest compatibility at adapter integration boundary
Adapter integration boundary MUST evaluate `baymax_compat` against current Baymax runtime version before adapter activation.

Semver range expressions MUST be supported, and pre-release versions (including `-rc`) MUST be accepted when expression resolution allows them.

#### Scenario: Runtime version matches manifest compatibility range
- **WHEN** current runtime version satisfies adapter `baymax_compat` expression
- **THEN** adapter activation can proceed to subsequent checks

#### Scenario: Runtime version is out of compatibility range
- **WHEN** current runtime version does not satisfy adapter `baymax_compat`
- **THEN** adapter activation fails fast with compatibility-mismatch classification

### Requirement: Runtime SHALL apply required and optional capability semantics deterministically
Manifest capability declarations MUST support `required` and `optional` sets with deterministic enforcement:
- missing `required` capability MUST fail fast,
- missing `optional` capability MAY downgrade behavior and MUST emit deterministic downgrade reason.

#### Scenario: Required capability is unavailable
- **WHEN** adapter declares a required capability that runtime or adapter implementation cannot satisfy
- **THEN** activation fails fast and adapter is not accepted

#### Scenario: Optional capability is unavailable
- **WHEN** adapter declares an optional capability that is not available
- **THEN** runtime activates adapter with deterministic downgrade behavior and reason classification

### Requirement: Manifest validation SHALL run in offline deterministic mode
Manifest validation and compatibility checks MUST be executable offline and MUST NOT require external network access.

#### Scenario: CI validates adapters without network access
- **WHEN** CI executes manifest contract checks in isolated environment
- **THEN** checks run deterministically without external credentials or network dependencies
