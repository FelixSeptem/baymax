## 1. Contract And Data Model Freeze

- [x] 1.1 Freeze scheduler task/attempt/lease state model and idempotency key schema.
- [x] 1.2 Freeze scheduler and subagent reason namespace (`scheduler.*` / `subagent.*`) and correlation fields.
- [x] 1.3 Confirm A2A-scheduler integration boundary and ensure no MCP responsibility overlap.

## 2. Scheduler Core Module Baseline

- [x] 2.1 Add scheduler module scaffold (queue store, claim API, heartbeat API, complete/fail API interfaces).
- [x] 2.2 Implement enqueue/claim lifecycle with atomic lease creation and attempt tracking.
- [x] 2.3 Implement heartbeat renew and lease-expiry requeue/takeover logic.
- [x] 2.4 Implement idempotent terminal commit for duplicate result/failure submissions.

## 3. Backend And Persistence Baseline

- [x] 3.1 Implement in-memory scheduler backend with concurrency-safe behavior.
- [x] 3.2 Implement persistent scheduler backend baseline (sqlite or approved equivalent) with crash-recovery semantics.
- [x] 3.3 Add backend parity tests for enqueue/claim/heartbeat/expire/requeue/complete paths.

## 4. A2A Integration And Subagent Guardrails

- [x] 4.1 Integrate scheduler claim execution path with A2A dispatch and normalized terminal mapping.
- [x] 4.2 Add parent-child run guardrails (`max_depth`, `max_active_children`, child timeout budget) and fail-fast enforcement.
- [x] 4.3 Implement takeover-safe retry semantics for retryable remote failures under lease expiration.

## 5. Config, Timeline, And Diagnostics

- [x] 5.1 Add `scheduler.*` and `subagent.*` config schema/defaults with `env > file > default` precedence.
- [x] 5.2 Add startup/hot-reload validation and rollback tests for invalid scheduler/subagent configuration.
- [x] 5.3 Emit scheduler/subagent timeline reasons and correlation metadata on key transitions.
- [x] 5.4 Extend run diagnostics with additive scheduler/subagent summary fields and replay-idempotent aggregation checks.
- [x] 5.5 Ensure scheduler observability ingestion remains on `observability/event.RuntimeRecorder` single-writer path.

## 6. Contract Tests, Regression Gates, And Docs

- [x] 6.1 Add contract tests for worker crash + lease expiry + takeover execution.
- [x] 6.2 Add contract tests for duplicate submit/result replay idempotency.
- [x] 6.3 Add Run/Stream semantic-equivalence tests for scheduler-managed subagent flows.
- [x] 6.4 Add A2A+scheduler integration regression tests for retry and error-layer normalization.
- [x] 6.5 Update `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, and `docs/v1-acceptance.md`.
- [x] 6.6 Execute validation gates: `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, and multi-agent shared contract checks.
