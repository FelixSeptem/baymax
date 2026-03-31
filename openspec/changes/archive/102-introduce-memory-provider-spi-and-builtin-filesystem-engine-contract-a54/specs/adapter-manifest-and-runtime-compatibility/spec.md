## ADDED Requirements

### Requirement: Adapter manifest SHALL declare memory provider profile and contract fields
Adapter manifest schema MUST support memory-specific declarations with at minimum:
- `memory.provider`
- `memory.profile`
- `memory.contract_version`
- `memory.operations.required`
- `memory.operations.optional`
- `memory.fallback.supported`

Missing required memory manifest fields MUST fail validation before adapter activation.

#### Scenario: Memory manifest omits contract version
- **WHEN** adapter manifest declares memory integration without `memory.contract_version`
- **THEN** manifest validation fails fast with deterministic missing-field classification

#### Scenario: Memory manifest includes required fields
- **WHEN** adapter manifest provides all required memory declarations
- **THEN** validation proceeds to runtime compatibility checks

### Requirement: Runtime SHALL enforce manifest compatibility with effective memory mode and profile
Runtime MUST validate memory manifest compatibility against effective runtime config before activating adapter.

Compatibility checks MUST include:
- mode compatibility (`external_spi|builtin_filesystem`),
- profile compatibility,
- contract version compatibility window.

#### Scenario: Runtime mode profile and manifest are compatible
- **WHEN** runtime effective memory mode and selected profile satisfy manifest declarations
- **THEN** adapter activation proceeds to conformance or execution stage

#### Scenario: Manifest profile mismatches runtime selection
- **WHEN** manifest declares memory profile different from effective runtime profile
- **THEN** activation fails fast with deterministic profile-mismatch classification

### Requirement: Memory operation capability semantics SHALL be deterministic for required and optional sets
Memory operation capabilities declared in manifest MUST preserve deterministic semantics:
- missing required operation MUST fail fast,
- missing optional operation MAY downgrade with canonical downgrade reason.

#### Scenario: Required delete operation is unavailable
- **WHEN** manifest declares `Delete` as required but adapter cannot satisfy it
- **THEN** runtime blocks activation with required-operation-missing classification

#### Scenario: Optional metadata filter capability is unavailable
- **WHEN** optional filter capability is unavailable for selected profile
- **THEN** runtime allows activation with deterministic downgrade classification and observability record
