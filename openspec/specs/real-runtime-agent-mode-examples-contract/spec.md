# real-runtime-agent-mode-examples-contract Specification

## Purpose
TBD - created by archiving change introduce-real-runtime-agent-mode-examples-contract-a71. Update Purpose after archive.
## Requirements
### Requirement: Agent Mode Examples SHALL Implement Real Runtime Semantics
`examples/agent-modes/*/{minimal,production-ish}` MUST implement mode-specific business semantics and MUST execute real runtime behavior aligned to that mode's domain intent.

Shared helper code MAY be reused for non-semantic concerns (logging, output formatting, common test harness), but shared code MUST NOT host mode business decision logic, mode semantic markers, or mode-specific state-transition logic.

#### Scenario: Every mode implements its own business semantic path
- **WHEN** maintainers review a mode implementation
- **THEN** the mode contains explicit business-semantic logic for that mode (not only generic dispatch/wrapper calls)
- **AND** runtime output/diagnostics evidence can be traced back to mode-owned semantic logic

#### Scenario: Mode-specific semantic anchor exists
- **WHEN** maintainers review a mode implementation
- **THEN** the implementation contains at least one mode-specific semantic anchor that can be verified by runtime output or diagnostics evidence

#### Scenario: Generic template-only implementation is rejected
- **WHEN** a mode implementation only provides shared template behavior without mode-specific semantic logic
- **THEN** validation fails and the change is blocked

#### Scenario: Shared semantic engine is rejected
- **WHEN** mode business semantics are centralized in a shared generic engine that multiple modes only parameterize
- **THEN** validation fails and the change is blocked

### Requirement: Semantic Ownership SHALL Be Per-Mode
Each mode MUST own its business-semantic implementation in mode-scoped code paths, and MUST NOT outsource semantic behavior to a single cross-mode semantic engine.

#### Scenario: Mode-scoped semantic ownership is enforced
- **WHEN** semantic ownership validation runs for all required modes
- **THEN** each mode has mode-scoped semantic implementation and evidence mapping

#### Scenario: Wrapper-only mode entrypoint fails ownership validation
- **WHEN** a mode entrypoint only calls a shared semantic executor with pattern/variant parameters and no mode-scoped business logic
- **THEN** ownership validation fails and blocks merge

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

### Requirement: Real Runtime Agent Modes SHALL Enforce Doc-First Replacement Order
For mode replacement work, maintainers MUST update mode-level documentation artifacts before implementing or refactoring mode semantic code.

Required artifacts include:
- `examples/agent-modes/MATRIX.md`
- `examples/agent-modes/*/{minimal,production-ish}/README.md`

#### Scenario: Documentation baseline precedes mode code replacement
- **WHEN** a mode is scheduled for semantic replacement
- **THEN** matrix and mode readme baselines are updated first with semantic anchor, runtime path evidence, and verification markers

#### Scenario: Code-first replacement is rejected
- **WHEN** mode semantic code is changed before the required documentation baseline is established
- **THEN** contract validation fails and the mode replacement is rejected

### Requirement: Real Runtime Mode Semantics SHALL Not Regress to Structural Templates
Mode implementations MUST keep mode-owned semantic structure and MUST NOT regress to highly homogeneous template skeletons across multiple modes.

#### Scenario: Structural template regression is detected
- **WHEN** contract validation detects high structural homogeneity and wrapper-only semantic ownership patterns across mode implementations
- **THEN** validation fails with anti-template classification and blocks merge

#### Scenario: Mode-specific semantic structure is preserved
- **WHEN** a mode implementation is inspected
- **THEN** mode-specific control flow and business semantics remain readable and traceable in mode-scoped code

### Requirement: Real Runtime Variant Difference SHALL Be Behavior-Derived
`minimal` and `production-ish` variants MUST produce differences from runtime behavior branches rather than marker-only string differences.

#### Scenario: Behavior-derived variant difference passes
- **WHEN** both variants execute for the same mode
- **THEN** runtime path evidence and semantic outcomes differ due to behavior branch changes

#### Scenario: Marker-only variant difference fails
- **WHEN** variants share the same behavior path and only differ in marker values or formatted output text
- **THEN** validation fails and blocks merge

