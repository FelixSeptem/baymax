# Native event history implementation verification

## Scope and environment

- Change: `retain-source-native-event-history-and-cursor-replay` (Spec 2 / BXM-2).
- Branch: `retain-source-native-event-history-and-cursor-replay`.
- Base revision: `7675273a5ca85dac7585bae85cd842ca0b7b3e1f`.
- Verification date: 2026-10-19; Windows/amd64, Go 1.26.9, PowerShell 7.
- Go cache: workspace-local `.gocache-native-history`; final lint cache:
  `.gocache-native-history/lint`.
- The change was archived as 168 after all verification evidence was recorded.
  Commit, merge and push are tracked by the feature-branch lifecycle rather
  than being treated as history replay semantics.

## Verified behavior

The source-neutral history API provides bounded source references, opaque cursors,
detached pages, explicit source capabilities and typed unsupported, malformed,
foreign, expired, gap and unavailable errors. Callback EventID/TraceID remain
observation references and cannot substitute for journal identity or cursor.

The existing mailbox owner appends reference-only lifecycle facts to its own
snapshot. FileStore reopens the same journal identity and sequence in another
process; confirmed-cursor replay recovers retained terminal evidence without
admitting a message or invoking an execution path. MemoryStore advertises only
process-local retention. Pages expose source-local sequence through `History`,
without assigning a global protocol sequence or order across sources.

Retention is the last 1024 lifecycle entries, with no time guarantee. Expired and
gapped reads fail with safe floor evidence and no substituted latest page.
Pending batched writes, failed persistence and a legacy file snapshot without a
committed journal return `ErrSourceUnavailable`; reads never flush the store.

Runner, isolated child, team and workflow history are explicitly unsupported.
Their existing real-time callback and execution behavior remains covered. The
capability manifest promotes history/replay only from concrete mailbox source
facts, and does not claim checkpoint restore or execution recovery.

## Commands and results

| Command | Result |
| --- | --- |
| `go test ./core/types ./orchestration/mailbox ./orchestration/planbridge ./host/eventadapter ./tool/diagnosticsreplay ./integration -run '^TestNativeHistory\|^TestMailboxHistory\|^TestFileMailboxHistory\|^TestCallbackAdapterNeverOwnsHistory\|^TestRetainedFactCanReachSink' -count=1` | exit 0, through both native-history gates |
| `go test ./tool/diagnosticsreplay ./integration -run '^TestNativeHistoryReplay\|^TestNativeHistoryDiagnosticsReplay' -count=1` | exit 0 after final fixture mutation cases |
| `go test ./... -timeout 15m` | exit 0, including final post-review rerun |
| `go test -race ./... -timeout 20m` | exit 0, including final post-review rerun |
| `golangci-lint run --config .golangci.yml` | exit 0, 0 issues with workspace lint cache, including final post-review rerun |
| `go run ./examples/native-execution-events` | exit 0; offline history/restart/unsupported markers |
| `pwsh -NoProfile -File scripts/check-native-event-history-contract.ps1` | exit 0, including final post-review rerun |
| `bash scripts/check-native-event-history-contract.sh` (Git Bash) | exit 0 after sandbox escalation before final review fixes; final rerun approval was cancelled before execution |
| `pwsh -File scripts/check-openspec-documentation-impact.ps1` | exit 0 |
| `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` | exit 0 |
| `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` | exit 0 |
| `pwsh -File scripts/check-docs-consistency.ps1` | exit 0 |
| `openspec validate retain-source-native-event-history-and-cursor-replay --strict` | exit 0 |
| `git diff --check` | exit 0 |
| `pwsh -NoProfile -File scripts/check-quality-gate.ps1` | exit 0; 81 stages, 1497.6 seconds; final review fixes separately covered by final full tests/race/lint and PowerShell contract gate |

The dual-platform contract gates exercise source audit, cursor identity/order,
restart, terminal recovery, retention, no-side-effect behavior and pure offline
fixture replay. Mutation cases reject callback identity substitution, invented
terminal outcomes, terminal/type disagreement, unsafe latest fallback, cursor or
sequence drift, error pages containing substitute events and payload leakage.

The final post-review regression command also passed (exit 0):

```powershell
go test ./core/types ./orchestration/mailbox ./tool/diagnosticsreplay ./integration -run 'TestNativeExecutionTerminalUsesMailboxHistoryOutcome|TestMailboxHistoryExpiryPreservesTransitionTerminal|TestNativeHistoryReplayAcceptsExpiryTransitionEndingInDLQ|TestNativeHistoryReplayRejectsSemanticDrift|TestNativeHistoryDiagnosticsReplay' -count=1
```

The comprehensive quality gate included tests, race, lint, contract gates,
performance checks, full-chain smoke and strict vulnerability analysis. The
vulnerability stage reported zero called vulnerabilities; it also reported two
imported-package and one required-module vulnerabilities without called paths.

