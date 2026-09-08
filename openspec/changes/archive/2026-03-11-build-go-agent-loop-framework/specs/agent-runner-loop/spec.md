## ADDED Requirements

### Requirement: Runner SHALL execute a bounded agent loop
The Runner MUST execute the agent loop as explicit iterative steps and MUST stop when one of the configured termination conditions is met.

#### Scenario: Final answer terminates run
- **WHEN** the model returns a `final_answer` and no pending tool calls
- **THEN** the Runner MUST finalize the run and return `RunResult` with terminal status

#### Scenario: Iteration limit terminates run
- **WHEN** iteration count reaches `LoopPolicy.MaxIterations`
- **THEN** the Runner MUST stop further model/tool steps and return an iteration-limit error classification

### Requirement: Runner SHALL enforce per-step timeout
The Runner MUST apply `LoopPolicy.StepTimeout` to each model step and tool dispatch step.

#### Scenario: Model step timeout
- **WHEN** a model request exceeds `StepTimeout`
- **THEN** the Runner MUST record timeout classification and apply configured retry/failure policy

#### Scenario: Tool dispatch timeout
- **WHEN** tool dispatch exceeds `StepTimeout`
- **THEN** the Runner MUST mark affected tool calls as timed out and continue or abort based on failure policy

### Requirement: Runner SHALL return standardized run result
The Runner MUST return a standardized `RunResult` containing run identifiers, usage summary, latency, iteration count, tool call summaries, and optional terminal error.

#### Scenario: Successful run result
- **WHEN** a run completes without terminal error
- **THEN** the returned `RunResult` MUST include `RunID`, `FinalAnswer`, `Iterations`, `ToolCalls`, `TokenUsage`, and `Latency`

#### Scenario: Failed run result
- **WHEN** a run terminates with unrecoverable error
- **THEN** the returned `RunResult` MUST include structured `Error` classification and diagnostic warnings
