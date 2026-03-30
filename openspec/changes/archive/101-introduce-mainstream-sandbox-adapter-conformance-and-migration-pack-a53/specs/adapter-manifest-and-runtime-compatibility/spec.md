## ADDED Requirements

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
