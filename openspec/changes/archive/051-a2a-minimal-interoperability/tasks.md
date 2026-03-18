## 1. Protocol Contract And Scope

- [x] 1.1 Finalize minimal A2A lifecycle contract (`submit/status/result`) and error mapping matrix.
- [x] 1.2 Define Agent Card schema subset required for baseline capability discovery and routing.
- [x] 1.3 Confirm A2A vs MCP boundary rules in architecture docs and spec deltas.

## 2. A2A Runtime Skeleton

- [x] 2.1 Add `a2a` module scaffold for client/server/card/router interfaces.
- [x] 2.2 Implement minimal task lifecycle endpoints and normalized status transitions.
- [x] 2.3 Implement baseline capability-based routing with deterministic selection rules.

## 3. Reliability And Semantics

- [x] 3.1 Implement bounded timeout/retry policy for client request and callback delivery.
- [x] 3.2 Implement normalized error classification mapping to runtime taxonomy.
- [x] 3.3 Ensure equivalent semantic outcomes for A2A interactions across Run and Stream paths.

## 4. Observability And Diagnostics

- [x] 4.1 Extend timeline mapping with A2A correlation metadata and reason codes.
- [x] 4.2 Extend diagnostics run summary with additive A2A aggregate fields.
- [x] 4.3 Validate single-writer + idempotent replay behavior for duplicated A2A events.

## 5. Validation And Documentation

- [x] 5.1 Add unit/integration tests for submit/status/result happy path and error path.
- [x] 5.2 Add cross-protocol contract tests covering combined A2A + MCP scenarios.
- [x] 5.3 Update `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, and roadmap status notes.
