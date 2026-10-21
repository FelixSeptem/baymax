## ADDED Requirements

### Requirement: Composite continuation replay SHALL be bounded and side-effect free

Replay fixtures SHALL validate composite source identity, graph/lease/message
facts, claim/result/terminal evidence, duplicate/conflict behavior, lifecycle
races, capability cells, and Run/Stream parity using opaque bounded references.
Replay MUST NOT construct an owner, claim a checkpoint, enqueue, publish, Run,
Stream, retry, resume, ack, nack, invoke a model/tool, or mutate source state.

#### Scenario: Replay rejects source identity drift
- **WHEN** a fixture changes a child attempt or parent causation reference
- **THEN** replay returns a deterministic drift failure without calling a source
  owner or execution dependency
