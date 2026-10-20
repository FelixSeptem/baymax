# Verification evidence

Date: 2026-10-20. Branch: `persist-native-continuations-and-restore-composite-graphs`.

## Delivered scope

The direct `native_continuation.v1` baseline covers persistent source material, atomic resume claims, new-Engine restore, and `input_required`, `approval_required`, and `budget_exhausted` in Run and Stream. Child/fanout, team, workflow, mailbox pause/restore, and planbridge resumed execution remain explicitly unsupported. Composite graph restore remains a roadmap follow-up; completion of this change does not close every BXM-4 cell.

## Current verification

| Command | Result | Evidence / tasks |
| --- | --- | --- |
| `go test ./core/runner ./core/types ./core/checkpoint ./orchestration/planbridge ./tool/diagnosticsreplay -count=1` | Passed | Tasks 1.1–3.2: DTO bounds/compatibility, source ownership/locking, corruption/expiry/conflict/unavailable, host denial/concurrent claim, successor isolation, three pause reasons, Run/Stream parity and read-only replay. |
| `go test ./adapter/scaffold -run '^TestGeneratedConformanceBootstrapOfflineExecutable$' -count=1` | Passed after cache permission retry | The generated offline module compiles the relocated Action Gate implementation from tracked `core/runner/control.go`; no new-file Git visibility dependency remains. |
| `go run ./examples/durable-continuation-restore` | Passed | Six `<reason>_<run\|stream>_restored` markers, plus duplicate/conflict/foreign/unavailable/composite/parity markers. |
| `openspec validate persist-native-continuations-and-restore-composite-graphs --strict` | Passed | Proposal, design and six spec deltas validate. |
| `git diff --check` | Passed | No whitespace errors. |
| Documentation impact, example impact, roadmap status and docs consistency gates | Passed inside full quality gate | All declared documentation paths are changed; public semantic declarations are consistent. |
| Go line budget and split semantic equivalence gates | Passed inside full quality gate | No content change to the existing line-budget exception CSV; Action Gate implementation moved to tracked `control.go` to keep `runner.go` within its original budget. |

`TestNativeContinuationCrossProcess` launches three separate OS processes per pause reason and mode (pause, restore, duplicate), covering six reason/mode combinations. It checks retained run identity and tool outcomes, original parent/effect binding, and no model/tool calls during duplicate projection.

## Environment and diagnostic reruns

- Default shared Go build cache access was denied for initial scaffold/agent-mode diagnostics. The scaffold test passed with normal cache permissions; subsequent gate execution uses the repository-local `.gocache-final` and `.gocache-lint` caches.
- `go test ./examples/agent-modes -count=1` was an invalid package target: its root Go files are excluded by build constraints. The repository's agent-mode smoke, stability and integration suites are the applicable verification; this command is not counted as a passing check.
- Earlier full-race execution exposed the offline scaffold's tracked-file bootstrap boundary after Action Gate was extracted into a new untracked file. The implementation was moved into existing tracked `control.go`, and the focused offline bootstrap regression above now passes.

## Final review corrections and verification

Two final review findings were reproduced with regression tests before correction:

- `TestClaimDetachesProposedBudgetFromCaller`: a public store claim retained the caller's budget pointer; mutating it could make a reopened snapshot corrupt. `ClaimResume` now copies the proposed budget before retaining the claim. The focused regression passes.
- `TestNativeContinuationNewEngineRestoresCompletedToolBoundary`: restore could replace runtime-configured concurrency with library defaults. Restore now resolves the saved policy against the matching current runtime configuration before continuing. Run and Stream both preserve three workers, a queue of seven, and reject backpressure; the focused regression passes.

| Command | Result | Evidence / tasks |
| --- | --- | --- |
| `pwsh -NoProfile -File scripts/check-native-continuation-contract.ps1` | Passed, exit 0 | Full affected package suites, source-owner dependency scan, golden/drift fixtures; the deliberately invalid terminal fixture fails as expected. Task 3.3. |
| `C:/Program Files/Git/bin/bash.exe scripts/check-native-continuation-contract.sh` | Passed, exit 0 | Same package suites and expected invalid-fixture rejection as PowerShell; shell/PowerShell continuation gate parity verified. Task 3.3. |
| `go test -race ./core/types ./core/checkpoint ./core/runner ./orchestration/planbridge ./tool/diagnosticsreplay -count=1` | Passed, exit 0 | Focused race coverage, including all three direct pause reasons across separate pause/restore/duplicate processes in Run and Stream. Tasks 5.1 and 5.4. |
| `go test -race ./... -count=1` | Passed, exit 0 | Full repository race run after both final review corrections; includes host and integration coverage. Task 5.1. |
| `openspec validate persist-native-continuations-and-restore-composite-graphs --strict` | Passed, exit 0 | Final proposal/design/spec validation. |
| `git diff --check` | Passed, exit 0 | Final review corrections have no whitespace errors. |

