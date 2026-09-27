## Purpose

This capability provides a bounded, reference-only, deterministic projection and replay contract for tracing capability assets across identity, version, ownership, scope, dependencies, consumers, drift, withdrawal impact, and compatible replacement evidence without changing runtime authority.

## ADDED Requirements

### Requirement: Capability asset provenance SHALL use a versioned bounded projection

The audit SHALL accept a versioned `capability_asset_provenance.v1` projection containing a bounded asset kind, stable asset identity, optional version and digest, owner, scope, source reference, verification metadata, and bounded dependency and consumer references. Required identity, kind, owner, and scope fields MUST be validated before audit output is produced.

#### Scenario: Complete asset projection is accepted
- **WHEN** an input contains a supported projection version and valid bounded identity, kind, owner, scope, source, and reference fields
- **THEN** the audit produces a normalized provenance projection with a deterministic projection digest

#### Scenario: Required provenance field is missing
- **WHEN** an input omits a required identity, kind, owner, or scope field
- **THEN** the audit fails with a deterministic malformed-provenance classification and emits no accepted projection

### Requirement: Provenance normalization SHALL be deterministic and additive-compatible

Normalization MUST validate field syntax, deduplicate and sort dependency and consumer references by canonical identity, and apply documented nullable/default values for omitted optional fields. Unknown additive fields MUST be ignored safely. Equivalent inputs with different source ordering MUST normalize to the same projection digest.

#### Scenario: Equivalent reference order is normalized
- **WHEN** two valid inputs contain the same references in different orders and with duplicate entries
- **THEN** both inputs produce equivalent normalized references and the same projection digest

#### Scenario: Unknown additive field is present
- **WHEN** a valid input contains an unknown non-required field
- **THEN** normalization succeeds without changing known-field semantics or the projection digest

### Requirement: Provenance references SHALL enforce bounded integrity and scope semantics

Dependency and consumer references MUST contain bounded identifiers or digests and MUST resolve to the declared asset identity space. A reference that crosses an undeclared scope, exceeds a configured bound, or contains body-bearing/unbounded data MUST produce a deterministic reference-integrity or scope-violation finding.

#### Scenario: Cross-scope consumer reference is detected
- **WHEN** an asset declares a consumer reference outside its allowed scope
- **THEN** the audit emits a scope-violation finding and does not treat the reference as an authorized consumer

#### Scenario: Unbounded reference payload is rejected
- **WHEN** a dependency or consumer reference contains raw content, credentials, reasoning, or exceeds the configured size limit
- **THEN** the audit rejects the reference with a bounded payload classification

### Requirement: Provenance drift and dependency conflicts SHALL be explicit findings

The audit MUST compare expected and observed identity, version, digest, owner, scope, dependency, and consumer metadata using stable finding categories. It MUST distinguish missing evidence, malformed metadata, declared drift, and conflicting duplicate identity rather than silently selecting a last writer.

#### Scenario: Version or digest drift is replayed
- **WHEN** expected and observed projections share an identity but differ in version or digest
- **THEN** replay emits a deterministic provenance-drift finding containing bounded expected and observed references

#### Scenario: Conflicting duplicate identity is supplied
- **WHEN** two projections claim the same identity and scope but have incompatible owner or dependency metadata
- **THEN** the audit emits a deterministic identity-conflict finding and does not apply last-write-wins behavior

### Requirement: Withdrawal impact SHALL be reference-based and incomplete evidence SHALL be visible

Given a withdrawal request and a complete normalized dependency/consumer graph, the audit MUST return the bounded affected consumer and dependent set plus a deterministic impact digest. If required references are absent or contradictory, the audit MUST report incomplete or conflicting evidence and MUST NOT claim a complete impact set.

#### Scenario: Complete withdrawal graph produces impact
- **WHEN** a withdrawn asset has valid dependency and consumer references with no graph conflict
- **THEN** the audit returns the sorted bounded affected set and stable withdrawal-impact digest without changing runtime state

#### Scenario: Missing consumer evidence prevents completeness claim
- **WHEN** a withdrawn asset lacks required consumer references or contains contradictory graph edges
- **THEN** the audit reports impact-incomplete or impact-conflict evidence and does not mark the affected set complete

### Requirement: Replacement compatibility SHALL be advisory and deterministic

The audit MAY compare a withdrawn asset with a proposed replacement, but compatibility output MUST remain an advisory finding based on identity, an optional bounded version range, compatible digest set, scope, dependency, and declared capability metadata. The audit MUST NOT activate, withdraw, or mutate either asset.

#### Scenario: Compatible replacement is identified
- **WHEN** a replacement preserves the required identity relationship, supported version range, scope, dependencies, and declared capabilities
- **THEN** the audit emits a bounded compatible-replacement finding with stable evidence references

#### Scenario: Incompatible replacement is identified
- **WHEN** a replacement changes a required identity relationship, violates scope, removes a required dependency, or falls outside compatibility metadata
- **THEN** the audit emits a deterministic replacement-incompatible finding and performs no activation or rollback action

### Requirement: Provenance audit SHALL be replayable and Run/Stream equivalent

The repository MUST provide versioned offline fixtures and replay that exercise valid, malformed, drifted, scope-violating, withdrawal, and replacement cases. Replaying the same fixture MUST be idempotent and produce the same normalized output and finding digest. Equivalent Run and Stream envelopes MUST produce equivalent audit results.

#### Scenario: Repeated replay is idempotent
- **WHEN** the same provenance fixture is replayed multiple times
- **THEN** normalized output, finding order, and evidence digests remain identical and no state is mutated

#### Scenario: Run and Stream envelopes are equivalent
- **WHEN** equivalent asset inputs are evaluated through Run and Stream audit entry points
- **THEN** both paths produce equivalent projection, findings, and impact completeness semantics

### Requirement: Provenance audit SHALL preserve privacy and ownership boundaries

Projection, fixture, replay, and diagnostics outputs MUST contain only bounded identifiers, digests, enums, and references. They MUST NOT contain raw prompt/transcript/reasoning, credential material, workspace content, or a second authoritative asset store. The audit MUST not write lifecycle events through a path other than existing owner-specific observability contracts.

#### Scenario: Sensitive or body-bearing metadata is supplied
- **WHEN** an input contains credentials, raw reasoning, workspace content, or unbounded body data
- **THEN** validation rejects or redacts it with a deterministic privacy-boundary classification and emits no sensitive projection

#### Scenario: Audit output is consumed by runtime
- **WHEN** a runtime owner reads an audit result
- **THEN** the result remains advisory/reference-only and does not alter activation, policy, memory, context, provider, or terminal outcome semantics
