## 1. API Reference Coverage Baseline

- [x] 1.1 Audit exported integration surfaces under `core/*`, `runtime/*`, `context/*`, and `skill/*` and define D1 coverage scope list
- [x] 1.2 Add or update godoc comments and minimal usage examples for prioritized exported APIs in the D1 scope
- [x] 1.3 Add README/docs navigation entry that links to D1 API reference materials with Chinese-first wording

## 2. Diagnostics Replay Tooling

- [x] 2.1 Implement replay library package that parses diagnostics JSON input and normalizes minimal timeline fields (`phase/status/reason/timestamp` + correlation IDs)
- [x] 2.2 Define deterministic replay validation reason codes for malformed JSON and missing-required-field failures
- [x] 2.3 Provide minimal invocation entry (library-first, optional thin CLI wrapper) for local debug replay

## 3. Contract Tests and Quality Gate

- [x] 3.1 Add replay contract fixtures for success and failure paths with stable expected outputs/error codes
- [x] 3.2 Add automated tests for replay normalization ordering and deterministic reason-code behavior
- [x] 3.3 Add CI replay gate job (recommended name `diagnostics-replay-gate`) and wire it as an independent required-check candidate

## 4. Documentation and Final Verification

- [x] 4.1 Update docs troubleshooting section with JSON replay usage and output interpretation guide (Chinese-first, English acceptable)
- [x] 4.2 Ensure CI/docs references consistently mention replay gate and API reference scope (`core/runtime/context/skill`)
- [x] 4.3 Run baseline verification (`go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`) and record results for implementation PR
