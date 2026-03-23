## 1. Contract Baseline And API Surface

- [ ] 1.1 Freeze A39 action/state matrix (`cancel`: `queued|awaiting_report`; `retry_terminal`: `failed|dead_letter`) and map to scheduler state model.
- [ ] 1.2 Define Task Board control request/response DTOs in scheduler domain, including required `operation_id`.
- [ ] 1.3 Define normalized control action enums and fail-fast validation errors for unsupported actions/states.
- [ ] 1.4 Ensure query path remains read-only and control path is isolated from query mutation logic.

## 2. Scheduler Manual Control Implementation

- [ ] 2.1 Add scheduler public control entrypoint (library API) for manual actions.
- [ ] 2.2 Implement `cancel` transition for `queued` tasks with deterministic terminalization semantics.
- [ ] 2.3 Implement `cancel` transition for `awaiting_report` tasks with deterministic lifecycle cleanup.
- [ ] 2.4 Implement `cancel` fail-fast path for `running` tasks (no force-kill semantics).
- [ ] 2.5 Implement `retry_terminal` transition for `failed|dead_letter -> queued`.
- [ ] 2.6 Implement manual retry budget enforcement with `max_manual_retry_per_task=3`.
- [ ] 2.7 Add `operation_id` idempotency dedup store/path and replay-safe convergence behavior.
- [ ] 2.8 Preserve unaffected enqueue/claim/heartbeat/requeue/commit behavior under control operations.

## 3. Reason Taxonomy And Timeline Wiring

- [ ] 3.1 Extend scheduler canonical reason set with `scheduler.manual_cancel` and `scheduler.manual_retry`.
- [ ] 3.2 Emit `scheduler.manual_cancel` timeline events for successful cancel transitions.
- [ ] 3.3 Emit `scheduler.manual_retry` timeline events for successful retry transitions.
- [ ] 3.4 Ensure control failure paths do not emit non-canonical or cross-namespace reason codes.

## 4. Runtime Config And Diagnostics

- [ ] 4.1 Extend `runtime/config` with `scheduler.task_board.control.enabled` and `scheduler.task_board.control.max_manual_retry_per_task`.
- [ ] 4.2 Add startup validation and hot-reload validation for task-board control fields.
- [ ] 4.3 Ensure invalid hot reload rolls back atomically to previous valid snapshot.
- [ ] 4.4 Extend diagnostics aggregates with manual-control additive fields (`total/success/rejected/idempotent_dedup`).
- [ ] 4.5 Add action-level breakdown for manual control aligned to canonical reasons.
- [ ] 4.6 Ensure diagnostics replay remains idempotent for duplicated `operation_id` submissions.

## 5. Unit And Integration Contract Tests

- [ ] 5.1 Add scheduler unit tests for action validation and state-matrix fail-fast behavior.
- [ ] 5.2 Add scheduler unit tests for `cancel` semantics on `queued|awaiting_report` and reject on `running`.
- [ ] 5.3 Add scheduler unit tests for `retry_terminal` semantics and manual retry budget enforcement.
- [ ] 5.4 Add scheduler unit tests for `operation_id` idempotency and duplicate replay stability.
- [ ] 5.5 Add timeline reason taxonomy tests covering `scheduler.manual_cancel` and `scheduler.manual_retry`.
- [ ] 5.6 Add integration contract tests for memory/file backend parity under equivalent control requests.
- [ ] 5.7 Add integration contract tests for Run/Stream semantic equivalence with manual control actions.
- [ ] 5.8 Add integration replay tests ensuring additive diagnostics counters do not inflate under duplicate control events.

## 6. Gate And Documentation Alignment

- [ ] 6.1 Add task-board manual-control suites to `scripts/check-multi-agent-shared-contract.sh`.
- [ ] 6.2 Add task-board manual-control suites to `scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 6.3 Ensure `scripts/check-quality-gate.sh` and `.ps1` cover the new shared suites as blocking checks.
- [ ] 6.4 Update `docs/runtime-config-diagnostics.md` with new config fields and diagnostics fields.
- [ ] 6.5 Update `docs/mainline-contract-test-index.md` with A39 contract rows and gate mapping.
- [ ] 6.6 Update `docs/development-roadmap.md` and `README.md` with A39 scope/status snapshot.

## 7. Validation And Release Readiness

- [ ] 7.1 Run `go test ./orchestration/scheduler ./runtime/config ./runtime/diagnostics ./integration -count=1`.
- [ ] 7.2 Run `go test -race ./...`.
- [ ] 7.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 7.4 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 7.5 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 7.6 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 7.7 Run `openspec validate introduce-task-board-control-and-manual-recovery-contract-a39 --strict`.
