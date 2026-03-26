## ADDED Requirements

### Requirement: Arbitration explainability SHALL expose bounded secondary reasons deterministically
Runtime arbitration explainability MUST expose bounded secondary reasons with deterministic ordering and dedup semantics.

Minimum rules:
- maximum secondary reason count is bounded,
- secondary reasons are deduplicated by canonical code,
- output order is deterministic for equivalent inputs.

#### Scenario: Multiple secondary candidates exist
- **WHEN** arbitration evaluates more than one non-primary eligible candidate
- **THEN** runtime emits bounded and deterministically ordered secondary reason codes

#### Scenario: Duplicate secondary candidate codes appear
- **WHEN** equivalent candidates produce duplicated canonical codes
- **THEN** runtime emits each canonical code at most once in secondary output

### Requirement: Explainability output SHALL include machine-readable rule version and remediation hint
Runtime MUST include machine-readable explainability metadata with rule version and remediation hint taxonomy.

Minimum required fields:
- `runtime_arbitration_rule_version`
- `runtime_remediation_hint_code`
- `runtime_remediation_hint_domain`

#### Scenario: Explainability payload is generated
- **WHEN** runtime emits arbitration explainability output
- **THEN** payload includes canonical rule-version and remediation-hint fields

#### Scenario: Unsupported hint taxonomy is produced
- **WHEN** explainability output contains non-canonical remediation hint code
- **THEN** contract validation fails and blocks merge
