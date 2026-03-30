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

### Requirement: Adapter contract profile versioning SHALL include sandbox profile-pack track
Adapter contract profile versioning MUST include sandbox profile-pack replay track for mainstream sandbox adapter contracts.

Initial sandbox profile-pack replay identifier for this milestone MUST include `sandbox.v1`.

#### Scenario: Adapter replay uses recognized sandbox profile version
- **WHEN** replay fixtures declare `contract_profile_version=sandbox.v1`
- **THEN** replay loader accepts fixture profile and executes sandbox contract assertions

#### Scenario: Adapter replay uses unsupported sandbox profile version
- **WHEN** replay fixtures declare unknown sandbox profile version
- **THEN** replay loader fails fast with deterministic profile-version classification

### Requirement: Replay baseline SHALL include sandbox backend profile fixtures and drift classes
Repository MUST maintain deterministic offline fixtures for sandbox adapter profile-pack replay.

Drift classes MUST include at minimum:
- `sandbox_backend_profile_drift`
- `sandbox_manifest_compat_drift`
- `sandbox_session_mode_drift`

#### Scenario: Sandbox replay fixture matches canonical baseline
- **WHEN** sandbox adapter replay runs with expected fixture output
- **THEN** replay validation succeeds without drift classification

#### Scenario: Sandbox replay fixture detects manifest compatibility drift
- **WHEN** replay output diverges on manifest compatibility semantics
- **THEN** replay validation fails with deterministic `sandbox_manifest_compat_drift` classification

### Requirement: Sandbox profile-pack replay SHALL preserve backward compatibility with existing adapter profile tracks
Adding sandbox profile-pack replay track MUST NOT break validation of existing adapter profile replay tracks.

#### Scenario: Existing profile fixtures and sandbox fixtures run together
- **WHEN** replay gate executes mixed suites for existing profiles and `sandbox.v1`
- **THEN** all suites are parsed and validated deterministically without cross-profile regression

