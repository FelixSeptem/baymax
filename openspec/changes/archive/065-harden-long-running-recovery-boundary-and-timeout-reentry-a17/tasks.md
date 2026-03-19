## 1. Recovery Boundary Model and Validation

- [x] 1.1 Add recovery-boundary config model with defaults: `resume_boundary=next_attempt_only`, `inflight_policy=no_rewind`, `timeout_reentry_policy=single_reentry_then_fail`, `timeout_reentry_max_per_task=1`.
- [x] 1.2 Wire recovery-boundary config loading with precedence `env > file > default`.
- [x] 1.3 Add fail-fast validation for unsupported boundary policy values.
- [x] 1.4 Ensure boundary policy auto-activation is tied to `recovery.enabled=true`.

## 2. Composer and Recovery Runtime Convergence

- [x] 2.1 Add next-attempt-only resume-boundary checks in composer recovery flow.
- [x] 2.2 Add no-rewind enforcement for restored terminal child tasks.
- [x] 2.3 Add timeout reentry budget tracking and single-reentry enforcement per task.
- [x] 2.4 Keep recovery conflict behavior aligned with existing `fail_fast` policy.
- [x] 2.5 Add unit/integration tests for boundary conflict classification and deterministic fail convergence.

## 3. Scheduler and Workflow Boundary Enforcement

- [x] 3.1 Enforce no-rewind restore semantics in scheduler restore/claim path.
- [x] 3.2 Enforce single timeout reentry then fail behavior for scheduler-managed long-running tasks.
- [x] 3.3 Ensure workflow resume path does not re-run terminal expanded steps under recovery boundary controls.
- [x] 3.4 Add regression tests for crash/restart continuation with boundary controls active.

## 4. Timeline Diagnostics and Compatibility

- [x] 4.1 Extend timeline mapping for recovery-boundary transitions using existing `recovery.*` and `scheduler.*` reasons only.
- [x] 4.2 Ensure recovery-boundary timeline events include required correlation fields (`run_id`, `task_id`, `attempt_id`) where applicable.
- [x] 4.3 Add additive diagnostics fields: `recovery_resume_boundary`, `recovery_inflight_policy`, `recovery_timeout_reentry_total`, `recovery_timeout_reentry_exhausted_total`.
- [x] 4.4 Add parser compatibility tests for `additive + nullable + default` behavior on new boundary fields.

## 5. Contract Matrix and Shared Gate

- [x] 5.1 Add integration contract tests for crash/restart/replay/timeout recovery-boundary matrix.
- [x] 5.2 Add Run/Stream equivalence tests for boundary-controlled recovery scenarios.
- [x] 5.3 Add replay-idempotency tests to ensure boundary enforcement does not inflate aggregates.
- [x] 5.4 Extend `tool/contributioncheck` assertions with recovery-boundary reason/field checks.
- [x] 5.5 Integrate A17 suites into `check-multi-agent-shared-contract.sh` and `.ps1`.

## 6. Documentation and Validation

- [x] 6.1 Update `README.md` with long-running recovery boundary semantics and default values.
- [x] 6.2 Update `docs/runtime-config-diagnostics.md` with boundary config and additive diagnostics fields.
- [x] 6.3 Update `docs/mainline-contract-test-index.md` with A17 recovery-boundary mapping rows.
- [x] 6.4 Update `docs/development-roadmap.md` with A17 scope and sequencing after A16.
- [x] 6.5 Run `go test ./...`.
- [x] 6.6 Run `go test -race ./...`.
- [x] 6.7 Run `golangci-lint run --config .golangci.yml`.
- [x] 6.8 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 6.9 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
