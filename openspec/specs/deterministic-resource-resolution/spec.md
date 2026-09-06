# deterministic-resource-resolution Specification

## Purpose
TBD - created by archiving change extension-lifecycle-governance-resource-resolution-contract. Update Purpose after archive.
## Requirements
### Requirement: Resource discovery SHALL be source-scoped and deterministic
The runtime MUST classify discovered resources by source scope and origin, and MUST produce a stable ordered candidate list independent of filesystem enumeration order.

#### Scenario: Same resources are discovered in stable order
- **WHEN** the same project and user resource set is discovered repeatedly
- **THEN** candidate order, identity, and digest inputs are identical

#### Scenario: Unreadable resource is classified deterministically
- **WHEN** a candidate resource cannot be read or parsed
- **THEN** discovery returns a stable warning or blocking classification according to resource policy and continues or stops deterministically

### Requirement: Resource precedence SHALL resolve conflicts without ambiguity
The runtime MUST apply a fixed precedence order for project-explicit, project-auto, user-explicit, user-auto, and bundled/local package-like resources. Duplicate identities MUST yield one selected winner and a bounded conflict record.

#### Scenario: Explicit project resource overrides auto-discovered user resource
- **WHEN** both resources expose the same identity and project-explicit has higher precedence
- **THEN** project-explicit resource is selected and conflict metadata identifies the losing candidate

#### Scenario: Equal-precedence conflict is rejected deterministically
- **WHEN** two candidates with the same identity and precedence cannot be ordered by stable tie-break
- **THEN** runtime rejects the conflict with deterministic classification rather than selecting nondeterministically

### Requirement: Resource resolution SHALL preserve digest and compatibility provenance
The selected resource and any rejected conflict candidates MUST retain bounded source, version, digest, and compatibility provenance for diagnostics and replay.

#### Scenario: Selected resource provenance is queryable
- **WHEN** a resource is activated
- **THEN** diagnostics and replay input include its source scope, origin, version, and digest

#### Scenario: Resource content changes between reloads
- **WHEN** the same resource identity reloads with a different digest
- **THEN** runtime treats it as a new generation candidate and records deterministic content-change metadata

### Requirement: Resource resolution SHALL remain embedded and side-effect bounded
Resource discovery and resolution MUST NOT require a hosted registry, remote control plane, or external package manager, and MUST NOT mutate runtime state before activation admission succeeds.

#### Scenario: Offline resolution
- **WHEN** resource resolution runs without network access
- **THEN** local discovery, precedence, validation, and replay fixture generation complete deterministically

#### Scenario: Denied candidate has no activation side effect
- **WHEN** a candidate fails compatibility, capability, or policy admission
- **THEN** candidate code is not executed and active resource state remains unchanged

### Requirement: Resource resolution SHALL be consistent across Run and Stream
Equivalent Run and Stream inputs MUST use the same discovery, precedence, deduplication, and admission result.

#### Scenario: Run and Stream resolve the same candidate set
- **WHEN** equivalent requests use identical resource roots and configuration snapshots
- **THEN** both paths select equivalent resources and emit equivalent conflict or denial reasons
