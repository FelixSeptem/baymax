## ADDED Requirements

### Requirement: Project SHALL provide phased tutorial examples aligned with runtime maturity
The project MUST provide tutorial examples in phased batches aligned to roadmap milestones: foundational examples first, advanced concurrency/async examples later.

#### Scenario: R2 milestone examples are published
- **WHEN** roadmap reaches R2 examples milestone
- **THEN** foundational examples (minimal chat, basic tool loop, mixed MCP call, stream interruption) are available and runnable

#### Scenario: R3 milestone examples are published
- **WHEN** roadmap reaches R3 examples milestone
- **THEN** advanced examples (parallel fanout, async job progress, multi-agent async channel) are available and runnable

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
