## 1. Contract And Scope

- [x] 1.1 Finalize delivery mode matrix (`callback|sse`) and deterministic fallback policy.
- [x] 1.2 Finalize Agent Card version negotiation policy (`strict_major + compatible_minor`) and error mapping table.
- [x] 1.3 Confirm A2A/MCP boundary rules and shared contract-gate checks for naming consistency.

## 2. Runtime Config Integration

- [x] 2.1 Add `a2a.delivery.*` and `a2a.card.version_policy.*` config structures and defaults in `runtime/config`.
- [x] 2.2 Add YAML/ENV mapping with `env > file > default` precedence.
- [x] 2.3 Add startup/hot-reload validation and rollback tests for invalid delivery/version configs.

## 3. A2A Delivery And Negotiation Runtime

- [x] 3.1 Implement delivery-mode negotiation and fallback state machine in A2A runtime path.
- [x] 3.2 Implement bounded callback retry and SSE reconnect controls.
- [x] 3.3 Implement Agent Card version negotiation and normalized incompatibility handling.

## 4. Observability And Diagnostics

- [x] 4.1 Extend timeline reason mapping with `a2a.sse_subscribe|a2a.sse_reconnect|a2a.delivery_fallback|a2a.version_mismatch`.
- [x] 4.2 Extend diagnostics with additive A2A delivery/version fields (`a2a_delivery_mode`, `a2a_version_*`, fallback metadata).
- [x] 4.3 Verify single-writer and replay-idempotent behavior for duplicated delivery/version events.

## 5. Contract Tests And Docs

- [x] 5.1 Add unit/integration tests for delivery negotiation success/fallback/failure paths.
- [x] 5.2 Add compatibility-matrix tests for card version negotiation and Run/Stream semantic equivalence.
- [x] 5.3 Update `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/development-roadmap.md`, and `docs/mainline-contract-test-index.md`.
- [x] 5.4 Execute regression gates: `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, and relevant A2A contract gates.
