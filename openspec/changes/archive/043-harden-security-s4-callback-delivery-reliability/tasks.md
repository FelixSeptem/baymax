## 1. Runtime Config and Validation

- [x] 1.1 Add `security.security_event.delivery.*` config schema and defaults (`mode=async`, bounded queue, `drop_old`, timeout, retry=3, circuit breaker)
- [x] 1.2 Wire S4 config loading with deterministic precedence (`env > file > default`)
- [x] 1.3 Implement fail-fast validation for invalid delivery enums/thresholds and malformed retry/circuit settings
- [x] 1.4 Extend hot-reload path to atomically apply valid S4 config and rollback on invalid update

## 2. Runner Delivery Executor Integration

- [x] 2.1 Introduce callback delivery executor abstraction (sync/async) and integrate into deny-alert dispatch path
- [x] 2.2 Implement async bounded queue worker with overflow policy `drop_old`
- [x] 2.3 Implement timeout + exponential backoff retry (max 3 attempts) for callback dispatch
- [x] 2.4 Implement Hystrix-style circuit breaker (`closed/open/half_open`) for callback sink
- [x] 2.5 Enforce deny decision semantic invariance (delivery failure MUST NOT alter deny outcome)
- [x] 2.6 Enforce Run/Stream semantic equivalence for delivery mode, retry behavior, and circuit outcomes

## 3. Diagnostics and Observability Contract

- [x] 3.1 Add additive S4 diagnostics fields (`alert_delivery_mode`, `alert_retry_count`, `alert_queue_drop*`, `alert_circuit_state`, failure reason)
- [x] 3.2 Normalize delivery failure reason codes and circuit state taxonomy across permission/rate-limit/io-filter sources
- [x] 3.3 Preserve backward compatibility for existing diagnostics consumers

## 4. Security Delivery Gate and Contract Tests

- [x] 4.1 Add S4 contract tests for async delivery, `drop_old` overflow behavior, retry budget, and circuit breaker transitions
- [x] 4.2 Add S4 contract tests for deny-only trigger invariance and callback failure non-interference
- [x] 4.3 Add S4 contract tests for Run/Stream delivery semantic equivalence and invalid-reload rollback
- [x] 4.4 Add cross-platform gate scripts (`check-security-delivery-contract.sh` and `.ps1`)
- [x] 4.5 Add independent CI job `security-delivery-gate` and expose as required-check candidate

## 5. Documentation and Baseline Verification

- [x] 5.1 Update runtime docs with S4 delivery config examples, Hystrix-style circuit behavior, and diagnostics field definitions
- [x] 5.2 Run baseline validation (`go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`) and record results in implementation PR
