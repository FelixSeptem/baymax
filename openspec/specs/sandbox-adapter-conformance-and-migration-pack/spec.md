# sandbox-adapter-conformance-and-migration-pack Specification

## Purpose
TBD - created by archiving change introduce-mainstream-sandbox-adapter-conformance-and-migration-pack-a53. Update Purpose after archive.
## Requirements
### Requirement: Repository SHALL define mainstream sandbox adapter profile pack
Repository MUST define canonical mainstream sandbox adapter profiles for:
- `linux_nsjail`
- `linux_bwrap`
- `oci_runtime`
- `windows_job`

Each profile MUST declare:
- backend identifier,
- minimal required capability set,
- supported session modes (`per_call|per_session`),
- host platform constraints.

#### Scenario: Maintainer inspects profile-pack definitions
- **WHEN** maintainer reviews version-controlled sandbox adapter profile pack
- **THEN** all four canonical backend profiles are present with deterministic schema fields

#### Scenario: Contributor declares unknown backend profile
- **WHEN** adapter profile references unsupported backend identifier
- **THEN** validation fails fast with deterministic profile-unknown classification

### Requirement: Sandbox adapter onboarding SHALL be profile-driven and deterministic
Sandbox adapter onboarding MUST resolve adapter behavior from profile-pack definition rather than backend-specific ad hoc flags.

Equivalent profile input MUST yield semantically equivalent normalized behavior contract output.

#### Scenario: Adapter onboarding uses canonical profile id
- **WHEN** contributor configures adapter using a canonical profile-pack identifier
- **THEN** runtime and conformance tooling resolve deterministic backend/capability/session semantics

#### Scenario: Equivalent profile input is loaded repeatedly
- **WHEN** profile-pack input remains unchanged across repeated load operations
- **THEN** resolved onboarding semantics remain semantically equivalent without drift

### Requirement: Migration mapping SHALL preserve canonical sandbox reason and capability taxonomy
Migration from legacy sandbox wrappers to profile-pack adapters MUST preserve canonical capability and reason taxonomy semantics.

#### Scenario: Legacy wrapper migrates to profile-pack adapter
- **WHEN** integrator applies documented migration mapping for a legacy wrapper
- **THEN** resulting adapter emits canonical capability and reason taxonomy compatible with contract assertions

#### Scenario: Migration introduces non-canonical reason namespace
- **WHEN** migrated adapter emits reason codes outside approved canonical namespace
- **THEN** conformance validation fails with taxonomy-drift classification

### Requirement: Run and Stream integration SHALL preserve profile-pack semantic equivalence
For equivalent request and effective profile-pack configuration, Run and Stream integrations MUST preserve semantically equivalent sandbox adapter contract outcomes.

#### Scenario: Equivalent request under same profile in Run and Stream
- **WHEN** equivalent request is executed through Run and Stream with same profile-pack adapter
- **THEN** both paths produce semantically equivalent backend-profile contract classification

### Requirement: Sandbox adapter conformance SHALL include egress policy matrix coverage
Sandbox adapter conformance suites MUST validate canonical egress behavior across supported backend/profile matrix.

Coverage MUST include:
- deny path
- allow path
- allow-and-record path
- selector override precedence

#### Scenario: Backend matrix validates egress deny behavior
- **WHEN** conformance suite executes deny-case fixtures on supported backend profiles
- **THEN** all backends return canonical egress deny classification

#### Scenario: Backend matrix validates selector override precedence
- **WHEN** fixtures define both global and selector-specific egress rules
- **THEN** conformance assertions confirm selector override precedence deterministically

### Requirement: Sandbox migration mapping SHALL include egress and allowlist onboarding guidance
Migration documentation and template pack MUST include explicit mapping for:
- legacy unrestricted network behavior to egress policy contract,
- legacy adapter activation rules to allowlist contract.

#### Scenario: Maintainer reviews migration mapping for sandbox adapters
- **WHEN** migration docs are inspected for A57
- **THEN** egress/allowlist migration entries include compatibility notes rollback notes and conformance suite ids

#### Scenario: Template onboarding references new gate scripts
- **WHEN** integrator uses sandbox adapter onboarding template
- **THEN** template references A57 gate commands and required fixture suites

