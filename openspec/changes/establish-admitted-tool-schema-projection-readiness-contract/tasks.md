## 1. Contract model and normalization

- [ ] 1.1 Define bounded `tool_schema_projection_readiness.v1` input/output types, enums, limits, nullable/default fields, reason codes, and canonical digest rules; verify compile-time construction and validation tests cover valid, empty, oversized, non-admitted, privacy, duplicate, and unsupported-source inputs.
- [ ] 1.2 Implement deterministic sample-window normalization and stable-versus-transient pressure/quality classification; verify repeated-window, outlier, ordering, threshold-boundary, and insufficient-measurement tests produce stable digests and verdicts.
- [ ] 1.3 Implement advisory subset opportunity evaluation using only admitted identities and canonical facts; verify subset, required-schema retention, capability preservation, omission reasons, and admission-bypass negative tests.
- [ ] 1.4 Implement independent evidence-axis aggregation and `not_ready`/`ready_for_runtime_design`/`blocked` conclusion rules; verify missing-quality, degraded-quality, stable-ready, semantic-loss, and blocked-integrity scenarios.

## 2. Replay, fixtures, and parity

- [ ] 2.1 Add versioned success, transient, stable-ready, insufficient-evidence, blocked-integrity, privacy, overflow, and historical-default fixtures without raw schema/prompt/tool-result payloads; verify fixture size and privacy boundary tests.
- [ ] 2.2 Add diagnostics replay adapter with unknown-field tolerance, historical defaults, canonical drift classifications, and idempotent repeated replay; verify replay tests for digest/window/metrics/selection/conclusion/reason-order drift.
- [ ] 2.3 Add Run/Stream parity fixtures and comparison tests for every evidence axis and conclusion; verify equivalent inputs match and deliberate divergence returns `run_stream_parity_drift`.

## 3. Governance boundaries and gates

- [ ] 3.1 Extend `tool/contributioncheck` boundary checks to reject runtime selector/router wiring, admission bypass, remote discovery, provider SDK/tokenizer use, raw payload persistence, and RuntimeRecorder/control-plane writes; verify each violation has a stable classification.
- [ ] 3.2 Add dedicated shell and PowerShell readiness contract gates and wire them into the quality gate; verify both scripts execute the same fixture suite and fail fast on contract drift.
- [ ] 3.3 Update `docs/mainline-contract-test-index.md` with readiness owner, fixtures, replay tests, parity checks, and both gate paths; verify docs consistency checks discover the complete mapping.

## 4. Roadmap and status convergence

- [ ] 4.1 Correct `docs/development-roadmap.md` and `README.md` so archived scenario simulation is not listed as in progress, update the baseline/archive sequence, and add this readiness change as an active candidate with its trigger and non-goals; verify status-parity and roadmap-status gates pass.
- [ ] 4.2 Record Example Impact Assessment as `无需示例变更（附理由）` in proposal, design, and tasks and verify the example-impact governance gate accepts all three artifacts without modifying `examples/agent-modes`.

## 5. Verification and delivery

- [ ] 5.1 Run focused tests for the new schemaaudit/diagnosticsreplay/contributioncheck packages and verify all readiness fixtures, negative cases, and parity checks pass.
- [ ] 5.2 Run `openspec validate --all`, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; record any Windows-environment limitation without weakening contract assertions.
- [ ] 5.3 Review the diff for scope, architecture-boundary compliance, additive/default compatibility, absence of proposal-number identifiers, and clean separation from user-owned changes; verify the change is ready for review and later archive.

Example Impact Assessment: `无需示例变更（附理由）`。本任务集只交付离线 readiness contract、fixture、replay、gate 和治理文档，不修改 `examples/agent-modes` 的代码、配置或运行时语义。
