## ADDED Requirements

### Requirement: Replay tooling SHALL validate workflow continuation evidence read-only

Diagnostics replay SHALL support bounded, versioned workflow continuation fixtures containing only opaque graph/checkpoint identities, graph digest/version, pause reason, claim/result/terminal evidence, capability declaration, Run/Stream projection, and expected typed outcomes. Replay SHALL validate duplicate, foreign, stale, conflicting, expired, corrupt, unavailable, crash-window, cancel/timeout/retry race, and unsupported-form facts deterministically. It SHALL NOT construct a workflow owner, claim a checkpoint, Run, Stream, retry, resume, enqueue, invoke a model/tool, or mutate diagnostics/source state.

#### Scenario: Workflow crash-window fixture remains non-executing
- **WHEN** replay validates a claimed workflow attempt with no committed terminal or successor checkpoint
- **THEN** it emits the declared in-progress or recovery-required classification
- **AND** it does not attempt to continue the workflow

### Requirement: Workflow continuation replay SHALL detect graph and identity drift

Replay SHALL classify graph version/digest, parent/run/session/causation, dependency state, completed-effect reference, claim identity, terminal evidence, capability scope, and Run/Stream parity drift. Overlapping identical source identities MAY deduplicate; conflicting payloads for one identity SHALL fail closed.

#### Scenario: Conflicting completed-step evidence fails closed
- **WHEN** a fixture repeats one source step identity with conflicting terminal/effect evidence
- **THEN** replay reports deterministic identity or payload drift and does not choose one projection
