## 1. Contract Types and Normalization

- [x] 1.1 Define versioned corpus, corpus item, reference, metric/rubric, Badcase, experiment comparison, and feedback recommendation DTOs with additive nullable fields.
- [x] 1.2 Implement deterministic normalization, enum validation, bounded reference/digest validation, and stable digest generation for corpus and rubric declarations.
- [x] 1.3 Implement Badcase reproduction status classification and experiment shard deduplication/conflict detection without mutating source runtime state.
- [x] 1.4 Add positive, negative, boundary, idempotency, and local/distributed parity unit tests for all new contract types.

## 2. Configuration and Runtime Integration

- [x] 2.1 Add only the minimum optional eval corpus/comparison/feedback configuration needed by the contract, with defaults and strict `env > file > default` parsing.
- [x] 2.2 Add fail-fast validation and atomic hot-reload rollback tests for invalid versions, bounds, modes, and approval settings.
- [x] 2.3 Integrate corpus/Badcase/experiment/feedback projection with existing run and stream eval paths while preserving Run/Stream semantic equivalence.

## 3. Observability and Diagnostics

- [x] 3.1 Extend `RuntimeRecorder` with bounded additive nullable correlation fields and existing redaction/cardinality enforcement.
- [x] 3.2 Extend OTel/export and diagnostics bundle projections without embedding sensitive corpus input or reviewer text.
- [x] 3.3 Add parser compatibility tests proving legacy payloads and unknown future fields remain readable with documented defaults.

## 4. Replay and Contract Gates

- [x] 4.1 Add diagnostics replay fixtures for corpus success/version drift, Badcase reproducibility/unavailability/drift, rubric drift, aggregate idempotency/conflict, approval missing, and parity.
- [x] 4.2 Implement deterministic replay drift classifications and preserve expected/observed digests for mismatches.
- [x] 4.3 Extend the agent eval/tracing contract gate and quality gate with shell/PowerShell-equivalent assertions and fixture coverage.
- [x] 4.4 Add integration tests covering end-to-end correlation from corpus item through run/trace/result/feedback and ensuring no control-plane dependency.

## 5. Documentation and Examples

- [x] 5.1 Update `tracing-eval-smoke` MATRIX/README documentation first with semantic anchor, runtime path, expected markers, rollback notes, and Example Impact Assessment evidence.
- [x] 5.2 Add executable smoke assertions and fixtures for versioned corpus, Badcase replay, experiment comparison, and read-only feedback projection.
- [x] 5.3 Update runtime configuration/diagnostics docs, module boundaries, contract test index, README milestone snapshot, and roadmap status.

## 6. Verification and Delivery

- [x] 6.1 Run affected package tests and integration/replay tests, including positive/negative/boundary cases.
- [x] 6.2 Run `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml`.
- [x] 6.3 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`; resolve all failures. The module pins `toolchain go1.26.6`, the patched Go release for the standard-library advisories reported by `govulncheck`.
- [x] 6.4 Review diff for additive compatibility, architecture boundaries, secret/cardinality safety, and complete task evidence before archive.
