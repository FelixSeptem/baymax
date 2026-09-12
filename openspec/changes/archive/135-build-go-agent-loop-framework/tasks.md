## 1. Project Skeleton and Contracts

- [x] 1.1 Create package layout for `core/types`, `core/runner`, `observability/event`, `observability/trace`, `model/openai`, `tool/local`, `mcp/stdio`, `mcp/http`, and `skill/loader`
- [x] 1.2 Define public contracts for `Runner`, `ModelClient`, `Tool`, `MCPClient`, `SkillLoader`, and `EventHandler`
- [x] 1.3 Define shared DTOs for `RunRequest`, `RunResult`, `ModelRequest`, `ModelResponse`, `ToolResult`, and classified errors
- [x] 1.4 Add baseline unit tests to validate contract serialization and default policy values

## 2. M0 Runner Loop Baseline

- [x] 2.1 Implement explicit runner state machine with `Init`, `ModelStep`, `DecideNext`, `Finalize`, and `Abort`
- [x] 2.2 Implement non-stream `Run` path for single-turn model completion without tools
- [x] 2.3 Implement stream `Stream` path with model delta forwarding via callback
- [x] 2.4 Emit minimum lifecycle events `run.started`, `model.requested`, `model.completed`, and `run.finished`
- [x] 2.5 Add tests for normal completion, timeout abort, and iteration-limit abort branches

## 3. M1 Local Tool Runtime

- [x] 3.1 Implement local tool registry with namespaced identifiers `local.<name>`
- [x] 3.2 Add JSON schema validation before tool invocation and structured validation error output
- [x] 3.3 Implement tool dispatch with per-iteration call limits and configurable concurrency
- [x] 3.4 Implement write-tool serialization mode and deterministic execution ordering
- [x] 3.5 Implement tool-result merge and model feedback loop for multi-iteration execution
- [x] 3.6 Add tests for successful tool loops, validation failure, and fail-fast vs continue policy behavior

## 4. M2 MCP Stdio Adapter

- [x] 4.1 Implement stdio MCP client lifecycle with initialize and list-tools warmup
- [x] 4.2 Implement session pool controls for read/write worker sizing
- [x] 4.3 Implement timeout and cancellation propagation for each MCP call
- [x] 4.4 Normalize stdio tool responses into shared `ToolResult` model
- [x] 4.5 Emit normalized MCP events for requested/completed/failed outcomes
- [x] 4.6 Add adapter tests for timeout, retry, and pooled-call behavior

## 5. M2 MCP HTTP/SSE Adapter

- [x] 5.1 Implement HTTP/SSE MCP client with configurable endpoint and auth headers
- [x] 5.2 Implement heartbeat monitor and exponential-backoff reconnect policy
- [x] 5.3 Preserve stable call identifiers across reconnect boundaries
- [x] 5.4 Align timeout, retry, and cancellation semantics with stdio adapter
- [x] 5.5 Add tests for reconnect flow, duplicate-call prevention, and event ordering guarantees

## 6. Skill Discovery and Resolution

- [x] 6.1 Implement AGENTS-first discovery flow and SKILL file indexing rules
- [x] 6.2 Implement explicit mention trigger and semantic trigger with explicit-priority override
- [x] 6.3 Implement conflict resolution precedence `system built-in > AGENTS > SKILL`
- [x] 6.4 Compile active skills into `SkillBundle` (`SystemPromptFragments`, `EnabledTools`, `WorkflowHints`)
- [x] 6.5 Emit `skill.loaded` and skill-failure warning events with non-blocking fallback behavior
- [x] 6.6 Add tests for missing skills, conflict resolution, and partial compile failure

## 7. Observability and Runtime Diagnostics

- [x] 7.1 Define versioned runtime event schema and payload correlation fields (`run_id`, iteration, call_id)
- [x] 7.2 Implement OTel root span `agent.run` and child spans for skill/model/tool/mcp steps
- [x] 7.3 Implement JSON stdout logger with trace/span correlation fields
- [x] 7.4 Ensure emitted events, logs, and spans can be joined by run and trace identity
- [x] 7.5 Add tests/assertions for event ordering and correlation field completeness

## 8. Integration, Performance, and Acceptance

- [x] 8.1 Build fake model, fake tools, and fake MCP servers for deterministic integration scenarios
- [x] 8.2 Implement end-to-end integration tests for multi-turn tool calls and mixed local/MCP dispatch
- [x] 8.3 Add streaming integration tests to verify no event loss and causal ordering
- [x] 8.4 Add benchmark suite for iteration latency, tool fan-out, and MCP reconnect overhead
- [x] 8.5 Validate acceptance criteria and document known limitations for v1 non-goals
