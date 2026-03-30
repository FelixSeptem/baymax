# adapter-manifest-and-runtime-compatibility Specification

## Purpose
TBD - created by archiving change introduce-adapter-manifest-and-runtime-compatibility-contract-a26. Update Purpose after archive.
## Requirements
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

### Requirement: Adapter manifest SHALL include contract profile version field
Adapter manifest contract MUST include `contract_profile_version`.

This field MUST be validated together with manifest compatibility checks before adapter activation.

#### Scenario: Manifest omits contract profile version
- **WHEN** adapter manifest is missing `contract_profile_version`
- **THEN** manifest validation fails fast before adapter activation

#### Scenario: Manifest profile and baymax compatibility both valid
- **WHEN** manifest passes both `contract_profile_version` and `baymax_compat` checks
- **THEN** adapter activation may proceed to negotiation stage

### Requirement: Adapter manifest SHALL declare sandbox profile-pack compatibility metadata
For sandbox-executor adapters, manifest contract MUST additionally declare:
- `sandbox_backend` (`linux_nsjail|linux_bwrap|oci_runtime|windows_job`)
- `sandbox_profile_id`
- `host_os`
- `host_arch`
- `session_modes_supported`

Missing or malformed sandbox metadata MUST fail manifest validation.

#### Scenario: Sandbox adapter manifest declares complete metadata
- **WHEN** manifest includes canonical sandbox backend/profile/platform/session fields
- **THEN** manifest validation succeeds for sandbox metadata section

#### Scenario: Sandbox adapter manifest omits backend field
- **WHEN** sandbox adapter manifest lacks `sandbox_backend`
- **THEN** manifest validation fails fast with deterministic missing-field classification

### Requirement: Runtime SHALL enforce sandbox manifest compatibility at activation boundary
Runtime MUST enforce sandbox adapter manifest compatibility before adapter activation, including backend support, host platform compatibility, and session mode compatibility.

#### Scenario: Host platform mismatches manifest declaration
- **WHEN** runtime host platform does not satisfy manifest `host_os` or `host_arch`
- **THEN** adapter activation fails fast with deterministic host-mismatch classification

#### Scenario: Requested session mode is unsupported by manifest
- **WHEN** runtime requests sandbox session mode absent from `session_modes_supported`
- **THEN** adapter activation fails fast with deterministic session-mode-unsupported classification

