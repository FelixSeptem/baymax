## 1. Readiness Contract Baseline

- [x] 1.1 Define readiness result model (`ready|degraded|blocked`) and canonical finding schema fields.
- [x] 1.2 Freeze readiness finding code taxonomy and domain/severity conventions for contract tests.
- [x] 1.3 Add library-level readiness preflight API surface in runtime domain.
- [x] 1.4 Ensure readiness API path is read-only and does not mutate run/scheduler state.

## 2. Runtime Config Controls

- [x] 2.1 Add `runtime.readiness.enabled` with default `true` in runtime config schema.
- [x] 2.2 Add `runtime.readiness.strict` with default `false` in runtime config schema.
- [x] 2.3 Add `runtime.readiness.remote_probe_enabled` with default `false` in runtime config schema.
- [x] 2.4 Wire env/file/default resolution for readiness controls with `env > file > default`.
- [x] 2.5 Add startup validation for readiness controls and fail-fast behavior.
- [x] 2.6 Add hot-reload validation and atomic rollback on invalid readiness updates.

## 3. Preflight Evaluation Engine

- [x] 3.1 Implement readiness evaluator aggregation pipeline in runtime manager path.
- [x] 3.2 Add local config validity preflight checks as readiness findings.
- [x] 3.3 Add scheduler backend/fallback visibility checks and degraded classification.
- [x] 3.4 Add mailbox backend/fallback visibility checks and degraded classification.
- [x] 3.5 Add recovery backend activation/fallback checks and blocked classification on unrecoverable errors.
- [x] 3.6 Implement strict escalation rule (`degraded -> blocked` when strict enabled).
- [x] 3.7 Ensure repeated preflight calls with unchanged snapshot return semantically equivalent results.

## 4. Diagnostics Integration

- [x] 4.1 Add additive readiness diagnostics fields (`runtime_readiness_status`, counts, primary code).
- [x] 4.2 Record readiness findings summary through existing diagnostics write path.
- [x] 4.3 Preserve additive + nullable + default compatibility semantics for new fields.
- [x] 4.4 Ensure readiness diagnostics replay remains idempotent for equivalent events.
- [x] 4.5 Extend diagnostics query assertions for readiness status/count visibility.

## 5. Composer Passthrough

- [x] 5.1 Add composer-level readiness passthrough API for managed runtime path.
- [x] 5.2 Ensure composer passthrough does not introduce new status taxonomy beyond runtime readiness.
- [x] 5.3 Ensure composer readiness query is read-only for scheduler/task state.
- [x] 5.4 Add equivalence assertions between composer readiness and runtime readiness outputs.

## 6. Tests And Contract Coverage

- [x] 6.1 Add runtime config unit tests for readiness defaults and env/file override precedence.
- [x] 6.2 Add runtime config manager tests for invalid readiness hot-reload rollback behavior.
- [x] 6.3 Add unit tests for readiness status classification matrix (`ready/degraded/blocked`).
- [x] 6.4 Add unit tests for strict escalation semantics.
- [x] 6.5 Add diagnostics store tests for readiness additive field persistence and replay idempotency.
- [x] 6.6 Add composer tests for readiness passthrough equivalence and read-only behavior.
- [x] 6.7 Add integration contract tests for effective config parity and deterministic readiness results.
- [x] 6.8 Add integration tests for fallback visibility mapping to degraded findings.

## 7. Gate And Documentation Alignment

- [x] 7.1 Add readiness suites into `scripts/check-quality-gate.sh`.
- [x] 7.2 Add readiness suites into `scripts/check-quality-gate.ps1`.
- [x] 7.3 Update `docs/runtime-config-diagnostics.md` with readiness config and diagnostics fields.
- [x] 7.4 Update `docs/mainline-contract-test-index.md` with readiness contract test mapping.
- [x] 7.5 Update `docs/development-roadmap.md` with A40 scope/status mapping.
- [x] 7.6 Update `README.md` milestone snapshot and readiness capability summary.

## 8. Validation

- [x] 8.1 Run `go test ./runtime/config ./runtime/diagnostics ./orchestration/composer ./integration -count=1`.
- [x] 8.2 Run `go test -race ./...`.
- [x] 8.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 8.4 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 8.5 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 8.6 Run `openspec validate introduce-runtime-readiness-preflight-and-degradation-contract-a40 --strict`.
