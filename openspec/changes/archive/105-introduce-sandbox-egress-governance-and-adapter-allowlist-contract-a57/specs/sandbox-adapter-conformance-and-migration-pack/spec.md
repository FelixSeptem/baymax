## ADDED Requirements

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
