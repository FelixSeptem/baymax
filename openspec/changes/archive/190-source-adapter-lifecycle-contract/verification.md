# Source adapter lifecycle verification

Validated 2026-11-02 before archival.

| Check | Result |
| --- | --- |
| Initial source/state/file/replay/example TDD | Missing implementation observed RED before implementation; focused suites GREEN. |
| `go test ./... -timeout 15m` | passed; also passed sequentially inside final quality gate |
| `go test -race ./...` | passed sequentially inside final quality gate |
| `golangci-lint run --config .golangci.yml` | passed, 0 issues; rerun after final integration addition |
| `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | passed, 91 steps, 1438.03s / 1800s; includes normal/race/lint, dedicated gate, smoke and strict vulnerability scan |
| `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` | passed |
| `pwsh -NoProfile -File scripts/check-openspec-documentation-impact.ps1` | passed |
| `openspec validate --all` | 150 passed, 0 failed before archive; existing unrelated Purpose warnings remain |
| Dedicated `check-source-adapter-lifecycle-contract.ps1` and `.sh` | passed after final integration addition; intentional invalid fixture exits 1 with `source_adapter.replay_drift`; all six example markers present |
| `go test ./integration ./tool/contributioncheck -run TestSourceAdapter -count=1` | passed after adding Observer/source boundary integration and gate wiring |
| `go test -race ./integration -run TestSourceAdapter -count=1` | passed after final integration addition |

Independent review caught permissive decoding into initialized file state and
revision exhaustion before the second CAS. File decode now begins from zero,
and Prepare reserves both revisions before effect; regression tests cover both.
A subsequent review caught a missing integration file claimed by the index.
The new test exercises real native runner/Observer isolation and injected source,
file reopen and host CAS across confirmation failure. Both gates include it.
The gate-wiring regression first failed for missing `./integration`, then passed.
Final independent review reports no remaining Critical or Important findings.

An early full race run overlapped a normal suite. Contributioncheck tests
temporarily modify shared tracked example fixtures; the processes interfered.
This was not established as a runtime or baseline defect. The final quality gate
ran full normal and race suites sequentially and both passed. Future full suites
and fixture-modifying gates must not overlap in one worktree.

All checks used shared machine caches, never new repository cache directories.
Git Bash initially failed under the sandbox; its dedicated gate subsequently
passed outside the sandbox. No feature or contract check was skipped.

Main-spec sync assessment: add `source-adapter-lifecycle` with explicit Purpose
and six requirements; append the source/Observer independence requirement to
`native-execution-event-adapter`. Archive script performs that synchronization.
Deployment remains process-local memory or same-host single-writer sequential
file reopen, with cooperative cancellation and host-owned effect CAS.

Delivery follows verification: archive, commit, merge latest master, verify merged
packages/docs/OpenSpec, push, verify remote SHA, then remove only the merged
feature branch/worktree. Unrelated root modifications remain preserved.
