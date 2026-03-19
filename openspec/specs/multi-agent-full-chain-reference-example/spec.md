# multi-agent-full-chain-reference-example Specification

## Purpose
TBD - created by archiving change introduce-full-chain-multi-agent-reference-example-a20. Update Purpose after archive.
## Requirements
### Requirement: Repository SHALL provide a runnable full-chain multi-agent reference example
The repository MUST provide a single tutorial example that composes `teams`, `workflow`, `a2a`, `scheduler`, and `recovery` in one executable path.

The example MUST be runnable with a local `go run` command and MUST NOT require external services for default execution.

#### Scenario: Contributor runs full-chain reference example
- **WHEN** contributor executes the documented example run command
- **THEN** example completes without requiring external network infrastructure

#### Scenario: User inspects full-chain composition path
- **WHEN** user reads the example entrypoint
- **THEN** code shows explicit composition across teams/workflow/a2a/scheduler/recovery domains

### Requirement: Full-chain example SHALL default to in-memory A2A integration
The full-chain reference example MUST use in-memory A2A transport/server path as default integration mode.

Network bridge behavior MAY be documented as extension path but MUST NOT be required for baseline execution.

#### Scenario: Example starts with default settings
- **WHEN** user runs the example without transport overrides
- **THEN** example uses in-memory A2A path and executes end-to-end flow successfully

#### Scenario: User reviews extension guidance
- **WHEN** user checks the example README
- **THEN** README distinguishes default in-memory mode from optional network extension mode

### Requirement: Full-chain example SHALL cover Run and Stream execution semantics
The example MUST expose both `Run` and `Stream` invocation paths for the same composed workflow intent.

#### Scenario: User runs non-stream path
- **WHEN** example executes through `Run`
- **THEN** output includes terminal summary markers for the composed flow

#### Scenario: User runs stream path
- **WHEN** example executes through `Stream`
- **THEN** output includes streaming events and terminal convergence consistent with composed flow semantics

### Requirement: Full-chain example SHALL include async, delayed, and recovery minimum composition
The reference flow MUST include at least one async-reporting path, one delayed-dispatch path, and one recovery-enable path in the same tutorial capability surface.

#### Scenario: Async and delayed steps are exercised
- **WHEN** user runs the full-chain example scenario
- **THEN** observable output includes async reporting markers and delayed dispatch markers

#### Scenario: Recovery path is exercised
- **WHEN** user runs the recovery-enabled example path
- **THEN** output includes recovery-related markers without violating existing terminal semantics

### Requirement: Full-chain example SHALL publish observability checkpoints
The example MUST document and emit minimal observability checkpoints for debugging, including run identifiers and key correlation markers from composed paths.

#### Scenario: User inspects example logs
- **WHEN** full-chain example is executed
- **THEN** logs or structured output contain traceable run/task correlation checkpoints

