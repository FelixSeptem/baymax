## 1. Runtime Config and Validation

- [x] 1.1 Add CA3 reranker runtime config fields (`enabled`, timeout, retry bounds, provider/model threshold profile map)
- [x] 1.2 Add fail-fast validation for reranker config and mandatory provider/model threshold profile presence on startup/hot reload
- [x] 1.3 Preserve default E3 behavior when reranker is disabled
- [x] 1.4 Define stable schema version fields for threshold tuning toolkit input/output contracts

## 2. CA3 Quality Pipeline Integration

- [x] 2.1 Integrate deterministic reranker stage after base hybrid score and before final gate decision
- [x] 2.2 Enforce provider/model-specific threshold lookup for selected provider/model
- [x] 2.3 Keep policy semantics unchanged: `best_effort` fallback and `fail_fast` termination
- [x] 2.4 Ensure Run/Stream semantic equivalence for reranker enabled/disabled and fallback paths
- [x] 2.5 Add reranker extension interface and default built-in implementation path
- [x] 2.6 Ensure Anthropic reranker usable path is implemented (not diagnostics-only fallback)

## 3. Threshold Tuning Toolkit

- [x] 3.1 Implement offline tuning command/tool entry with corpus/label input validation
- [x] 3.2 Implement threshold sweep and recommendation generation with deterministic output behavior
- [x] 3.3 Emit operator-friendly markdown summary as minimal required output format
- [x] 3.4 Emit non-accepting recommendation reason codes when minimum quality gates are not met
- [x] 3.5 Add corpus readiness analysis and low-confidence warning reason fields
- [x] 3.6 Emit corpus-readiness warning/confidence guidance without hard-blocking recommendation output

## 4. Diagnostics and Observability

- [x] 4.1 Add CA3 reranker diagnostics fields (reranker-used, provider/model, threshold-source, threshold-hit, fallback-reason)
- [x] 4.2 Propagate new diagnostics fields through runner/event/diagnostics store APIs
- [x] 4.3 Keep diagnostics changes additive and backward-compatible

## 5. Tests and Performance Baseline

- [x] 5.1 Add contract tests for reranker success/fallback/fail-fast semantics
- [x] 5.2 Add contract tests for threshold profile precedence and deterministic fallback chain
- [x] 5.3 Add toolkit contract tests for schema validation and output format/version guarantees
- [x] 5.4 Add Run/Stream equivalence tests for reranker-enabled flows
- [x] 5.5 Add tests for custom reranker extension registration success/failure semantics
- [x] 5.6 Add provider coverage tests for OpenAI/Gemini/Anthropic reranker usable paths
- [x] 5.7 Add/extend benchmark cases for reranker enabled vs disabled latency regression checks
- [x] 5.8 Execute and pass `go test ./...`
- [x] 5.9 Execute and pass `go test -race ./...`
- [x] 5.10 Execute and pass `golangci-lint run --config .golangci.yml`

## 6. Documentation Sync

- [x] 6.1 Update `README.md` with CA3 reranker and threshold tuning workflow
- [x] 6.2 Update `docs/runtime-config-diagnostics.md` for reranker/tuning config and diagnostics fields
- [x] 6.3 Update `docs/context-assembler-phased-plan.md` with E4 scope and boundaries
- [x] 6.4 Update `docs/development-roadmap.md` to reflect E4 proposal and milestones
- [x] 6.5 Update `docs/v1-acceptance.md` and `docs/mainline-contract-test-index.md` with new contract coverage
