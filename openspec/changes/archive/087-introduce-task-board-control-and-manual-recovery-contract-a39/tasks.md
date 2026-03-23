## 1. Contract Baseline And API Surface

- [x] 1.1 Freeze A39 action/state matrix (`cancel`: `queued|awaiting_report`; `retry_terminal`: `failed|dead_letter`) and map to scheduler state model.
- [x] 1.2 Define Task Board control request/response DTOs in scheduler domain, including required `operation_id`.
- [x] 1.3 Define normalized control action enums and fail-fast validation errors for unsupported actions/states.
- [x] 1.4 Ensure query path remains read-only and control path is isolated from query mutation logic.

## 2. Scheduler Manual Control Implementation

- [x] 2.1 Add scheduler public control entrypoint (library API) for manual actions.
- [x] 2.2 Implement `cancel` transition for `queued` tasks with deterministic terminalization semantics.
- [x] 2.3 Implement `cancel` transition for `awaiting_report` tasks with deterministic lifecycle cleanup.
- [x] 2.4 Implement `cancel` fail-fast path for `running` tasks (no force-kill semantics).
- [x] 2.5 Implement `retry_terminal` transition for `failed|dead_letter -> queued`.
- [x] 2.6 Implement manual retry budget enforcement with `max_manual_retry_per_task=3`.
- [x] 2.7 Add `operation_id` idempotency dedup store/path and replay-safe convergence behavior.
- [x] 2.8 Preserve unaffected enqueue/claim/heartbeat/requeue/commit behavior under control operations.

## 3. Reason Taxonomy And Timeline Wiring

- [x] 3.1 Extend scheduler canonical reason set with `scheduler.manual_cancel` and `scheduler.manual_retry`.
- [x] 3.2 Emit `scheduler.manual_cancel` timeline events for successful cancel transitions.
- [x] 3.3 Emit `scheduler.manual_retry` timeline events for successful retry transitions.
- [x] 3.4 Ensure control failure paths do not emit non-canonical or cross-namespace reason codes.

## 4. Runtime Config And Diagnostics

- [x] 4.1 Extend `runtime/config` with `scheduler.task_board.control.enabled` and `scheduler.task_board.control.max_manual_retry_per_task`.
- [x] 4.2 Add startup validation and hot-reload validation for task-board control fields.
- [x] 4.3 Ensure invalid hot reload rolls back atomically to previous valid snapshot.
- [x] 4.4 Extend diagnostics aggregates with manual-control additive fields (`total/success/rejected/idempotent_dedup`).
- [x] 4.5 Add action-level breakdown for manual control aligned to canonical reasons.
- [x] 4.6 Ensure diagnostics replay remains idempotent for duplicated `operation_id` submissions.

## 5. Unit And Integration Contract Tests

- [x] 5.1 Add scheduler unit tests for action validation and state-matrix fail-fast behavior.
- [x] 5.2 Add scheduler unit tests for `cancel` semantics on `queued|awaiting_report` and reject on `running`.
- [x] 5.3 Add scheduler unit tests for `retry_terminal` semantics and manual retry budget enforcement.
- [x] 5.4 Add scheduler unit tests for `operation_id` idempotency and duplicate replay stability.
- [x] 5.5 Add timeline reason taxonomy tests covering `scheduler.manual_cancel` and `scheduler.manual_retry`.
- [x] 5.6 Add integration contract tests for memory/file backend parity under equivalent control requests.
- [x] 5.7 Add integration contract tests for Run/Stream semantic equivalence with manual control actions.
- [x] 5.8 Add integration replay tests ensuring additive diagnostics counters do not inflate under duplicate control events.

## 6. Gate And Documentation Alignment

- [x] 6.1 Add task-board manual-control suites to `scripts/check-multi-agent-shared-contract.sh`.
- [x] 6.2 Add task-board manual-control suites to `scripts/check-multi-agent-shared-contract.ps1`.
- [x] 6.3 Ensure `scripts/check-quality-gate.sh` and `.ps1` cover the new shared suites as blocking checks.
- [x] 6.4 Update `docs/runtime-config-diagnostics.md` with new config fields and diagnostics fields.
- [x] 6.5 Update `docs/mainline-contract-test-index.md` with A39 contract rows and gate mapping.
- [x] 6.6 Update `docs/development-roadmap.md` and `README.md` with A39 scope/status snapshot.

## 7. Validation And Release Readiness

- [x] 7.1 Run `go test ./orchestration/scheduler ./runtime/config ./runtime/diagnostics ./integration -count=1`.
- [x] 7.2 Run `go test -race ./...`.
- [x] 7.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 7.4 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [x] 7.5 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 7.6 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 7.7 Run `openspec validate introduce-task-board-control-and-manual-recovery-contract-a39 --strict`.
