## 1. Contract Freeze And Interface Alignment

- [x] 1.1 Confirm final field contract for composed correlation (`workflow_id/team_id/step_id/task_id/agent_id/peer_id`) and freeze DTO mapping points.
- [x] 1.2 Define/confirm reason namespace additions (`workflow.dispatch_a2a`, `team.dispatch_remote`, `team.collect_remote`) and align with shared contract gate rules.
- [x] 1.3 Finalize orchestration-to-A2A integration interface boundaries and verify no MCP ownership overlap.

## 2. Workflow Remote-Step Integration

- [x] 2.1 Extend workflow DSL schema/validation to support A2A remote step kind and required fields.
- [x] 2.2 Implement workflow adapter path for A2A remote step execution under existing retry/timeout semantics.
- [x] 2.3 Ensure workflow checkpoint/resume and deterministic scheduling remain stable with remote-step participation.

## 3. Teams Mixed Local/Remote Execution

- [x] 3.1 Extend teams task model to express local vs remote worker execution target.
- [x] 3.2 Implement mixed execution dispatch/collect flow (local runner + A2A remote worker) under serial/parallel/vote strategies.
- [x] 3.3 Align mixed execution failure/cancellation convergence with existing team lifecycle and aggregate semantics.

## 4. Runtime Config And Diagnostics Integration

- [x] 4.1 Add composed-orchestration config fields under existing domain scopes and wire `env > file > default` precedence.
- [x] 4.2 Add startup/hot-reload fail-fast validation and rollback tests for invalid composed config.
- [x] 4.3 Extend run diagnostics with additive composed summary fields and preserve replay-idempotent aggregation behavior.

## 5. Timeline And Boundary Governance

- [x] 5.1 Emit composed orchestration timeline reasons and cross-domain correlation metadata on relevant paths.
- [x] 5.2 Verify composed events still ingest through `observability/event.RuntimeRecorder` single-writer path only.
- [x] 5.3 Update/extend shared multi-agent contract gate checks for composed consistency (reason namespace + `peer_id` naming + status mapping).

## 6. Contract Tests, Regression, And Documentation

- [x] 6.1 Add unit/integration contract tests for workflow A2A remote step (success/failure/timeout/retry).
- [x] 6.2 Add contract tests for teams mixed local+remote execution and Run/Stream semantic equivalence.
- [x] 6.3 Add composed replay-idempotency and A2A+MCP boundary regression tests.
- [x] 6.4 Update `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/v1-acceptance.md`, `docs/mainline-contract-test-index.md`, and `docs/development-roadmap.md`.
- [x] 6.5 Execute validation gates: `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, and multi-agent shared contract checks.