## Failures resolved and environment observations

- Initial full test/race runs exposed release-status parity drift: the active
  roadmap bullet contained multiple backtick spans and README lacked the active
  milestone. The roadmap bullet and README were corrected; both full reruns
  passed. No parser behavior was changed.
- An intermediate diagnostics replay compile error was fixed before final full
  test/race runs and fixture verification.
- After checking the final tasks, OpenSpec changed the status from in-progress
  to complete. The roadmap consistency gate correctly rejected the stale
  in-progress entry. README and roadmap were first updated to describe the
  complete pending-archive state, then synchronized to archive 168 after the
  repository archive script completed; no archive entry or parser behavior was
  invented.
- Initial lint used the user cache and printed cache-write warnings despite
  reporting 0 issues; the final run used a writable workspace lint cache and
  passed without those warnings.
- Git Bash could not create its signal pipe inside the Windows sandbox (Win32
  error 5). The identical shell gate passed with approved elevated execution;
  the gate was not weakened. The later post-review elevated rerun was cancelled
  at the approval step and did not execute. The equivalent PowerShell gate and
  full tests/race/lint passed after the final code changes.
- A reviewer's targeted test attempt used the default Go cache and failed to
  start with Access Denied. The primary agent's post-review targeted and full
  runs used the workspace cache and passed.
- PowerShell profile and Git LF/CRLF notices are environmental output, not
  contract failures. All table entries reflect process exit codes.

## Risks, limitations and rollback

- Terminal evidence describes retained mailbox lifecycle (`acked`, `dead_letter`,
  `expired`), not model completion or host business commit. A journal-wide
  `Complete` page only reaches the current retained tail.
- Retention can expire a slow consumer's cursor. Hosts must handle typed
  expiry/gap and must not silently read latest or re-run a parent.
- Durability follows the existing FileStore snapshot commit boundary. This
  change does not add multi-writer file coordination, checkpoint restore or a
  new owner/scheduler.
- A legacy file becomes history-capable only after an explicit normal owner
  persistence action commits a new journal. Reads cannot manufacture historical
  facts for pre-journal transitions.
- Rollback stops exposing the new reader/manifest history support and preserves
  mailbox records and journal snapshots. It does not rewrite cursors, replay
  execution, or claim to undo business effects.
- Existing unrelated `.artifacts` changes and cache/output directories were
  preserved and are not implementation deliverables.

## Review and final status

Independent review found two terminal-evidence issues, both corrected and
verified. `NativeExecutionEvent.IsTerminal` now recognizes the explicit mailbox
history terminal outcomes `acked`, `dead_letter` and `expired`; unknown outcomes,
other source kinds and ordinary consume facts remain nonterminal. `SourceState`
preserves the source-owned terminal outcome. The offline evaluator now accepts
`mailbox.expired` with either `expired` or `dead_letter`, because TTL expiry can
finish in DLQ. Owner state is preserved rather than rewritten to match a
transition name. Regression tests cover both corrections and documentation
explains the transition/final-state distinction.

Read-only follow-up review confirmed both findings were resolved without
residual issues in those changes. The comprehensive quality gate completed
successfully; final full tests/race/lint, the PowerShell contract gate and
documentation/OpenSpec checks also passed after the review fixes. All 24 tasks
are complete. The repository archive script synchronized five delta specs and
archived the change as 168; feature-branch integration follows the repository
lifecycle after this recorded archive step.

## Archive integration verification

- `pwsh -NoProfile -File scripts/openspec-archive-seq.ps1 -ChangeName retain-source-native-event-history-and-cursor-replay`: exit 0. Five main specs were synchronized (nine added requirements) and archive 168 was created; `openspec list --json` reports no active changes.
- Post-archive `pwsh -NoProfile -File scripts/check-docs-consistency.ps1`: exit 0, including roadmap parity. README and roadmap links target archive 168.
- An additional `openspec validate --all --strict` run reported 30 passes and 112 failures across 142 main specs. The failures were historical Purpose placeholders. Two of those specs are touched by this change (`diagnostics-replay-tooling` and `multi-agent-mailbox-contract`); their Purpose text was replaced with actual capability descriptions. Untouched historical specs were left outside this change.
- All five synchronized main specs then passed individual strict validation (exit 0): `native-execution-event-history`, `host-runtime-capability-manifest`, `native-execution-event-adapter`, `multi-agent-mailbox-contract`, and `diagnostics-replay-tooling`.
- The post-archive `pwsh -NoProfile -File scripts/check-quality-gate.ps1` rerun passed (exit 0): 81 stages, 1258.85 seconds, including full tests, race, lint, history contract, performance regression, full-chain smoke and strict vulnerability scanning. The scan reported zero called vulnerabilities; two imported-package and one required-module vulnerabilities had no called paths.
