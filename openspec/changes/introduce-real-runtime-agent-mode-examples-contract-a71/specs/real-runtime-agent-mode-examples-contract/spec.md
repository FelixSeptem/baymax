## ADDED Requirements

### Requirement: Agent Mode Examples SHALL Implement Real Runtime Semantics
`examples/agent-modes/*/{minimal,production-ish}` MUST execute mode-specific runtime semantics, and MUST NOT rely on a single generic template implementation to represent all modes.

#### Scenario: Mode-specific semantic anchor exists
- **WHEN** maintainers review a mode implementation
- **THEN** the implementation contains at least one mode-specific semantic anchor that can be verified by runtime output or diagnostics evidence

#### Scenario: Generic template-only implementation is rejected
- **WHEN** a mode implementation only provides shared template behavior without mode-specific semantic logic
- **THEN** validation fails and the change is blocked

### Requirement: Agent Mode Examples SHALL Cover Full Mode Matrix With Dual Variants
The repository MUST provide real runtime examples for all required mode families and each mode MUST provide both `minimal` and `production-ish` variants with non-identical semantic behavior.

#### Scenario: Full mode coverage is complete
- **WHEN** matrix coverage validation runs for a71 scope
- **THEN** all required 28 modes are present with `minimal` and `production-ish` entries

#### Scenario: Variant semantic distinction is required
- **WHEN** `minimal` and `production-ish` outputs are compared for a mode
- **THEN** production-ish output includes governance-oriented semantic evidence that is not a no-op copy of minimal output

### Requirement: Agent Mode Runtime Path Evidence SHALL Be Explicit
Each mode variant MUST expose runtime path evidence in execution output, and the path MUST map to the mode's intended runtime domains.

#### Scenario: Runtime path evidence is emitted
- **WHEN** an example variant is executed
- **THEN** output includes explicit runtime path evidence and verification status for that mode

#### Scenario: Missing runtime path evidence blocks acceptance
- **WHEN** runtime output does not contain mode-required path evidence
- **THEN** the example is treated as incomplete and fails acceptance

### Requirement: Agent Mode README SHALL Be Runtime-Synchronized
For every mode variant, README MUST be synchronized with behavior and MUST include `Run`, `Prerequisites`, `Real Runtime Path`, `Expected Output/Verification`, and `Failure/Rollback Notes` sections.

#### Scenario: README is updated with behavior change
- **WHEN** `main.go` behavior changes for a mode variant
- **THEN** the corresponding README is updated in the same change set

#### Scenario: Missing required README sections fails validation
- **WHEN** README validation runs and required sections are absent
- **THEN** validation fails and blocks merge

### Requirement: Agent Mode Contract/Gate/Replay Mapping SHALL Be Auditable
For each mode, mapping among semantic anchors, related contracts, required gates, and replay fixtures MUST be documented and verifiable.

#### Scenario: Mapping matrix stays consistent
- **WHEN** matrix/playbook consistency validation runs
- **THEN** each mode has non-empty and consistent mappings to contract, gate, and replay references

#### Scenario: Mapping drift is blocked
- **WHEN** a mode implementation changes but mapping artifacts are not updated
- **THEN** consistency validation fails and blocks merge
