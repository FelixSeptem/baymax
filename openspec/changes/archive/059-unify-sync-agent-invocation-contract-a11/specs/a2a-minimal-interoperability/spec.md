## ADDED Requirements

### Requirement: A2A WaitResult SHALL align with shared synchronous invocation contract
A2A `WaitResult` behavior consumed by orchestration modules MUST align with shared synchronous invocation contract for terminal convergence, cancellation, and error normalization.

#### Scenario: Shared invocation consumes A2A WaitResult
- **WHEN** orchestration path invokes A2A through shared synchronous invocation
- **THEN** `WaitResult` participates in terminal-only completion semantics and normalized error mapping

### Requirement: A2A synchronous waiting SHALL preserve polling compatibility defaults
A2A synchronous waiting consumed via shared invocation MUST preserve compatibility defaults for polling interval when caller does not override it.

#### Scenario: Caller omits poll interval
- **WHEN** shared synchronous invocation is called without poll interval override
- **THEN** A2A waiting behavior uses the existing default polling compatibility value
