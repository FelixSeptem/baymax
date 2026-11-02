# Verification Record

Verified locally on 2026-11-02 from `feat/common-execution-graph-contract`.

## Focused Contract Evidence

| Command | Result |
| --- | --- |
| `go test ./orchestration/graph` | passed |
| `go test ./orchestration/workflow ./orchestration/teams ./orchestration/composer ./core/checkpoint ./core/runner -count=1` | passed |
| `go test ./examples/execution-graph -run TestOfflineExampleEmitsContractMarkers -count=1` | passed |
| `go test ./tool/contributioncheck -run ExecutionGraph -count=1` | passed |
| `pwsh -File scripts/check-execution-graph-contract.ps1` | passed |
| `C:\Program Files\Git\bin\bash.exe scripts/check-execution-graph-contract.sh` | passed |

The unprivileged shared Go cache denied access for the focused continuation command. The same command was re-run with the repository-approved elevated execution path and passed. Git Bash initially failed to create its Windows signal pipe under the sandbox and the shell gate was re-run with the same approved execution path; it passed. Neither condition changed source code or verification scope.

## Required Repository Gates

| Command | Result |
| --- | --- |
| `go test ./...` | passed |
| `go test -race ./...` | passed |
| `golangci-lint run --config .golangci.yml` | passed (`0 issues`) |
| `pwsh -File scripts/check-docs-consistency.ps1` | passed |
| `openspec validate --all` | passed (`149 passed, 0 failed`; pre-existing warnings only) |
| `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` | passed |
| `pwsh -File scripts/check-openspec-documentation-impact.ps1` | passed |
| `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` | passed |
| `pwsh -File scripts/check-quality-gate.ps1` | passed (90 steps; full tests, race, lint, contract gates, smoke tests, and strict vulnerability scan) |

The standard quality gate contains intentional negative replay-drift checks. Their child commands print expected failures to prove drift rejection, while the corresponding parent contract steps and the top-level quality gate exit successfully.
