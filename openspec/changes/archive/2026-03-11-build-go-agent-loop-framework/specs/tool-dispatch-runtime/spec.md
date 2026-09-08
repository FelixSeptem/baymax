## ADDED Requirements

### Requirement: Tool runtime SHALL use namespaced identifiers
The system MUST expose local tools using `local.<name>` identifiers and MUST prevent implicit override by conflicting names.

#### Scenario: Local tool registration
- **WHEN** a local tool named `search` is registered
- **THEN** the runtime MUST expose it as `local.search`

#### Scenario: Conflict prevention
- **WHEN** another tool with same final segment `search` exists in a different namespace
- **THEN** the runtime MUST require fully-qualified names and MUST NOT auto-select one

### Requirement: Tool runtime SHALL validate inputs against JSON schema
The system MUST validate tool invocation arguments before execution and MUST return structured validation errors for invalid arguments.

#### Scenario: Valid invocation
- **WHEN** tool arguments satisfy declared schema
- **THEN** the runtime MUST invoke the tool and capture result payload

#### Scenario: Invalid invocation
- **WHEN** tool arguments violate schema constraints
- **THEN** the runtime MUST skip execution and return a structured `ToolResult.error`

### Requirement: Tool runtime SHALL support controlled concurrency
The system MUST allow multiple tool calls in one iteration and MUST support policy controls for max concurrency and serial execution of write tools.

#### Scenario: Parallel read-only tools
- **WHEN** multiple read-only tool calls are requested in one iteration
- **THEN** the runtime MUST execute them concurrently up to configured limit

#### Scenario: Serialized write tools
- **WHEN** tool metadata marks calls as write operations requiring serialization
- **THEN** the runtime MUST execute those calls sequentially in deterministic order

### Requirement: Tool failures SHALL be returned to model context
Tool invocation failures MUST be represented in `ToolResult.error` and MUST be fed back into the next model step unless failure policy is fail-fast.

#### Scenario: Continue on tool error
- **WHEN** a tool call fails and failure policy allows continuation
- **THEN** the runtime MUST include failure details in model input for next iteration

#### Scenario: Fail-fast on tool error
- **WHEN** a tool call fails and failure policy requires fail-fast
- **THEN** the Runner MUST terminate the run with tool error classification
