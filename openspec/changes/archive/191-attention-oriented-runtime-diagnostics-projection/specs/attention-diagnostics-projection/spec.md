## Purpose

Expose a bounded, deterministic attention view of supplied diagnostic snapshots
while preserving diagnostic writer and execution ownership boundaries.

## ADDED Requirements

### Requirement: Attention SHALL preserve evidence and execution ownership
The view MUST consume caller-supplied records and reference-only host evidence
without writing diagnostics, reading external stores or invoking execution.
Terminal success and automatic retry MUST NOT imply business-effect completion,
source progress or current authorization.

#### Scenario: Healthy and retrying records
- **WHEN** healthy, terminal-success and automatically retrying records have no actionable evidence
- **THEN** no attention item is inferred and no owner state changes

#### Scenario: Explicit pending host effect
- **WHEN** the host supplies effect-pending evidence alongside terminal success
- **THEN** the view returns reconcile_effect owned by host/effect with the supplied reference

### Requirement: Attention SHALL expose finite stable mappings
ProtocolCapabilityReason exactly protocol.capability.missing_required or
ProviderCapabilityOutcome exactly blocked MUST yield
unsupported/inspect_capability/runtime/capability. The blocked outcome includes
unknown catalog models and MUST NOT imply a specific missing capability.
Credential-only rejection with accepted capability, degraded provider capability
and optional protocol downgrade MUST NOT yield unsupported. Recovery conflict or recorded
stream terminal conflict MUST yield recovery-required/inspect_recovery/runtime/recovery.
Positive gate denied/timeout counts MUST yield gate-failed/inspect_gate/runtime/gate.
Supplemental authorization-required, source-required and effect-pending MUST map
respectively to revalidate_authorization/host/authorization,
inspect_source/host/source and reconcile_effect/host/effect.
Other fields MUST NOT be inferred as these facts.

#### Scenario: Mixed input page
- **WHEN** records contain capability, recovery and gate evidence with source/authorization/effect references
- **THEN** each unique item exposes its fixed code, action, owner and bounded reference

#### Scenario: Real provider admission output
- **WHEN** Composer rejects a missing capability or unknown catalog model in Run or Stream
- **THEN** the recorded blocked outcome yields the same unsupported item, while credential-only rejection does not

### Requirement: Attention SHALL validate atomically and remain bounded
Each input list MUST have at most 200 entries; run IDs MUST be unique in records.
IDs/references MUST have 1..256 ASCII bytes drawn from letters, digits and
dot/underscore/colon/slash/at-sign/hyphen. Supplemental codes MUST be from the
three supported host codes. Zero limit MUST default to 50; other limits MUST be
1..200. Invalid input MUST return a stable error with no partial result or raw echo.

#### Scenario: Bad last entry
- **WHEN** the last entry has an invalid reference or repeated record ID
- **THEN** the entire request fails without partial items

### Requirement: Attention SHALL be deterministic and snapshot-local
Items MUST deduplicate and sort by run ID, code, reference before limiting.
Total MUST count unique candidates and Truncated MUST report omitted candidates.
The view MUST NOT create a cursor or change existing query semantics.

#### Scenario: Input order and limit
- **WHEN** equivalent records and duplicate supplemental evidence arrive in different order
- **THEN** output order and counts are identical, with deterministic truncation

### Requirement: Attention replay SHALL detect semantic drift without execution
Bounded strict replay fixtures MUST compare full projected results and reject
owner/action/classification/count drift as attention.replay_drift. Duplicate,
case-alias and non-ASCII object keys MUST be rejected before typed decoding.

#### Scenario: Wrong expected owner
- **WHEN** a replay fixture expects a different owner
- **THEN** replay fails with attention.replay_drift and invokes no execution
