## ADDED Requirements

### Requirement: Optional admission provenance SHALL bind fresh resume facts

Trusted paused bindings and current host facts SHALL support nullable reference-only admission provenance. When present, admission reference and policy revision MUST be nonempty valid UTF-8 references of at most 256 bytes and source scope digest MUST be at most 128 bytes with the existing whitespace/control restrictions. Both sides omitted MUST preserve legacy v1 behavior; partial presence or malformed provenance MUST reject with invalid_binding before claim. Provenance MUST be loaded from trusted host records and fresh authorization facts, never inferred from a client, callback event or shared RunID.

#### Scenario: Legacy host omits provenance

- **WHEN** both paused and current provenance are omitted and all existing checks pass
- **THEN** existing admission behavior is preserved without claiming provenance evidence

#### Scenario: Fresh facts match original admission

- **WHEN** both valid provenance values match and all current authority checks succeed
- **THEN** confirm can proceed through the existing source claim and continuation

#### Scenario: Admission or execution provenance changes

- **WHEN** admission reference changes for any decision, or policy revision or source scope changes for confirm
- **THEN** resume rejects respectively with binding_mismatch, authority_denied or scope_drift before source claim, provider, tool or effect and preserves checkpoint eligibility

#### Scenario: Provenance is missing or invalid on one side

- **WHEN** provenance exists on only one side or contains empty, oversized, invalid UTF-8 or whitespace/control fields
- **THEN** admission rejects with invalid_binding without consuming the checkpoint

#### Scenario: Authorized cancellation survives execution provenance drift

- **WHEN** deny or timeout retains the admission reference and cancellation authority but execution policy or source scope has changed
- **THEN** existing cancellation admission remains available without executing business writes
