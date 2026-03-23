## 1. Readiness Contract Baseline

- [ ] 1.1 Define readiness result model (`ready|degraded|blocked`) and canonical finding schema fields.
- [ ] 1.2 Freeze readiness finding code taxonomy and domain/severity conventions for contract tests.
- [ ] 1.3 Add library-level readiness preflight API surface in runtime domain.
- [ ] 1.4 Ensure readiness API path is read-only and does not mutate run/scheduler state.

## 2. Runtime Config Controls

- [ ] 2.1 Add `runtime.readiness.enabled` with default `true` in runtime config schema.
- [ ] 2.2 Add `runtime.readiness.strict` with default `false` in runtime config schema.
- [ ] 2.3 Add `runtime.readiness.remote_probe_enabled` with default `false` in runtime config schema.
- [ ] 2.4 Wire env/file/default resolution for readiness controls with `env > file > default`.
- [ ] 2.5 Add startup validation for readiness controls and fail-fast behavior.
- [ ] 2.6 Add hot-reload validation and atomic rollback on invalid readiness updates.

## 3. Preflight Evaluation Engine

- [ ] 3.1 Implement readiness evaluator aggregation pipeline in runtime manager path.
- [ ] 3.2 Add local config validity preflight checks as readiness findings.
- [ ] 3.3 Add scheduler backend/fallback visibility checks and degraded classification.
- [ ] 3.4 Add mailbox backend/fallback visibility checks and degraded classification.
- [ ] 3.5 Add recovery backend activation/fallback checks and blocked classification on unrecoverable errors.
- [ ] 3.6 Implement strict escalation rule (`degraded -> blocked` when strict enabled).
- [ ] 3.7 Ensure repeated preflight calls with unchanged snapshot return semantically equivalent results.

## 4. Diagnostics Integration

- [ ] 4.1 Add additive readiness diagnostics fields (`runtime_readiness_status`, counts, primary code).
- [ ] 4.2 Record readiness findings summary through existing diagnostics write path.
- [ ] 4.3 Preserve additive + nullable + default compatibility semantics for new fields.
- [ ] 4.4 Ensure readiness diagnostics replay remains idempotent for equivalent events.
- [ ] 4.5 Extend diagnostics query assertions for readiness status/count visibility.

## 5. Composer Passthrough

- [ ] 5.1 Add composer-level readiness passthrough API for managed runtime path.
- [ ] 5.2 Ensure composer passthrough does not introduce new status taxonomy beyond runtime readiness.
- [ ] 5.3 Ensure composer readiness query is read-only for scheduler/task state.
- [ ] 5.4 Add equivalence assertions between composer readiness and runtime readiness outputs.

## 6. Tests And Contract Coverage

- [ ] 6.1 Add runtime config unit tests for readiness defaults and env/file override precedence.
- [ ] 6.2 Add runtime config manager tests for invalid readiness hot-reload rollback behavior.
- [ ] 6.3 Add unit tests for readiness status classification matrix (`ready/degraded/blocked`).
- [ ] 6.4 Add unit tests for strict escalation semantics.
- [ ] 6.5 Add diagnostics store tests for readiness additive field persistence and replay idempotency.
- [ ] 6.6 Add composer tests for readiness passthrough equivalence and read-only behavior.
- [ ] 6.7 Add integration contract tests for effective config parity and deterministic readiness results.
- [ ] 6.8 Add integration tests for fallback visibility mapping to degraded findings.

## 7. Gate And Documentation Alignment

- [ ] 7.1 Add readiness suites into `scripts/check-quality-gate.sh`.
- [ ] 7.2 Add readiness suites into `scripts/check-quality-gate.ps1`.
- [ ] 7.3 Update `docs/runtime-config-diagnostics.md` with readiness config and diagnostics fields.
- [ ] 7.4 Update `docs/mainline-contract-test-index.md` with readiness contract test mapping.
- [ ] 7.5 Update `docs/development-roadmap.md` with A40 scope/status mapping.
- [ ] 7.6 Update `README.md` milestone snapshot and readiness capability summary.

## 8. Validation

- [ ] 8.1 Run `go test ./runtime/config ./runtime/diagnostics ./orchestration/composer ./integration -count=1`.
- [ ] 8.2 Run `go test -race ./...`.
- [ ] 8.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 8.4 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 8.5 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 8.6 Run `openspec validate introduce-runtime-readiness-preflight-and-degradation-contract-a40 --strict`.
