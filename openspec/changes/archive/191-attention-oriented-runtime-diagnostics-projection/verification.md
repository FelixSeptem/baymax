# Attention diagnostics verification

Validated 2026-11-03. The final full quality gate exited successfully with
92 completed steps in 1302.15 seconds; all implementation checks are complete.

| Check | Observed result |
| --- | --- |
| Initial projection, replay and gate wiring TDD | Missing implementation observed RED before implementation; focused tests GREEN. |
| `go test ./runtime/diagnostics ./tool/diagnosticsreplay ./integration -run TestAttention -count=1` | passed |
| `go test -race ./runtime/diagnostics ./tool/diagnosticsreplay ./integration -run TestAttention -count=1` | passed |
| `pwsh -NoProfile -File scripts/check-attention-diagnostics-contract.ps1` | passed, including expected negative fixture exit and example markers |
| Git Bash `scripts/check-attention-diagnostics-contract.sh` | passed, including expected `attention.replay_drift` failure |
| `pwsh -NoProfile -File scripts/check-openspec-documentation-impact.ps1` | passed |
| `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` | passed |
| `openspec validate attention-oriented-runtime-diagnostics-projection --strict` | passed |
| `openspec validate --all --json` | 151 items passed, 0 failed before archive |
| `git diff --check` | passed |
| First full `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | ordinary and race full suites passed; lint rejected a missing test-slice preallocation; corrected without behavior changes |
| `golangci-lint run --config .golangci.yml` after correction | passed, 0 issues |
| Final full `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | passed, exit 0; 92 steps, 1302.15s; full ordinary 266.39s, full race 269.88s, lint 0 issues, semantic/performance/example gates passed; strict govulncheck found 0 reachable vulnerabilities |

Independent review identified an important mismatch between hypothetical
capability strings and actual producers, and a Unicode JSON-key alias bypass.
Regression tests reproduced both failures before their fixes. The projection
now recognizes `protocol.capability.missing_required` and provider `blocked`,
and replay rejects non-ASCII keys before decoding. Real Composer admission tests
cover Run/Stream capability rejection, unknown catalog model and credential-only
rejection without invoking a provider. The reviewer subsequently accepted both
fixes and reported no remaining Critical or Important findings.

An additional `openspec validate --all --strict` check fails on inherited main
spec Purpose placeholders. Required non-strict global validation passes; the new
change passes strict validation. Historical specs are outside this change.

The first full gate fell back to a worktree-local `.gocache` because sandboxed
access to the configured shared directory was unavailable. The final gate runs
with shared-cache access and explicit machine cache paths. Existing cache/output
files will be preserved during worktree cleanup. Full suites run sequentially
because contributioncheck temporarily changes tracked example fixtures. Gate
generated A64 reports are excluded from the feature commit.

The projection is a pure read model over a caller snapshot. Host references are
explicit evidence, never authorization, durable effect proof or current source
facts inferred from a terminal record. Pagination and totals are snapshot-local.
Rollback removes the caller's projection invocation; no execution or owner store
is changed. Archive synchronization adds the attention-diagnostics-projection
main spec through the repository archive script.
