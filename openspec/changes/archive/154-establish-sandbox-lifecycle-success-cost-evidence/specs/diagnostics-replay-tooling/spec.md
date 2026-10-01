## ADDED Requirements

### Requirement: Replay tooling SHALL validate sandbox lifecycle-success-cost evidence fixtures
Diagnostics replay tooling MUST support the versioned fixture contract `sandbox_lifecycle_success_cost_evidence.v1` and MUST evaluate it offline and deterministically.

Fixture validation MUST cover canonical lifecycle phase ordering, cold launch versus per-session reuse versus recovery or resume, bounded successful-task cost summary, terminal outcome accounting, verdict, reason ordering, canonical digest, Run/Stream parity, and privacy-safe field rejection.

Replay drift classes MUST include at minimum:
- `sandbox_lifecycle_phase_drift`
- `sandbox_lifecycle_continuity_drift`
- `sandbox_success_cost_bucket_drift`
- `sandbox_lifecycle_baseline_drift`
- `sandbox_lifecycle_terminal_outcome_drift`
- `sandbox_lifecycle_run_stream_parity_drift`
- `sandbox_lifecycle_privacy_drift`

#### Scenario: Lifecycle-success-cost fixture matches canonical evidence
- **WHEN** replay tooling evaluates a valid `sandbox_lifecycle_success_cost_evidence.v1` fixture whose actual normalized evidence matches the expected projection
- **THEN** it returns a deterministic pass result without invoking a sandbox, host, network service, or provider

#### Scenario: Fixture contains prohibited evidence
- **WHEN** a lifecycle-success-cost fixture includes command-derived, credential, filesystem, endpoint, or raw output data
- **THEN** replay fails with deterministic `sandbox_lifecycle_privacy_drift` or schema classification and emits no normalized prohibited value

### Requirement: Lifecycle-success-cost replay support SHALL preserve mixed-fixture compatibility
Adding `sandbox_lifecycle_success_cost_evidence.v1` support MUST NOT break validation of historical fixture schemas. Replaying identical lifecycle-success-cost evidence repeatedly MUST be idempotent, and unknown additive fields that do not alter v1 required facts MUST be ignored safely.

#### Scenario: Historical and lifecycle-success-cost fixtures run together
- **WHEN** the replay gate executes archived fixtures together with valid `sandbox_lifecycle_success_cost_evidence.v1` fixtures
- **THEN** every supported fixture generation remains deterministic and no legacy parser regression is introduced

#### Scenario: Lifecycle evidence is replayed twice
- **WHEN** the same valid lifecycle-success-cost fixture is evaluated twice
- **THEN** both results have equivalent normalized output, verdict, reason ordering, and canonical digest
