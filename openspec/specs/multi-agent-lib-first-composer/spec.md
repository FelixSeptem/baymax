# multi-agent-lib-first-composer Specification

## Purpose
TBD - created by archiving change introduce-lib-first-agent-composer-with-scheduler-bridge-a8. Update Purpose after archive.
## Requirements
### Requirement: Composer SHALL provide a library-first unified orchestration entrypoint
The runtime MUST provide a dedicated `orchestration/composer` package that composes runner, workflow, teams, A2A, and scheduler capabilities behind a single library-first entrypoint, so hosts no longer need manual multi-module stitching.

#### Scenario: Host initializes composed runtime through composer package
- **WHEN** host code constructs and executes a multi-agent run through `orchestration/composer`
- **THEN** the composed path executes without requiring host-side manual wiring of workflow/teams/a2a/scheduler internals

### Requirement: Composer SHALL support scheduler-managed local and A2A child execution
Composer-managed subagent execution MUST support both local child-run and A2A child-run targets, and MUST converge both targets through scheduler terminal commit semantics.

#### Scenario: Parent run dispatches mixed child targets
- **WHEN** one composed run dispatches child tasks to both local and A2A targets under scheduler management
- **THEN** both targets produce normalized task terminal states and idempotent scheduler commits through the same convergence contract

### Requirement: Composer SHALL preserve Run/Stream semantic equivalence
For equivalent requests and effective configuration, composer-managed Run and Stream paths MUST preserve semantically equivalent terminal status category and additive aggregate summaries.

#### Scenario: Equivalent composed request through Run and Stream
- **WHEN** an equivalent composer-managed request executes once with Run and once with Stream
- **THEN** terminal status category and required additive summary counters remain semantically equivalent

