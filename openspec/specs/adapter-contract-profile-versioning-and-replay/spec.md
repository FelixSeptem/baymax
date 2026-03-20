# adapter-contract-profile-versioning-and-replay Specification

## Purpose
TBD - created by archiving change introduce-adapter-contract-profile-versioning-and-replay-gate-a28. Update Purpose after archive.
## Requirements
### Requirement: Adapter contract SHALL declare profile version
Adapter contract artifacts MUST declare `contract_profile_version` using repository-recognized profile identifiers.

Initial profile baseline for this capability MUST include `v1alpha1`.

#### Scenario: Adapter declares recognized profile
- **WHEN** adapter manifest and contract artifacts declare `contract_profile_version=v1alpha1`
- **THEN** profile parsing and contract loading succeed

#### Scenario: Adapter declares unknown profile
- **WHEN** adapter declares unsupported or malformed `contract_profile_version`
- **THEN** contract loading fails fast with deterministic profile-version classification

### Requirement: Runtime SHALL enforce profile compatibility window
Runtime MUST enforce profile compatibility window with default policy `current + previous`.

Profiles outside supported window MUST fail fast.

#### Scenario: Adapter profile within supported window
- **WHEN** adapter profile is current or previous supported profile
- **THEN** runtime continues contract validation flow

#### Scenario: Adapter profile outside supported window
- **WHEN** adapter profile is older than previous or newer than supported current
- **THEN** runtime fails fast with profile-compatibility mismatch classification

### Requirement: Repository SHALL provide deterministic adapter contract replay baseline
Repository MUST maintain versioned replay fixtures for adapter contract behaviors, including:
- manifest compatibility outcomes,
- negotiation/fallback outcomes,
- reason taxonomy outputs.

Replay execution MUST be deterministic and offline.

#### Scenario: Contract replay run matches baseline
- **WHEN** replay command executes against current fixtures
- **THEN** replay passes without drift classification

#### Scenario: Contract replay run diverges from baseline
- **WHEN** replay output differs from fixture expectations
- **THEN** replay fails fast with explicit drift classification and non-zero status

