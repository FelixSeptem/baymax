## ADDED Requirements

### Requirement: Conformance harness SHALL include mainstream sandbox backend matrix suites
External adapter conformance harness MUST include sandbox backend matrix suites for:
- `linux_nsjail`
- `linux_bwrap`
- `oci_runtime`
- `windows_job` (when Windows runner is available)

Platform-unavailable backend suites MAY be skipped only with deterministic skip classification.

#### Scenario: Linux runner executes sandbox backend matrix
- **WHEN** conformance harness runs on Linux environment
- **THEN** harness executes `linux_nsjail`, `linux_bwrap`, and `oci_runtime` suites with deterministic results

#### Scenario: Windows runner executes windows-job suite
- **WHEN** conformance harness runs on Windows environment
- **THEN** harness executes `windows_job` suite with deterministic contract assertions

### Requirement: Harness SHALL validate sandbox capability negotiation and session lifecycle semantics
Sandbox adapter conformance harness MUST validate:
- required capability missing fail-fast behavior,
- optional capability downgrade behavior,
- `per_call|per_session` lifecycle semantics,
- crash/reconnect/close-idempotent semantics for session lifecycle.

#### Scenario: Required capability is missing for selected backend profile
- **WHEN** harness executes adapter with unsatisfied required capability
- **THEN** suite fails with deterministic missing-required-capability classification

#### Scenario: Per-session lifecycle close is repeated
- **WHEN** harness invokes close repeatedly on same sandbox session
- **THEN** suite verifies idempotent close semantics without duplicate terminal side effects

### Requirement: Harness SHALL classify sandbox adapter drift using canonical classes
Sandbox adapter conformance harness MUST emit deterministic drift classes at minimum:
- `sandbox_backend_profile_drift`
- `sandbox_capability_claim_drift`
- `sandbox_session_lifecycle_drift`
- `sandbox_reason_taxonomy_drift`

#### Scenario: Backend profile mapping drifts from canonical fixture
- **WHEN** adapter backend/profile mapping output diverges from fixture expectation
- **THEN** harness fails with deterministic `sandbox_backend_profile_drift` classification
