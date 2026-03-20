## ADDED Requirements

### Requirement: Adapter manifest SHALL include contract profile version field
Adapter manifest contract MUST include `contract_profile_version`.

This field MUST be validated together with manifest compatibility checks before adapter activation.

#### Scenario: Manifest omits contract profile version
- **WHEN** adapter manifest is missing `contract_profile_version`
- **THEN** manifest validation fails fast before adapter activation

#### Scenario: Manifest profile and baymax compatibility both valid
- **WHEN** manifest passes both `contract_profile_version` and `baymax_compat` checks
- **THEN** adapter activation may proceed to negotiation stage
