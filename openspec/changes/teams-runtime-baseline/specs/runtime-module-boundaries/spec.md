## ADDED Requirements

### Requirement: Teams orchestration SHALL preserve runner-core boundary stability
Teams orchestration logic MUST be implemented outside `core/runner` and consumed via explicit interfaces so the runner main state machine remains focused on single-run loop semantics.

#### Scenario: Contributor introduces team orchestration logic
- **WHEN** a change implements Teams collaboration behavior
- **THEN** the implementation resides in the designated orchestration module and does not add cross-agent state transitions directly inside `core/runner`

### Requirement: Boundary checks SHALL cover Teams ownership rules
Boundary governance checks MUST verify both import direction and semantic ownership for Teams modules, including event emission and diagnostics write-path constraints.

#### Scenario: Teams module emits diagnostics
- **WHEN** Teams implementation adds observability output
- **THEN** output flows through `observability/event.RuntimeRecorder` without introducing direct diagnostics store writes
