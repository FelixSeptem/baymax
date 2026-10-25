# Verification Record

Date: 2026-10-25. Change: `enable-team-and-mailbox-native-continuations`.

## Focused contract evidence

| Command | Result |
| --- | --- |
| `go test ./orchestration/teams ./orchestration/mailbox ./orchestration/planbridge -count=1` | exit 0 |
| `go test -race ./orchestration/teams -count=1` | exit 0 |
| `go test -race ./orchestration/mailbox -count=1` | exit 0 |
| `go test -race ./orchestration/planbridge -count=1` | exit 0 |
| `pwsh -NoProfile -File scripts/check-team-mailbox-continuation-contract.ps1` | exit 0 |
| `pwsh -NoProfile -File scripts/check-host-execution-plan-contract.ps1` | exit 0 |
| `openspec validate enable-team-and-mailbox-native-continuations --strict` | exit 0 |
| `pwsh -NoProfile -File scripts/check-openspec-documentation-impact.ps1` | exit 0 |
| `pwsh -NoProfile -File scripts/check-openspec-example-impact-declaration.ps1` | exit 0 |
| `pwsh -NoProfile -File scripts/check-openspec-roadmap-status-consistency.ps1` | exit 0 |
| `git diff --check` | exit 0 |

The focused suites include FileStore sequential reopen, memory-store deployment
rejection, malformed/foreign/expiry/conflict checks, same-continuation
duplicate projection, team parallel/vote Run/Stream parity, original deadline
failure before dispatch, mailbox lease loss/retry/DLQ/timeout projection, and
read-only history replay. `ReadAfter` evidence verifies source stats/snapshot
remain unchanged and that replay does not execute any lifecycle or execution
operation.

## Shell-gate environment limitation

`bash scripts/check-team-mailbox-continuation-contract.sh` was attempted on
2026-10-25 and exited 1 before script execution because the available WSL Bash
service returned `Bash/Service/CreateInstance/E_ACCESSDENIED`. The PowerShell
equivalent passed. This is recorded as an environment permission limitation;
it does not constitute a passing shell-gate result and no assertion was
downgraded or bypassed.

## Full verification

All repository-wide verification commands reached terminal exit code 0 on
2026-10-25:

| Command | Result |
| --- | --- |
| `go test ./... -count=1` | exit 0 |
| `go test -race ./... -count=1` | exit 0 |
| `golangci-lint run --config .golangci.yml` | exit 0 (`0 issues.`) |
| `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | exit 0; `QUALITY_GATE_EXIT=0` |
| `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` | exit 0 |
| `openspec validate --all` | exit 0 |
| `git diff --check` | exit 0 |

The quality gate used its documented, verification-only timeout configuration
(`BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=3600` and
`BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=900`) after the default 1800-second
aggregate budget expired while its final repository race suite was still
running. The completed configured run executed 88 steps in 2303.81 seconds,
including successful full `go test ./...`, full `go test -race ./...`, lint,
semantic and performance gates, example smoke, and strict vulnerability scan.
The timeout adjustment did not suppress or bypass a failing assertion.

During this verification a roadmap/README status-parity parser defect was also
repaired: descriptive backticks in the active roadmap item made the parser read
the wrong change slug, and the README active-change entry lacked its canonical
parsable bullet. The descriptions now carry the canonical active change name
without nested backtick ambiguity;
`TestReleaseStatusParityDocsConsistency` passes as part of the full suite.
