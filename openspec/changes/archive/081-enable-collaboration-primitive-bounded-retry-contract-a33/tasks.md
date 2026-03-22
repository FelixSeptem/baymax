## 1. Preconditions and Contract Baseline

- [x] 1.1 Confirm existing collaboration primitive contracts/tests (A16 baseline) are green before introducing retry-enabled semantics.
- [x] 1.2 Confirm scheduler-managed retry ownership boundaries to prevent primitive+scheduler compounded retries.
- [x] 1.3 Add/adjust contract fixtures for retry policy defaults and error-layer classification mapping.

## 2. Runtime Config and Validation

- [x] 2.1 Extend `runtime/config` with `composer.collab.retry.max_attempts|backoff_initial|backoff_max|multiplier|jitter_ratio|retry_on`.
- [x] 2.2 Set defaults to recommended values (`enabled=false`, `max_attempts=3`, `backoff_initial=100ms`, `backoff_max=2s`, `multiplier=2.0`, `jitter_ratio=0.2`, `retry_on=transport_only`).
- [x] 2.3 Add startup and hot-reload fail-fast validation for invalid retry bounds and invalid `retry_on` values.
- [x] 2.4 Ensure invalid hot reload rolls back atomically to previous valid snapshot.

## 3. Collaboration Retry Execution

- [x] 3.1 Implement bounded exponential-backoff+jitter retry executor in `orchestration/collab` behind `composer.collab.retry.enabled`.
- [x] 3.2 Apply retry only to transport-retryable failures and skip retry for protocol/semantic failures by default.
- [x] 3.3 Scope retry to `delegation sync` and `async submit` stages only, excluding accepted-after async-await convergence stages.
- [x] 3.4 Enforce single-owner retry behavior for scheduler-managed paths to avoid compounded retries.

## 4. Diagnostics and Observability Alignment

- [x] 4.1 Add additive diagnostics fields for collaboration retry aggregates (`collab_retry_total`, `collab_retry_success_total`, `collab_retry_exhausted_total`).
- [x] 4.2 Ensure collaboration retry diagnostics remain replay-idempotent and compatibility-window safe (`additive + nullable + default`).
- [x] 4.3 Emit timeline/diagnostic markers that distinguish retry-attempt, retry-success, and retry-exhausted outcomes.

## 5. Contract Tests and Shared Gate

- [x] 5.1 Add unit tests for retry policy evaluation, backoff bounds, and retry classification behavior.
- [x] 5.2 Add integration contract tests for retry-disabled default, retry-enabled convergence, and scheduler no-double-retry behavior.
- [x] 5.3 Add Run/Stream equivalence and replay-idempotency tests for collaboration retry-enabled flows.
- [x] 5.4 Integrate collaboration retry suites into `scripts/check-multi-agent-shared-contract.sh` and `scripts/check-multi-agent-shared-contract.ps1` as blocking checks.

## 6. Documentation and Mapping

- [x] 6.1 Update `README.md` and orchestration docs to describe collaboration retry defaults, scope, and non-goals.
- [x] 6.2 Update `docs/runtime-config-diagnostics.md` with retry config fields, validation rules, and diagnostics semantics.
- [x] 6.3 Update `docs/mainline-contract-test-index.md` with collaboration retry contract rows and gate paths.
- [x] 6.4 Update `docs/development-roadmap.md` with A33 scope and status mapping.

## 7. Validation

- [x] 7.1 Run `go test ./...`.
- [x] 7.2 Run `go test -race ./...`.
- [x] 7.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 7.4 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 7.5 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [x] 7.6 Run `openspec validate enable-collaboration-primitive-bounded-retry-contract-a33 --strict`.

