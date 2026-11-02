# Verification

Validated on 2026-11-02 before archival.

| Check | Result |
| --- | --- |
| `go test ./...` | passed |
| `go test -race ./...` | passed |
| `golangci-lint run --config .golangci.yml` | passed, `0 issues.` |
| `pwsh -NoProfile -File scripts/check-host-resume-admission-contract.ps1` | passed before the final replay-order review correction; targeted replay was rerun after that correction |
| `go test ./tool/diagnosticsreplay -run TestHostResumeReplay -count=1` | passed after the replay-order and sentinel-collision regressions |
| `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` | passed |
| `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | passed: 90 steps, 1190.87s; includes ordinary/race full suites, lint and vulnerability scan |
| `pwsh -NoProfile -File scripts/check-openspec-documentation-impact.ps1` | passed |
| `openspec validate --all` | passed: 150 items, 0 failed |

Independent review found a replay reason-precedence mismatch for combined provenance and existing admission failures. The replay now reconstructs the bounded synthetic facts and invokes `types.ValidateHostResumeResult` once, so it retains live validation order. The review also identified a synthetic sentinel collision for `checkpoint_id == "changed"`; a collision regression covers the alternative sentinel.

An earlier concurrent gate invocation briefly received shared-cache access denial. The final complete quality gate was run with the repository's configured shared cache and passed.
