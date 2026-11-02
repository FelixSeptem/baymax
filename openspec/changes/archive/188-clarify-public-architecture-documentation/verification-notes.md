# Documentation Baseline

Captured from the clean `origin/master` worktree on 2026-10-22.

## OpenSpec State

`openspec list --json` returned no active changes. The change under implementation is the only new active change and is intentionally documentation-only with `skip_specs: true`.

## Architecture Files

- `docs/architecture/overview.md`
- `docs/architecture/design-principles.md`
- `docs/architecture/runtime-lifecycle.md`
- `docs/architecture/data-event-flow.md`
- `docs/architecture/module-ownership.md`

## Component Files

- `docs/components/README.md`
- `docs/components/runner.md`
- `docs/components/context.md`
- `docs/components/providers.md`
- `docs/components/tools.md`
- `docs/components/mcp.md`
- `docs/components/orchestration.md`
- `docs/components/runtime.md`
- `docs/components/observability.md`
- `docs/components/adapters.md`
- `docs/components/extensions.md`
- `docs/components/skills.md`

## Baseline Status Matches Requiring Review

- `orchestration/README.md`: application outbox is labelled `进行中 change` although the archive contains its completed change.
- `docs/runtime-config-diagnostics.md`: host context projection and application outbox retain `进行中` wording after their changes were archived.
- `docs/guides/integration.md`: host context assembly/handoff retains `进行中提案` wording after its direct contract was archived.
- `core/runner/README.md`: durable continuation is labelled `提案中` although direct continuation is already part of the current baseline.
- `core/types/README.md`: checkpoint compatibility is labelled `提案中` although the current direct continuation contract is already documented and tested.
- `docs/development-roadmap.md`: one historical paragraph still names `retain-run-and-composite-source-history` as ongoing even though the active-change list is empty; this must be changed to a historical/archived reference after checking the archive index.

The root README also contains historical archive bullets. Those are retained only when they provide a useful public milestone; they are not the source of active status.

## Implementation Verification

Completed on 2026-10-22 in the `clarify-public-architecture-documentation` worktree:

