## 1. Runtime Config and Validation

- [x] 1.1 Add S2 security config schema for `namespace+tool` permission policy, process-scoped rate-limit policy, and governance mode default `enforce`
- [x] 1.2 Add model input/output filtering config schema and extension registration settings with deterministic precedence (`env > file > default`)
- [x] 1.3 Implement fail-fast validation for malformed `namespace+tool` keys, invalid mode/stage enums, and invalid policy values
- [x] 1.4 Extend hot-reload pipeline to ensure valid S2 config updates are atomically applied and invalid updates rollback to previous snapshot

## 2. Tool Security Governance Integration

- [x] 2.1 Implement pre-dispatch permission evaluation keyed by `namespace+tool` with explicit `allow|deny` behavior
- [x] 2.2 Implement process-scoped rate-limit evaluator keyed by `namespace+tool` with windowed counters
- [x] 2.3 Enforce violation behavior as fail-fast `deny` for both permission and rate-limit paths
- [x] 2.4 Preserve Run/Stream semantic equivalence for permission and rate-limit decisions

## 3. Model I/O Security Filtering Integration

- [x] 3.1 Add pluggable input/output filter interfaces and runtime wiring for host-provided implementations
- [x] 3.2 Execute input filters before provider invocation and output filters before final emission/return
- [x] 3.3 Implement blocking filter deny semantics for both input and output stages
- [x] 3.4 Preserve Run/Stream semantic equivalence for filter decision outcomes

## 4. Diagnostics and Observability Contract

- [x] 4.1 Add additive diagnostics fields for S2 decisions (`policy_kind`, `namespace+tool`, `filter_stage`, `decision`, `reason_code`)
- [x] 4.2 Emit normalized reason codes for permission deny, rate-limit deny, and I/O filter match/block outcomes
- [x] 4.3 Ensure new S2 diagnostics fields remain backward-compatible with existing consumers

## 5. Security Gate and Verification

- [x] 5.1 Add security-policy contract tests covering permission allow/deny, rate-limit deny, I/O filter deny, and invalid-reload rollback
- [x] 5.2 Add independent CI job (recommended `security-policy-gate`) and expose it as required-check candidate
- [x] 5.3 Update docs for S2 config fields, diagnostics fields, and security gate usage
- [x] 5.4 Run baseline validation (`go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`) and record results in implementation PR
