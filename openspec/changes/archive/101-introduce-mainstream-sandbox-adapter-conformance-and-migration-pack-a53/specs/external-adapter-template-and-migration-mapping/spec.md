## ADDED Requirements

### Requirement: Repository SHALL provide sandbox adapter onboarding templates for mainstream backends
Adapter template index MUST provide onboarding templates for mainstream sandbox backends:
- `linux_nsjail`
- `linux_bwrap`
- `oci_runtime`
- `windows_job`

Each template MUST include profile declaration snippet, manifest snippet, and conformance command reference.

#### Scenario: Integrator opens sandbox adapter template index
- **WHEN** integrator reads adapter template documentation
- **THEN** mainstream sandbox backend templates are discoverable with executable onboarding snippets

#### Scenario: Integrator follows template for OCI backend
- **WHEN** integrator uses `oci_runtime` onboarding template
- **THEN** template includes complete profile/manifest/conformance snippet chain

### Requirement: Migration mapping SHALL cover legacy wrapper to profile-pack adapter transitions
Migration documentation MUST provide deterministic mapping for legacy sandbox wrapper integrations to canonical profile-pack adapters.

Each mapping entry MUST include:
- previous wrapper pattern,
- target profile-pack pattern,
- compatibility and rollback notes.

#### Scenario: Contributor migrates legacy process wrapper
- **WHEN** contributor follows migration mapping from wrapper-based sandbox integration
- **THEN** resulting configuration aligns with canonical profile-pack adapter contract semantics

#### Scenario: Migration mapping misses rollback guidance
- **WHEN** mapping entry omits rollback notes
- **THEN** documentation contract validation fails

### Requirement: Template and migration docs SHALL remain conformance-linked
Each sandbox adapter template and migration mapping entry MUST link to at least one conformance suite identifier so documentation stays executable and verifiable.

#### Scenario: Maintainer updates sandbox template snippet
- **WHEN** maintainer changes template example
- **THEN** corresponding conformance suite link remains present and valid
