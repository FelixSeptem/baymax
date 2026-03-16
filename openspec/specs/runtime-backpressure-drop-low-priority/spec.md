# runtime-backpressure-drop-low-priority Specification

## Purpose
TBD - created by archiving change introduce-drop-low-priority-backpressure-r6. Update Purpose after archive.
## Requirements
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

### Requirement: Runtime SHALL determine low-priority calls by configuration rules
Runtime MUST determine drop eligibility by configuration rules only (tool-name and keyword rules), without requiring explicit priority field in call input arguments.

#### Scenario: Rule marks call as low priority
- **WHEN** a tool call matches configured low-priority rule
- **THEN** call becomes eligible for drop under queue pressure

#### Scenario: No rule match
- **WHEN** a tool call does not match any low-priority rule
- **THEN** call is treated as non-droppable under drop_low_priority policy

### Requirement: Runtime SHALL fail fast when all calls in a tool round are dropped
If all tool calls in a single dispatch round are dropped by `drop_low_priority`, runtime MUST terminate that run with fail-fast semantics.

#### Scenario: All calls dropped in one round
- **WHEN** all tool calls in the current iteration are dropped due to queue pressure
- **THEN** runner aborts with fail-fast error classification and emits terminal failure outcome

### Requirement: Runtime SHALL emit normalized observability reason for drop_low_priority
Runtime MUST emit `backpressure.drop_low_priority` as timeline reason when drop policy is applied and keep diagnostics fields semantically aligned.

#### Scenario: Drop policy triggered
- **WHEN** queue pressure triggers low-priority call drop
- **THEN** timeline includes `backpressure.drop_low_priority` and diagnostics counters reflect dropped-call count

### Requirement: Drop-low-priority backpressure SHALL apply consistently across local, mcp, and skill paths
The drop-low-priority mode MUST apply to `local`, `mcp`, and `skill` dispatch paths using the same configuration rule set and decision semantics.

#### Scenario: Same config across different dispatch paths
- **WHEN** identical drop-low-priority configuration is used for local, mcp, and skill calls
- **THEN** each path resolves priority and drop eligibility using the same rule semantics

#### Scenario: Path-specific round reaches all-drop condition
- **WHEN** local or mcp or skill dispatch round resolves to all dropped calls
- **THEN** fail-fast termination and error mapping behavior remains equivalent across paths

