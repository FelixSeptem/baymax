## 1. Profile Model and Shared Runtime

- [x] 1.1 Define MCP reliability profile schema and defaults (`dev/default/high-throughput/high-reliability`)
- [x] 1.2 Implement shared MCP runtime components for retry/backoff/reconnect and event normalization
- [x] 1.3 Migrate `mcp/http` to shared runtime while preserving transport-specific hooks
- [x] 1.4 Migrate `mcp/stdio` to shared runtime while preserving transport-specific hooks

## 2. Reliability Validation and Diagnostics

- [x] 2.1 Add fault-injection tests for heartbeat timeout, reconnect storm, transient retry, and non-retryable fail-fast
- [x] 2.2 Add/extend benchmark scenarios to compare profile behavior under load and failure
- [x] 2.3 Implement recent N MCP-call diagnostic summary output with normalized fields

## 3. Docs and State Alignment

- [x] 3.1 Update MCP runtime docs with profile defaults, tuning guidance, and failure semantics
- [x] 3.2 Update README and docs status sections to match actual archived/active change state
- [x] 3.3 Ensure proposal/design/spec/tasks remain consistent and validate with OpenSpec
