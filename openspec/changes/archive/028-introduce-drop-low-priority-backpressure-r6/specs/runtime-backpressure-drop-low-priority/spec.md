## ADDED Requirements

### Requirement: Runtime SHALL support drop_low_priority backpressure mode for local tool dispatch
Runtime MUST support `drop_low_priority` as a valid backpressure mode and apply it only to local tool dispatch queue pressure path in this milestone.

#### Scenario: Startup with drop_low_priority mode
- **WHEN** runtime configuration sets `concurrency.backpressure=drop_low_priority`
- **THEN** runtime starts successfully and enables drop policy only for local tool dispatch path

#### Scenario: Non-local execution path remains unchanged
- **WHEN** runtime runs mcp or skill execution while `drop_low_priority` is enabled
- **THEN** behavior remains unchanged from existing baseline and no new drop policy is applied on those paths

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
