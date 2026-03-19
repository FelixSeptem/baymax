## 1. Unified Query API Model and Validation

- [ ] 1.1 Add unified diagnostics query request/response models covering `run_id`, `team_id`, `workflow_id`, `task_id`, `status`, and `time_range`.
- [ ] 1.2 Implement query validation for page-size bounds, sort defaults, cursor format, and time-range validity with fail-fast behavior.
- [ ] 1.3 Implement pagination defaults (`page_size=50`), max bound (`200`), and default sort (`time desc`) in query execution path.
- [ ] 1.4 Implement opaque cursor encode/decode and deterministic cursor page traversal.

## 2. Runtime Diagnostics Integration and Compatibility

- [ ] 2.1 Integrate unified query execution in diagnostics store with multi-filter `AND` semantics.
- [ ] 2.2 Expose unified query entrypoint from runtime diagnostics manager without adding feature flag.
- [ ] 2.3 Keep `RecentRuns`, `RecentCalls`, and `RecentSkills` compatibility behavior unchanged.
- [ ] 2.4 Enforce `task_id` no-match semantics as empty result set without error.

## 3. Contract Tests and Integration Coverage

- [ ] 3.1 Add unit contract tests for multi-filter `AND` behavior and default `time desc` ordering.
- [ ] 3.2 Add unit contract tests for pagination defaults and page-size limit fail-fast (`>200` and invalid lower bound).
- [ ] 3.3 Add unit contract tests for opaque cursor stability and invalid cursor fail-fast behavior.
- [ ] 3.4 Add integration contract tests for unmatched `task_id` empty-set semantics and replay-idempotent query summaries.

## 4. Shared Gate and Contract Index Alignment

- [ ] 4.1 Add unified query suites into `scripts/check-multi-agent-shared-contract.sh`.
- [ ] 4.2 Add unified query suites into `scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 4.3 Update `tool/contributioncheck` and `docs/mainline-contract-test-index.md` for unified query traceability mapping.
- [ ] 4.4 Ensure gate failures classify unified query semantic drift with explicit failure reasons.

## 5. Documentation and Delivery

- [ ] 5.1 Update `README.md` with unified diagnostics query usage and compatibility notes.
- [ ] 5.2 Update `docs/runtime-config-diagnostics.md` with filters, pagination, sorting, cursor, and error semantics.
- [ ] 5.3 Update `docs/development-roadmap.md` with A18 scope and sequencing status.

## 6. Validation

- [ ] 6.1 Run `go test ./...`.
- [ ] 6.2 Run `go test -race ./...`.
- [ ] 6.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 6.4 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 6.5 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 6.6 Run `openspec validate introduce-unified-run-team-workflow-task-query-api-a18 --strict`.

