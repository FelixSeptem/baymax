## MODIFIED Requirements

### Requirement: Runtime SHALL support drop_low_priority backpressure mode for local tool dispatch
When `concurrency.backpressure=drop_low_priority`, the runtime MUST classify calls using configured rules and MAY drop only calls whose resolved priority is in `droppable_priorities` when backpressure is reached.

The drop-low-priority mode MUST apply consistently to `local`, `mcp`, and `skill` dispatch semantics using the same rule model.

The runtime MUST preserve fail-fast semantics: if all calls in a dispatch phase round are dropped by low-priority backpressure, the run MUST terminate with a tool error classification.

#### Scenario: Backpressure reaches queue limits with mixed priorities
- **WHEN** queue/inflight pressure is reached and a round contains droppable and non-droppable calls
- **THEN** only droppable calls are shed and non-droppable calls continue via blocking/normal path

#### Scenario: All calls in a round are dropped
- **WHEN** every call in a dispatch round is dropped due to low-priority backpressure
- **THEN** the runner fails fast in that round and returns consistent tool error classification

## ADDED Requirements

### Requirement: Drop-low-priority backpressure SHALL apply consistently across local, mcp, and skill paths
The drop-low-priority mode MUST apply to `local`, `mcp`, and `skill` dispatch paths using the same configuration rule set and decision semantics.

#### Scenario: Same config across different dispatch paths
- **WHEN** identical drop-low-priority configuration is used for local, mcp, and skill calls
- **THEN** each path resolves priority and drop eligibility using the same rule semantics

#### Scenario: Path-specific round reaches all-drop condition
- **WHEN** local or mcp or skill dispatch round resolves to all dropped calls
- **THEN** fail-fast termination and error mapping behavior remains equivalent across paths
