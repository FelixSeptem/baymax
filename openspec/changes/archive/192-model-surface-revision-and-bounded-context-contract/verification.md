# Verification

Change: `model-surface-revision-and-bounded-context-contract`

## Focused evidence

- Added regression coverage for invalid UTF-8 resolver roles/content and invalid UTF-8 model-surface revisions. Each test was run first against the pre-fix implementation and failed with the expected acceptance defect; the implementation now rejects resolver values as `host_context.limit_exceeded` with an empty resolution and model-surface revisions as `provider_model_surface_projection_drift` before canonical projection.
- `go test ./context/hostprojection ./context/assembler ./orchestration/teams -count=1` passed.
- `go test -race ./context/hostprojection ./model/conformance ./tool/diagnosticsreplay -count=1` passed.
- `go test ./model/conformance ./tool/diagnosticsreplay -count=1` passed.
- `pwsh -File scripts/check-host-context-assembly-handoff-contract.ps1` passed.
- `pwsh -File scripts/check-provider-request-projection-contract.ps1` passed.
- Provider request fixture regeneration with `BAYMAX_REGEN_REQUEST_PROJECTION_FIXTURE=1` passed; committed fixture contains only bounded safe model-surface facts and no raw values.

## Repository gates

- `go test ./...` passed in `scripts/check-quality-gate.ps1`.
- `go test -race ./...` passed in `scripts/check-quality-gate.ps1`.
- `golangci-lint run --config .golangci.yml` passed with `0 issues`.
- `pwsh -File scripts/check-quality-gate.ps1` passed after the UTF-8 remediation; its generated repository-local cache/report artifacts were removed or restored and are not delivery files.
- `pwsh -File scripts/check-docs-consistency.ps1` passed with exit code 0.
- `pwsh -File scripts/check-openspec-documentation-impact.ps1` passed.
- `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` passed.
- `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` passed.
- `openspec validate model-surface-revision-and-bounded-context-contract --strict` passed.
- `openspec validate --all` passed: 152 items, 0 failures.
- `git diff --check` passed; remaining output was Git's LF/CRLF conversion warning only.

## Review notes

- Independent implementation review found and this change resolved two UTF-8 admission gaps before archival: resolver material now requires valid UTF-8 for both role and content, and available model-surface revisions require valid UTF-8. A follow-up read-only review confirmed both fixes, their stable classifications, zero partial resolver result, synchronized specs/docs, and reported no new Critical or Important issue. The focused RED/GREEN tests and the subsequent package/race/gate reruns above provide the regression evidence.
- Resolver admission is fixed and all-or-nothing: role input <=64 valid UTF-8 bytes, content <=64 KiB valid UTF-8 bytes, normalized aggregate <=256 KiB; invalid or overflow input returns `host_context.limit_exceeded` with no partial material.
- Model-surface evidence is explicit and opt-in. Each slot carries only availability, revision, canonical SHA-256 digest, and length; absent and explicit empty summaries are equivalent unavailable defaults.
- Canonical projection, expected/observed drift, replay, historical fixtures, privacy, and Run/Stream parity are covered without provider SDK, resolver, diagnostics writer, or effect-owner calls.
- Quality-gate output included pre-existing non-blocking replay-golden diagnostics in unrelated workflow/plan/mailbox bridge suites; the gate completed successfully and the D-focused suites and full repository tests passed.
- Generated `.artifacts/a64` reports and per-step cache directories were removed from the worktree after verification; shared machine caches were used for Go and lint runs.

## Delivery state

Main capability spec synchronization and archive are intentionally performed by
`scripts/openspec-archive-seq.ps1` in the final delivery step, after this
verification record and all implementation tasks are complete.
