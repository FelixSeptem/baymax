# agent-mode-anti-template-doc-first-delivery-contract Specification

## Purpose
TBD - created by archiving change introduce-agent-mode-anti-template-doc-first-delivery-contract-a72. Update Purpose after archive.
## Requirements
### Requirement: Agent Mode Delivery SHALL Follow Doc-First Sequence
For `examples/agent-modes` changes under this contract, maintainers MUST finish mode documentation baseline before implementing mode code changes.

Documentation baseline MUST include, for each touched mode:
- semantic anchor
- real runtime path evidence
- expected verification markers
- failure/rollback notes

#### Scenario: Mode documentation is completed before code replacement
- **WHEN** a maintainer starts replacing a mode implementation
- **THEN** the corresponding matrix/readme baseline already contains semantic anchor, runtime path evidence, expected markers, and rollback notes

#### Scenario: Missing baseline blocks implementation acceptance
- **WHEN** mode code changes are submitted without the required documentation baseline
- **THEN** the change is treated as incomplete and MUST NOT be accepted

### Requirement: Per-Mode Business Semantics SHALL Be Mode-Owned
Each mode MUST own its business-semantic execution logic in mode-scoped code and MUST NOT rely on a shared cross-mode semantic template engine for business behavior.

Shared utilities MAY be used only for non-semantic concerns, including formatting, common assertions, and generic helpers.

#### Scenario: Mode-owned semantic logic exists
- **WHEN** a mode implementation is reviewed
- **THEN** mode-specific business decisions and state transitions are implemented in the mode scope

#### Scenario: Shared semantic template engine is rejected
- **WHEN** mode business semantics are executed by one shared template engine and modes only provide constants/markers
- **THEN** validation fails and the change is blocked

### Requirement: Variant Divergence SHALL Come From Runtime Behavior
`minimal` and `production-ish` variants MUST diverge in runtime behavior paths and MUST NOT differ only by static marker strings or output formatting.

#### Scenario: Variants diverge through behavior branches
- **WHEN** both variants of a mode are executed
- **THEN** they show distinct behavior branches with mode-relevant runtime evidence

#### Scenario: Marker-only divergence is rejected
- **WHEN** variant differences are limited to marker literals or print output while behavior path is unchanged
- **THEN** validation fails and blocks merge

### Requirement: Task Completion SHALL Require Four Evidence Types
A mode-level task under this contract MUST be marked complete only when all four evidence types are present:
- code evidence
- test evidence
- documentation evidence
- gate evidence

#### Scenario: Task completion requires four evidences
- **WHEN** maintainers mark a mode task as complete
- **THEN** code/test/documentation/gate evidence are all present and traceable

#### Scenario: Missing evidence prevents task completion
- **WHEN** any required evidence type is absent
- **THEN** the task MUST remain unchecked

