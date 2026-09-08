## ADDED Requirements

### Requirement: Host-injected provider-model capability catalog
The system SHALL accept a bounded, immutable provider-model capability catalog supplied by runtime configuration or an embedding host. Each descriptor MUST contain a normalized provider and model identity, a host-chosen catalog version, a positive context-window token count, and a sorted, deduplicated declared capability set. Validation MUST deterministically reject unknown descriptors, duplicate normalized identities, malformed capabilities, nonpositive context windows, and conflicting fallback declarations with bounded canonical reason codes. The catalog MUST NOT perform remote discovery, background refresh, or use a global mutable registry.

#### Scenario: A valid descriptor is admitted into the catalog
- **WHEN** a host supplies one descriptor with normalized identity, a positive context window, and a deduplicated capability set
- **THEN** the system stores an immutable descriptor under the catalog version and exposes its stable identity to admission evaluation

#### Scenario: Duplicate normalized descriptors are rejected
- **WHEN** a catalog candidate contains two descriptors that normalize to the same provider and model identity
- **THEN** the system rejects the candidate with the canonical duplicate-descriptor reason and does not publish the candidate

### Requirement: Capability admission reuses adapter negotiation
The system SHALL construct provider-model admission from `adapter/capability` required and optional capability negotiation. A missing required capability MUST block admission before any provider action. An optional capability gap MUST preserve the negotiated downgrade finding and MAY select a fallback only when the selected descriptor declares that fallback and the fallback independently passes catalog lookup, required-capability negotiation, and credential preflight. The evaluator MUST keep identity, capability order, fallback selection, and reason order deterministic.

#### Scenario: Missing required capability blocks before dispatch
- **WHEN** a request requires a capability absent from the selected descriptor
- **THEN** the evaluator returns a blocked admission with the adapter canonical required-capability reason and invokes no provider action

#### Scenario: Declared fallback handles an optional capability gap
- **WHEN** a request has an optional capability gap and the selected descriptor declares an eligible fallback that passes its own admission checks
- **THEN** the evaluator returns a degraded admission with the original optional-gap finding and the normalized fallback identity

### Requirement: Credential evidence is redacted and policy-driven
The system SHALL accept host-supplied `CredentialEvidence` containing only normalized provider identity, a status of `available`, `missing`, `invalid`, or `unverified`, and a bounded canonical reason code. Credential material, endpoints, raw provider responses, and validation payloads MUST NOT be represented in configuration, catalog values, diagnostics, events, or replay fixtures. `missing` and `invalid` evidence MUST block required use. `unverified` evidence MUST use the configured strict or non-strict readiness policy without introducing a separate status vocabulary.

#### Scenario: Missing credential evidence blocks a selected provider
- **WHEN** the selected descriptor has credential evidence with status `missing`
- **THEN** the evaluator returns a blocked admission with a canonical credential reason and records no credential material

#### Scenario: Unverified evidence follows strict-mode policy
- **WHEN** credential evidence is `unverified` for an otherwise admissible descriptor
- **THEN** the evaluator returns the existing strict-mode mapped readiness outcome and preserves the evidence status and reason code

### Requirement: Admissions use immutable catalog generation snapshots
The system SHALL snapshot the catalog version, selected descriptor, negotiated capability outcome, credential evidence status, fallback identity, and canonical reason codes at Run or Stream admission. A valid catalog reload MUST affect only later admissions. An invalid reload MUST retain the last valid catalog atomically, and no admitted Run or Stream MUST observe mixed catalog generations.

#### Scenario: In-flight stream retains its admission generation
- **WHEN** a Stream is admitted under catalog generation `v1` and a valid `v2` catalog is published while it is active
- **THEN** all recorded admission facts for that Stream remain from `v1` while later admissions evaluate against `v2`

#### Scenario: Invalid hot reload retains the active catalog
- **WHEN** a hot reload candidate fails catalog validation
- **THEN** the system reports the canonical reload-rollback finding and later admissions continue to use the complete preceding catalog generation

### Requirement: Provider-model admission has replayable parity coverage
The system SHALL provide deterministic, versioned replay fixtures and contract coverage for successful admission, unknown model, required capability denial, optional fallback, missing credential, unverified credential, invalid reload rollback, and equivalent Run and Stream inputs. Fixture parsing, finding order, and replay output MUST be bounded, stable, and drift-detectable.

#### Scenario: Equivalent Run and Stream inputs replay identically
- **WHEN** a replay fixture supplies the same catalog generation, credential evidence, request capabilities, and runtime policy to Run and Stream admission
- **THEN** both paths produce equivalent normalized admission status, selected identity, fallback identity, and canonical finding sequence
