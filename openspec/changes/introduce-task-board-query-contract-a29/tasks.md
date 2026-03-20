## 1. Task Board Query Contract Model

- [ ] 1.1 Add `TaskBoardQueryRequest/TaskBoardQueryResult` models in scheduler query surface with canonical filter fields.
- [ ] 1.2 Implement request validation for `state` enum, `time_range`, `page_size`, `sort`, and cursor boundary checks with fail-fast semantics.
- [ ] 1.3 Add normalized defaults: `page_size=50`, max `200`, sort `updated_at desc`.

## 2. Scheduler Query Execution

- [ ] 2.1 Implement `QueryTasks` read path based on scheduler snapshot to preserve backend-agnostic behavior.
- [ ] 2.2 Implement deterministic `AND` filtering for `task_id/run_id/workflow_id/team_id/state/priority/agent_id/peer_id/parent_run_id/time_range`.
- [ ] 2.3 Implement stable sorting (`updated_at|created_at`) and opaque cursor pagination with query-hash binding.
- [ ] 2.4 Ensure query path is read-only and does not mutate queue/task runtime state.

## 3. Contract Tests and Integration Coverage

- [ ] 3.1 Add scheduler unit contract tests for filter semantics, defaults, and no-match-empty-set behavior.
- [ ] 3.2 Add scheduler unit tests for invalid inputs (state/page_size/time_range/sort/cursor) fail-fast behavior.
- [ ] 3.3 Add scheduler unit tests for cursor determinism and query-boundary mismatch rejection.
- [ ] 3.4 Add integration contract tests for memory/file backend parity and restore/replay semantic stability.

## 4. Gate and Documentation Alignment

- [ ] 4.1 Add task-board contract suites into `scripts/check-multi-agent-shared-contract.sh`.
- [ ] 4.2 Add task-board contract suites into `scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 4.3 Update `docs/mainline-contract-test-index.md` with Task Board query contract mappings.
- [ ] 4.4 Update `README.md`, `docs/runtime-config-diagnostics.md`, and `docs/development-roadmap.md` with Task Board scope and non-goals.

## 5. Validation

- [ ] 5.1 Run `go test ./...`.
- [ ] 5.2 Run `go test -race ./...`.
- [ ] 5.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 5.4 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 5.5 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 5.6 Run `openspec validate introduce-task-board-query-contract-a29 --strict`.
