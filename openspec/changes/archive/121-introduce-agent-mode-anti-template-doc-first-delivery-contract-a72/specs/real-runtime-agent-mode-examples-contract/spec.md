## ADDED Requirements

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
