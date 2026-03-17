## 1. Runtime Config and Validation

- [x] 1.1 Add S3 security-event config schema (event enablement, deny-only alert policy, severity mapping, callback constraints) with defaults
- [x] 1.2 Wire S3 config loading with deterministic precedence (`env > file > default`)
- [x] 1.3 Implement fail-fast validation for invalid S3 event enums/policy values and malformed severity mapping
- [x] 1.4 Extend hot-reload path to atomically apply valid S3 config and rollback on invalid update

## 2. Runner Security Event and Callback Integration

- [x] 2.1 Introduce normalized S3 security event envelope and event builder in runner security path
- [x] 2.2 Implement deny-only callback alert dispatch for tool permission/rate-limit deny outcomes
- [x] 2.3 Implement deny-only callback alert dispatch for model input/output filter deny outcomes
- [x] 2.4 Enforce Run/Stream semantic equivalence for S3 event taxonomy and callback trigger behavior

## 3. Diagnostics and Observability Contract

- [x] 3.1 Add additive diagnostics fields for S3 (`severity`, alert dispatch status/failure reason)
- [x] 3.2 Normalize reason code and severity mapping across `permission|rate_limit|io_filter` sources
- [x] 3.3 Preserve backward compatibility for existing run diagnostics consumers

## 4. Security Event Gate and Contract Tests

- [x] 4.1 Add S3 contract tests for deny-only alert trigger, callback failure handling, severity normalization, and invalid-reload rollback
- [x] 4.2 Add independent CI job `security-event-gate` and expose it as required-check candidate
- [x] 4.3 Add cross-platform gate scripts (`check-security-event-contract.sh` and `.ps1`)

## 5. Documentation and Baseline Verification

- [x] 5.1 Update runtime docs with S3 config examples, event taxonomy fields, and callback sink usage
- [x] 5.2 Run baseline validation (`go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`) and record results in implementation PR