- `openspec validate --all` passed.
- `pwsh -File scripts/check-openspec-documentation-impact.ps1` passed.
- `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` passed.
- `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` passed.
- Focused `tool/contributioncheck` navigation/status tests passed.
- `go test -race ./tool/contributioncheck -run 'TestDocumentationNavigation|TestValidateStaleDocumentationStatus|TestCurrentDocumentationHasNoStaleStatusWording' -count=1 -p 1` passed using isolated `GOCACHE` and `GOTMPDIR` directories.
- `golangci-lint run --config .golangci.yml` passed using an isolated cache.
- `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` passed.
- The first `pwsh -NoProfile -File scripts/check-quality-gate.ps1` run (session `38587`) failed only because all tasks had been marked complete before archival, making `openspec list --json` report `complete` while the roadmap still listed the change as in progress. After restoring 5.3 as pending, the affected roadmap check and the complete React plan notebook gate passed independently.
- The second full `pwsh -NoProfile -File scripts/check-quality-gate.ps1` run (session `71793`) passed all preceding contract, docs, Go test, and race stages but exceeded the 1800-second aggregate budget during the final golangci-lint step, which had only four seconds remaining. The gate therefore exited non-zero for a budget timeout, not a lint finding.
- `go test ./...` passed for the repository run after the Windows test-binary lock was isolated; `mcp/stdio` was also rerun successfully with an isolated cache/temp directory.
- The second full `go test -race ./...` run completed successfully with no race report. An earlier race attempt encountered Windows `.test.exe` access-denied locks in several packages; the affected packages and the adapter determinism test were rerun individually where needed. The new documentation-navigation race suite passed as recorded above.
- Independent final `golangci-lint run --config .golangci.yml` reported `0 issues`; the process emitted Windows golangci cache `Access is denied` warnings while persisting facts, so the result is recorded as clean findings with an environment cache warning.
- `bash scripts/check-docs-consistency.sh` could not execute in this environment because the shell/service layer returned `E_ACCESSDENIED` and garbled service output; the PowerShell equivalent passed.
- `git diff --check` passed. The only remaining status-language matches are intentional explanatory prose in `README.md` and `docs/documentation-ownership.md`; no stale active-change claim remains.
- The final execution-model addition was verified on 2026-10-22: `openspec validate --all` passed (145/145), the documentation-impact, example-impact, and roadmap-status gates passed, and `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` passed.
- The focused navigation/source-map/stale-status suite passed with an isolated cache: `go test ./tool/contributioncheck -run 'TestDocumentationNavigation|TestValidateDocumentationSourceMap|TestCurrentDocumentationHasNoStaleStatusWording' -count=1 -p 1`.
- The same focused suite also passed under the race detector with an isolated cache: `go test -race ./tool/contributioncheck -run 'TestDocumentationNavigation|TestValidateDocumentationSourceMap|TestCurrentDocumentationHasNoStaleStatusWording' -count=1 -p 1`.
- The canonical `docs/architecture/execution-model.md` link was checked in the root README, docs index, architecture overview, ownership map, and navigation fixture; its source-map verification target exists.
- Final diff review found no stale active-change wording, no broken links in the new execution-model page, and no duplicated field-level contract. Untracked validation caches remain excluded. The two modified `.artifacts/a64` reports are gate-generated snapshots from the earlier quality-gate run and are called out separately from the functional documentation diff.
- The extended full quality gate was rerun with `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=3600` on 2026-10-22. All 86 quality-gate steps through Go tests, race tests, contract/replay suites, benchmarks, and `golangci-lint` passed. The command exited with code 1 only at strict `govulncheck ./...` because Go standard library `crypto/x509@go1.26.9` is affected by `GO-2026-6809`, fixed in `go1.26.10`; the scan also reported non-called dependency findings. This is an environment/toolchain security baseline issue, not a documentation regression.
- `pwsh -NoProfile -File scripts/check-docs-consistency.ps1` passed in the same run. `bash scripts/check-docs-consistency.sh` was not executable in this environment because the shell/service layer returned `E_ACCESSDENIED`; the PowerShell equivalent passed.
- Isolated `.gocache-*`, `.gotmp-*`, `.golangci-*`, and OpenSpec log files are untracked validation artifacts and are intentionally excluded from the change; they are not implementation outputs.
- The quality gate also refreshed the tracked `.artifacts/a64/harnessability-scorecard.json` and `.artifacts/a64/impacted-full-report.json` reports. Their changes are gate-generated timestamp/fixture/selection snapshots, not documentation implementation; they remain visible for the repository's existing artifact governance and are called out separately from the functional diff.
- The final orchestration README audit corrected the stale planbridge summary: source-owned child continuation and manifest-approved `mailbox_async` are supported where the native capability/binding permits them; mailbox continuation, resumed-plan continuation, composite durable continuation, subgraphs, shared queue children, and unapproved composite pause/restore remain fail-closed. The mailbox owner still owns message/delivery/lease/attempt/retry/DLQ/terminal lifecycle, and planbridge does not create a second owner.
- The archive link audit confirmed the planbridge references point to the real archived `162-establish-host-execution-plan-bridge-contract` change rather than a deleted active-change path.
- The public documentation index now separates canonical current references from historical/planning material, and the full public-Markdown relative-link validator covers package, example, guide, archive, and canonical paths while skipping only external URLs, fragment-only links, and `docs/superpowers` drafts.
- Focused navigation and stale-status tests passed after the orchestration wording correction, as did `pwsh -NoProfile -File scripts/check-docs-consistency.ps1`, `openspec validate --all`, and the OpenSpec documentation-impact, example-impact, and roadmap-status checks. The extended quality gate remains non-zero only at strict `govulncheck ./...` for the Go toolchain finding `GO-2026-6809` in `crypto/x509@go1.26.9` (fixed in `go1.26.10`); the Bash docs check remains unavailable because the environment returns `E_ACCESSDENIED`.
- The external-developer audit found and corrected four stale public status claims: mailbox outbox dispatch now points to archived 166, host resume admission to archived 164, context assembly/handoff to archived 165, and scenario simulation to archived 150. The host-resume guide now distinguishes the legacy metadata-only `RunCheckpointStore` SPI from the delivered `native_continuation.v1` owner path.
- Added `docs/guides/getting-started.md` as the canonical first-use path: Go/toolchain prerequisites, offline fake-model smoke, real OpenAI quickstart, Run/Stream expectations, host ownership checklist, verification and rollback. Added `examples/quickstart/README.md` without changing example runtime code.
- Extended the documentation navigation fixture to require the getting-started path, scan public package/example README relative links, and scan current canonical host-resume/mailbox/runtime pages for stale active/in-progress wording. The expanded focused suite passed with an isolated cache: `go test ./tool/contributioncheck -run 'TestDocumentationNavigationBaseline|TestValidateDocumentationLinksDetectsMissingTarget|TestValidateDocumentationNavigation|TestValidateDocumentationSourceMap|TestCurrentDocumentationHasNoStaleStatusWording' -count=1 -p 1`.
- The post-audit serial repository test passed: `go test ./... -count=1 -timeout 15m -p 1`; `go vet ./...` also passed. The first parallel ordinary/race runs encountered only Windows `.test.exe` access-denied locks; the affected ordinary `context/guard` package and all seven affected race packages (`a2a`, `adapter/profile`, `integration/extensionlifecycle`, `observability/trace`, `orchestration/invoke`, `orchestration/workflow`, `tool/schemaaudit`) passed on isolated serial reruns.
- The expanded focused suite passed under `go test -race`; independent `golangci-lint run --config .golangci.yml` with isolated Go/lint caches reported `0 issues`.
- The final owner-navigation audit added public entries for `docs/api-reference-d1.md`, `memory/README.md`, `host/eventadapter/README.md`, `context/handoff/README.md`, `context/hostprojection/README.md`, and `integration/README.md` in the README/index/source map. The focused test `TestPublicIndexesExposeAPIAndOwnerDocumentation` first failed for the five missing owner paths, then passed after the navigation update.
- Archived host-context status drift was corrected: `context/handoff/README.md` now identifies archived change 165 as delivered, `context/hostprojection/README.md` links its executed verification record, and `context/README.md` no longer calls the bridge active. `TestValidateStaleDocumentationStatusDetectsArchivedHostContextWording` first failed to classify the phrases, then passed after adding exact historical markers without rejecting ordinary runtime `pending` wording.
- The expanded `go test ./tool/contributioncheck -count=1 -p 1` passed in 20.055 seconds. `pwsh -NoProfile -File scripts/check-docs-consistency.ps1`, `openspec validate --all` (145 passed, 0 failed), `scripts/check-openspec-documentation-impact.ps1`, `scripts/check-openspec-example-impact-declaration.ps1`, `scripts/check-openspec-roadmap-status-consistency.ps1`, and `git diff --check` all passed after the owner-navigation correction.
- The corresponding focused race suite `go test -race ./tool/contributioncheck -run 'TestDocumentationNavigation|TestPublicIndexesExposeAPIAndOwnerDocumentation|TestValidateStaleDocumentationStatusDetectsArchivedHostContextWording|TestCurrentDocumentationHasNoStaleStatusWording' -count=1 -p 1` passed with isolated cache/temp directories.
