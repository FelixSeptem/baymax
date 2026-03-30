## ADDED Requirements

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
