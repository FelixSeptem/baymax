# Verification — Host Context Assembly / Handoff Contract

Verified on 2026-10-18 in worktree `D:\code\baymax\.worktrees\host-context-assembly-handoff-contract` on branch `feature/host-context-assembly-handoff-contract`.

## Focused contract evidence

| Command | Result |
| --- | --- |
| `go test ./context/hostprojection ./context/assembler -count=1` | exit 0; validates descriptor admission, resolver ordering, detached snapshots, bounds, privacy, and assembler/handoff integration. |
| `go test ./core/runner -count=1` | exit 0; validates direct Run/Stream parity and explicit caller-owned projected-handoff restore. |
| `go test ./tool/diagnosticsreplay ./tool/contributioncheck ./examples/host-context-assembly -count=1` | exit 0; validates deterministic reference-only replay, dual-platform gate equivalence/ownership scans, and all offline example markers. |
| `pwsh -NoProfile -File scripts/check-host-context-assembly-handoff-contract.ps1` | exit 0 as quality-gate step 75; its focused host-projection, assembler, runner, replay, example, and contribution suites all reported `ok` and the gate reported `passed`. |

## Standard quality and documentation evidence

| Command | Result |
| --- | --- |
| `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | completed successfully on the 1800-second budget; log `C:\Users\whenh\AppData\Local\Temp\baymax-host-context-quality-gate-1800.log` ends with `[quality-gate] done`, `steps_completed=79`, and `total_elapsed=1114.19s`. It includes `go test ./...` (step 73), `go test -race ./...` (step 74), the host-context gate (step 75), and lint reporting `0 issues.` (step 76). The earlier 900-second invocation timed out at the final lint step; it is not used as passing evidence. |
| `golangci-lint run --config .golangci.yml ./...` | executed by quality-gate step 76 and reported `0 issues.`. The explicit package target is required because `golangci-lint v2.11.3` rejects the Go-file-free repository root. |
| `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` | exit 0; includes documentation-impact, semantic-labelling, and active-roadmap consistency checks. |
| `openspec validate establish-host-context-assembly-handoff-contract --strict` | exit 0; `Change 'establish-host-context-assembly-handoff-contract' is valid`. |
| `git diff --check` | exit 0. |

## Quality-gate correction and regression protection

The initial full quality invocation reached its final lint step with only 31 seconds left in its old 900-second total budget, then timed out. Investigation reproduced the lint command independently: `golangci-lint v2.11.3` fails at the repository root because that directory contains no Go source package, while `golangci-lint run --config .golangci.yml ./...` completes cleanly.

The correction is deliberately narrow:

- both quality-gate implementations target `./...` for lint;
- the PowerShell full-gate default total budget is 1800 seconds, consistent with the observed 1114.19-second full-chain duration;
- `TestQualityGateKeepsHostContextLintReachable` was written first, observed failing before the correction, and now passes through the focused contribution suite.

## Cross-platform gate note

The PowerShell host-context gate was executed above. The equivalent shell script was not runnable in this Windows environment because Git Bash fails even for `bash --version` and a minimal `printf` command before script execution with `couldn't create signal pipe, Win32 error 5` (exit `-1073741502`); WSL Bash likewise fails before command execution with `Bash/Service/CreateInstance/E_ACCESS_DENIED` (exit 1). This is recorded as an unavailable platform launcher, not as a passing shell gate.

The required shell-side unavailable reporting and parity are still statically exercised by `TestHostContextProjectionGateScriptsAreEquivalentAndBlocking`, which passed in the focused contribution suite. Both gate files contain the required `unavailable` path and the same package/test anchors. No contract test was silently skipped or represented as a successful shell execution.

## Scope, risks, and rollback

- The contract is opt-in and direct Run/Stream admission only applies when the existing Context Assembler is enabled; the legacy no-projection and assembler-bypass behavior remains unchanged.
- Resolver material is authorized at use time and is transient. Projection and handoff digests establish integrity only; neither is business authorization.
- Handoff/replay retain opaque references and bounded correlation only. No resolved body, context store, budget write-back, provider SDK, direct recorder write, or competing continuation/terminal owner was introduced.
- The line-budget baselines were updated narrowly for the already-existing semantic owners: `context/assembler/assembler.go` (1517 lines) and `core/runner/runner.go` (5919 lines). The exception entries document this scoped increase.
- Rollback is operational: stop supplying `HostContextProjection` and `HostContextResolver`. Existing assembler and handoff paths continue; rollback does not claim to undo any host-owned business effect.

## Generated artifacts

The full quality gate refreshed `.artifacts/a64/harnessability-scorecard.json` and `.artifacts/a64/impacted-full-report.json`. Their observed changes are generation timestamp, changed-file count, and JSON map ordering; no contract assertion changed. They are retained as gate outputs and are not manually reset in this implementation change.
