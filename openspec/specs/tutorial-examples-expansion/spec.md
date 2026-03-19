# tutorial-examples-expansion Specification

## Purpose
TBD - created by archiving change optimize-runtime-concurrency-and-async-io. Update Purpose after archive.
## Requirements
### Requirement: Project SHALL provide phased tutorial examples aligned with runtime maturity
The project MUST provide tutorial examples in phased batches aligned to roadmap milestones: foundational examples first, advanced concurrency/async examples later.

#### Scenario: R2 milestone examples are published
- **WHEN** roadmap reaches R2 examples milestone
- **THEN** foundational examples (minimal chat, basic tool loop, mixed MCP call, stream interruption) are available and runnable

#### Scenario: R3 milestone examples are published
- **WHEN** roadmap reaches R3 examples milestone
- **THEN** advanced examples (parallel fanout, async job progress, multi-agent async channel, and multi-agent network bridge) are available and runnable

### Requirement: Each tutorial example SHALL include TODO extension points
Each tutorial example MUST include a TODO section or TODO file describing optimization opportunities, known limits, and future extension ideas.

#### Scenario: Contributor inspects an example
- **WHEN** a contributor opens an example directory
- **THEN** they can find explicit TODO items for follow-up optimization and extension work

### Requirement: Tutorial docs SHALL explain expected concurrency behavior
Tutorial documentation MUST describe expected concurrency behavior and caveats for each advanced example.

#### Scenario: User runs a parallel or async tutorial
- **WHEN** a user follows an advanced tutorial
- **THEN** documentation explains expected fanout/queue behavior and how to interpret runtime diagnostics

### Requirement: Advanced tutorials SHALL integrate runtime manager consistently
All R3 advanced tutorials MUST initialize runtime configuration through `runtime/config.Manager` so examples follow the same config/diagnostics path as production library usage.

#### Scenario: User opens an advanced example entrypoint
- **WHEN** user reads or runs examples `05` to `08`
- **THEN** example code creates and uses runtime manager as the standard runtime entry

### Requirement: Async and multi-agent tutorials SHALL emit structured events
Advanced tutorials involving asynchronous progress or agent collaboration MUST emit structured events for observable execution paths.

#### Scenario: User runs async progress or multi-agent examples
- **WHEN** user runs examples `06`, `07`, or `08`
- **THEN** output includes structured event records that expose stage/progress transitions and correlation identifiers

### Requirement: Network bridge tutorial SHALL use JSON-RPC 2.0 message protocol
The `08-multi-agent-network-bridge` tutorial MUST use JSON-RPC 2.0 request/response semantics over HTTP for inter-agent network communication, following MCP-aligned message conventions.

#### Scenario: User inspects network bridge message flow
- **WHEN** user runs or reads example `08`
- **THEN** request, response, and error messages conform to JSON-RPC 2.0 fields (`jsonrpc`, `id`, `method`, `params`, `result`, `error`)

#### Scenario: User inspects network bridge transport
- **WHEN** user runs or reads example `08`
- **THEN** JSON-RPC 2.0 messages are exchanged via HTTP endpoints instead of raw TCP socket transport

### Requirement: README SHALL provide pattern-oriented navigation for tutorials
README MUST provide a navigation index that maps tutorial example directories to their corresponding design patterns.

#### Scenario: User selects tutorial by pattern
- **WHEN** user reads README tutorial section
- **THEN** user can find example entrypoints by pattern category without scanning all directories manually

### Requirement: Multi-agent tutorials SHALL include clarification HITL path
At least one multi-agent tutorial MUST demonstrate clarification HITL interaction where an agent requests user clarification and resumes execution with the returned answer.

#### Scenario: User runs multi-agent clarification tutorial
- **WHEN** user executes the selected multi-agent example
- **THEN** example emits a structured clarification request, accepts simulated/user clarification input, and continues the workflow

#### Scenario: User inspects tutorial output events
- **WHEN** user observes runtime events for the tutorial run
- **THEN** output contains structured clarification lifecycle events for await/resume or await/cancel path

### Requirement: Tutorials SHALL include Action Gate parameter-rule minimal demonstration
At least one tutorial example MUST demonstrate Action Gate parameter-rule matching behavior, including a rule-hit path and resulting gate decision outcome.

#### Scenario: User runs parameter-rule tutorial path
- **WHEN** user executes the selected tutorial example
- **THEN** output includes a parameter-rule match signal and the corresponding gate decision behavior

#### Scenario: User inspects tutorial event output
- **WHEN** user observes runtime events for the tutorial run
- **THEN** timeline includes `gate.rule_match` reason semantics for matched parameter rules

### Requirement: Tutorial catalog SHALL include full-chain multi-agent reference example
Tutorial examples MUST include a dedicated full-chain reference example that demonstrates composition across `team + workflow + a2a + scheduler + recovery`.

#### Scenario: User browses tutorial directories
- **WHEN** user checks tutorial index and examples directory
- **THEN** a full-chain multi-agent reference example is listed and discoverable

#### Scenario: User executes full-chain tutorial command
- **WHEN** user runs the documented command for the full-chain example
- **THEN** tutorial runs successfully and demonstrates composed multi-agent path

### Requirement: Full-chain tutorial docs SHALL provide dual-path run guidance
The full-chain tutorial documentation MUST provide both `Run` and `Stream` execution guidance and expected observable outputs.

#### Scenario: User follows Run path documentation
- **WHEN** user executes tutorial in Run mode
- **THEN** documentation-aligned terminal summary output is observable

#### Scenario: User follows Stream path documentation
- **WHEN** user executes tutorial in Stream mode
- **THEN** documentation-aligned streaming output and terminal convergence are observable

### Requirement: Full-chain tutorial SHALL document async-delayed-recovery composition checkpoints
The tutorial documentation MUST call out where async reporting, delayed dispatch, and recovery semantics appear in the reference flow and what markers to verify.

#### Scenario: User validates async/delayed/recovery checkpoints
- **WHEN** user follows tutorial verification steps
- **THEN** user can locate explicit async, delayed, and recovery checkpoints in output or logs