An earlier standalone `go test ./... -count=1` failed in the host-jsonl EOF test and runtime/config reload/cooldown tests during concurrent verification. Focused reruns of the EOF test and sandbox auto-freeze/unfreeze/capacity tests passed. The final full quality gate independently reran the prescribed full normal and race suites and passed; the earlier failure remains recorded as an intermittent concurrent-verification failure.

The PATH `bash` resolves to WSL launchers without an installed `/bin/bash`; using the explicit Git for Windows Bash executable completed the native continuation shell gate successfully. No shell parity limitation remains for task 3.3. Separate Git Bash runs of the host-resume and host-execution-plan gates failed to start under the sandbox with `couldn't create signal pipe, Win32 error 5`; approved unsandboxed retries passed with exit 0.

| Command | Result | Evidence / tasks |
| --- | --- | --- |
| `pwsh -NoProfile -File scripts/check-host-resume-admission-contract.ps1` | Passed, exit 0 | Expected replay drift rejected and offline example markers verified. Task 5.2. |
| `pwsh -NoProfile -File scripts/check-host-execution-plan-contract.ps1` | Passed, exit 0 | Unsupported resumed-plan boundary, replay drift and example markers verified. Task 5.2. |
| Git for Windows Bash: `scripts/check-host-resume-admission-contract.sh` and `scripts/check-host-execution-plan-contract.sh` | Passed, exit 0 | Approved retries after sandbox signal-pipe failures; host gate shell parity verified. Task 5.2. |
| `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` | Passed, exit 0 | Documentation consistency verified. Tasks 5.2 and 5.3. |
| `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | Passed, exit 0 | All 82 quality-gate steps passed, including full `go test ./...`, `go test -race ./...`, lint, docs consistency and the native continuation PowerShell gate. Task 5.3. |

The full gate reported no called-code vulnerabilities. The earlier standalone test failure is retained as diagnostic context, not represented as a passing result.

## Completed-task governance reconciliation

After all 21 tasks were checked, `openspec list --json` correctly changed this proposal to `complete`. A subsequent roadmap check rejected the stale in-progress entry. The roadmap now records implementation and verification complete, pending archive, while retaining every unsupported composite boundary. No implementation code changed during this reconciliation.

The roadmap status, documentation impact, example impact, and full docs consistency gates were rerun after that correction and passed with exit 0. Strict validation of this change and `git diff --check` also passed. At implementation handoff, the change remained on its feature branch, unarchived and uncommitted.

## Archive and specification synchronization

On 2026-10-20, `pwsh -NoProfile -File scripts/openspec-archive-seq.ps1 -ChangeName persist-native-continuations-and-restore-composite-graphs` completed with exit 0. Archive sequence 169 contains all 21 completed tasks. Six main specifications were synchronized (12 added requirements and one modified requirement), and `openspec list --json` reports no active changes.

Each affected main specification passed `openspec validate <name> --type spec --strict`: diagnostics-replay-tooling, durable-native-continuation-restore, dynamic-action-gate-and-native-run-resume, host-execution-plan-bridge, host-resume-admission-revalidation, and host-runtime-capability-manifest.

The broader `openspec validate --all --strict` returned 33 passed and 110 failed. The failures concern existing placeholder Purpose text in historical main specifications; the six specifications affected by this change passed. This repository-wide limitation is retained explicitly and is not represented as a successful full strict validation.

The first post-archive docs/quality runs rejected the stale README description of this change as in progress. README and roadmap now identify archive 169. The subsequent standalone docs consistency gate passed with exit 0. A fresh `go test ./...` before archival also passed with exit 0.

## Post-archive final gate results

The post-archive full `pwsh -NoProfile -File scripts/check-quality-gate.ps1` run passed steps 1–80, including the continuation/history/host contract gates, full normal tests (265.41 seconds), full race tests (307.19 seconds), lint (`0 issues`), and semantic stability. It returned exit 1 at step 81 because the 1800-second total budget left only 88 seconds for the performance regression gate. The process tree was stopped at that timeout; this invocation is not counted as a passing full quality-gate run.

The remaining checks were completed independently without changing runtime code or weakening their assertions:

| Command | Result | Evidence |
| --- | --- | --- |
| `pwsh -NoProfile -File scripts/check-a64-performance-regression.ps1` | Passed, exit 0 | Context, diagnostics query, and multi-agent benchmarks passed their original thresholds, including the nested shared-contract suite. |
| `pwsh -NoProfile -File scripts/check-full-chain-example-smoke.ps1` | Passed, exit 0 | `FULL_CHAIN_SUCCESS`, async/delayed/recovery markers, and Run/Stream terminal alignment. |
| `govulncheck ./...` | Passed, exit 0 | No called-code vulnerabilities. Two imported-package vulnerabilities and one required-module vulnerability were reported outside the detected call paths. The initial sandbox attempt could not reach the vulnerability database through its placeholder proxy; the approved network-enabled retry passed. |
| `git diff --cached --check` | Passed, exit 0 | Three trailing blank lines in archived deltas were removed before staging the final archive. |

All required checks therefore have successful evidence. The post-archive full-script timeout and repository-wide strict specification limitation remain recorded separately from those successful check results.
