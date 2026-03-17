## 1. Runtime Config Governance Layer

- [x] 1.1 Add CA3 threshold governance config fields (`mode`, `profile_version`, provider:model rollout matcher) under reranker controls.
- [x] 1.2 Add fail-fast validation for governance mode enum and rollout matcher schema.
- [x] 1.3 Keep deterministic precedence (`env > file > default`) and preserve current defaults when governance is not enabled.

## 2. CA3 Governance Execution Path

- [x] 2.1 Integrate provider:model rollout matching into CA3 reranker threshold decision flow.
- [x] 2.2 Implement governance mode behavior: `enforce` affects gate decision, `dry_run` evaluates without enforcing gate changes.
- [x] 2.3 Preserve `best_effort` fallback and `fail_fast` termination semantics for governance evaluation failures.
- [x] 2.4 Ensure Run/Stream semantic equivalence for governance-enabled and governance-bypassed flows.

## 3. Observability and Diagnostics

- [x] 3.1 Add additive CA3 governance diagnostics fields (profile version, rollout hit/miss, threshold source/hit, fallback reason).
- [x] 3.2 Propagate new governance fields through runner payloads, event recorder, and diagnostics store mappings.
- [x] 3.3 Keep dry-run debugging semantics internal and avoid expanding required external diagnostics API contract beyond additive fields.

## 4. Tests and Benchmark Gates

- [x] 4.1 Add contract tests for `enforce` vs `dry_run` behavioral semantics.
- [x] 4.2 Add contract tests for deterministic provider:model rollout matching and fallback chain behavior.
- [x] 4.3 Add Run/Stream equivalence tests for governance-enabled threshold flows.
- [x] 4.4 Add/extend benchmark cases for governance enabled vs disabled latency regression checks.
- [x] 4.5 Execute and pass `go test ./...`.
- [x] 4.6 Execute and pass `go test -race ./...`.
- [x] 4.7 Execute and pass `golangci-lint run --config .golangci.yml`.

## 5. Documentation and Contract Index Sync

- [x] 5.1 Update `README.md` with CA3 threshold governance rollout and dry-run/enforce semantics.
- [x] 5.2 Update `docs/runtime-config-diagnostics.md` for governance config and additive observability fields.
- [x] 5.3 Update `docs/context-assembler-phased-plan.md` and `docs/development-roadmap.md` with E5 scope and boundaries.
- [x] 5.4 Update `docs/v1-acceptance.md` and `docs/mainline-contract-test-index.md` with E5 contract and benchmark coverage.
