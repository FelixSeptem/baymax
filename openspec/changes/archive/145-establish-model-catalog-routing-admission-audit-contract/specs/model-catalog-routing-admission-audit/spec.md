## Purpose

This capability defines a bounded, versioned, replayable contract for evaluating host-supplied provider/model candidates against existing capability, credential, fallback, readiness, and Run/Stream admission semantics without introducing model discovery or a hosted routing control plane.

## ADDED Requirements

### Requirement: Catalog routing audit SHALL be versioned, bounded, and provider-neutral

The system MUST accept a versioned `model_catalog_routing_admission.v1` audit input containing a host-supplied catalog generation, bounded candidate identities, requested required and optional capabilities, credential evidence statuses, readiness policy, and expected admission facts. The normalized representation MUST use stable identities, enums, ordinals, bounded reason codes, and digests or lengths for text-like values. It MUST NOT contain provider SDK types, endpoints, credential material, raw provider responses, prompts, or unbounded bodies.

#### Scenario: Equivalent candidate input normalizes deterministically
- **WHEN** the same catalog generation, candidates, capabilities, evidence statuses, and policy are supplied in equivalent ordering
- **THEN** normalization produces the same canonical candidate order, digest, and audit identity

#### Scenario: Audit input exceeds a declared bound
- **WHEN** candidate count, capability count, reason count, identity length, or serialized audit size exceeds its bound
- **THEN** validation fails fast with a stable overflow classification and returns no partial audit result

### Requirement: Candidate identities SHALL be normalized and ambiguity SHALL fail fast

Candidate provider/model identities MUST be normalized using the existing catalog identity rules. Duplicate normalized identities, conflicting candidate priority metadata, self-referential fallback declarations, and ambiguous equal-ranked candidates MUST fail deterministically. Validation MUST NOT apply last-write-wins behavior.

#### Scenario: Duplicate candidate identity is rejected
- **WHEN** two supplied candidates normalize to the same provider/model identity
- **THEN** the audit fails with a duplicate-candidate classification before capability or credential evaluation

#### Scenario: Equal-ranked candidates are ambiguous
- **WHEN** two otherwise admissible candidates have equal deterministic ranking inputs and no stable tie-break remains
- **THEN** the audit fails with an ambiguous-selection classification and selects no model

### Requirement: Candidate admission SHALL reuse existing capability, credential, fallback, and readiness semantics

Candidate evaluation MUST preserve the existing required/optional capability negotiation strategy, credential evidence vocabulary, declared fallback rules, and strict/non-strict readiness mapping. A missing required capability, missing or invalid credential evidence, or blocked readiness finding MUST prevent selection. An optional capability downgrade MAY select only an independently admissible declared fallback and MUST preserve the original downgrade reason.

#### Scenario: Required capability gap blocks every candidate
- **WHEN** all supplied candidates lack a required capability
- **THEN** the audit returns blocked admission with ordered capability reasons and no provider action

#### Scenario: Optional gap selects an independently admissible fallback
- **WHEN** a candidate has an optional capability gap and its declared fallback independently passes capability, credential, and readiness checks
- **THEN** the audit returns degraded admission with the original optional-gap reason and normalized fallback identity

#### Scenario: Unverified evidence follows readiness strictness
- **WHEN** a candidate has unverified credential evidence
- **THEN** strict readiness blocks it and non-strict readiness records degraded status using the existing credential reason vocabulary

### Requirement: Conditional candidate resolution SHALL be pure and deterministic

If audit evidence activates candidate resolution, selection MUST consume only the supplied catalog snapshot, candidates, request capabilities, credential evidence, and policy. Equivalent inputs MUST select the same identity and ordered skip reasons. Resolution MUST perform no network, clock, filesystem, provider, credential-probe, runtime mutation, or diagnostic write side effects.

The initial audit activates this condition when multiple independently admissible
host-supplied candidates are observed: exact-identity admission reports the
expressiveness gap as ambiguous rather than selecting by caller order, and the
resolver becomes the explicit opt-in selection path.

#### Scenario: Admissible candidates produce stable selection
- **WHEN** two or more candidates independently satisfy the request and have distinct deterministic ranking inputs
- **THEN** the resolver selects the highest-ranked candidate and emits stable skip reasons for candidates not selected

#### Scenario: Resolver cannot express a candidate safely
- **WHEN** a candidate requires an unsupported or ambiguous representation
- **THEN** resolution fails before provider invocation with a stable request-shape or ambiguous-selection classification and does not fall back to free-form text

#### Scenario: Audit records exact-identity expressiveness gap
- **WHEN** two host-supplied candidates independently pass existing exact-identity admission
- **THEN** the audit returns `ambiguous_selection` with no selected identity, and an explicitly invoked resolver may select the highest deterministic priority while retaining bounded skip reasons for the other candidate

### Requirement: Catalog generation and Run/Stream parity SHALL be preserved

Audit and conditional resolution results MUST retain the catalog generation, selected identity, fallback identity, capability outcome, credential status, readiness status, and ordered reason sequence. Equivalent Run and Stream inputs MUST produce semantically equivalent selection and admission facts. A later valid catalog generation MUST NOT alter an already admitted result.

#### Scenario: Run and Stream select equivalently
- **WHEN** Run and Stream receive equivalent catalog, candidate, capability, credential, and policy inputs
- **THEN** both paths produce equivalent selected identity, fallback identity, status, generation, and ordered reasons

#### Scenario: Catalog reload occurs after admission
- **WHEN** a later valid catalog generation is published after an audit or admission result is produced
- **THEN** the existing result retains its original generation and selection facts while later evaluations use the new generation

### Requirement: Audit replay SHALL detect semantic drift and remain backward compatible

The replay tool MUST validate successful selection, no-candidate blocked outcomes, capability denial, credential denial/degradation, fallback, ambiguous selection, invalid generation rollback, and Run/Stream parity. Replaying the same fixture twice MUST produce identical canonical digest and classifications. Historical fixtures without optional resolver fields MUST remain readable with documented defaults, and unknown fields MUST be ignored safely.

#### Scenario: Repeated replay is idempotent
- **WHEN** one valid audit fixture is replayed twice
- **THEN** canonical selection digest, status, reason order, and classification are identical

#### Scenario: Historical fixture omits conditional resolver fields
- **WHEN** a pre-resolver fixture contains only exact-identity admission facts
- **THEN** replay succeeds using compatibility defaults and does not invent a selection drift

#### Scenario: Expected and observed selection differ
- **WHEN** observed candidate selection, generation, reason order, or Run/Stream parity differs from the fixture expectation
- **THEN** replay fails with a stable drift classification that identifies the differing fact

### Requirement: Candidate audit SHALL not introduce discovery or a parallel control plane

The capability MUST use only host-supplied catalog and evidence. It MUST NOT perform remote discovery, background refresh, credential storage or probing, create a global mutable registry/router, add provider SDK dependencies to shared/runtime packages, or define a second readiness or terminal-state machine.

#### Scenario: Discovery-only input is supplied
- **WHEN** an audit request lacks a host-supplied catalog generation and attempts to request remote discovery
- **THEN** validation fails with a bounded unsupported-source classification and performs no network action

#### Scenario: Shared package attempts to persist provider-specific payload
- **WHEN** a boundary check observes provider SDK types, credential payloads, endpoint data, or raw response bodies in the audit path
- **THEN** the contribution gate rejects the change before merge
