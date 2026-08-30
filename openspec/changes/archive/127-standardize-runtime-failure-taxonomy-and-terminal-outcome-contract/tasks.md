## 1. Contract Model And Validation

- [x] 1.1 Define the additive normalized failure-family and terminal-outcome types in `core/types`, including execution phase, terminal state, source reason, retryable/resumable flags, causal correlation, and bounded attempt metadata.
- [x] 1.2 Implement validation for family/state/phase compatibility, required correlation, terminal-state immutability, and source-owned retry/resume declarations.
- [x] 1.3 Implement first-terminal-wins publication with idempotent duplicate handling and deterministic late-conflict recording.
- [x] 1.4 Add compatibility defaults so legacy protocol and runtime results remain valid when normalized terminal fields are absent.

## 2. Runner And Provider Projection

- [x] 2.1 Map existing ReAct completion, budget, provider, context-cancel, and tool-dispatch reasons into the normalized terminal projection without changing canonical ReAct reason codes.
- [x] 2.2 Align Run and Stream projection through one shared normalization path and verify equivalent terminal family, phase, state, reason, and attempt metadata.
- [x] 2.3 Update provider adapters to distinguish pre-stream construction/connection failures from post-start stream failures while preserving existing provider reason categories.
- [x] 2.4 Preserve valid partial provider output and completed tool-call facts on post-start failure, and prevent projection-layer retries of consumed work.

## 3. Tool And Policy Boundaries

- [x] 3.1 Map tool validation, policy denial, sandbox failure, execution failure, timeout, cancellation, and retry exhaustion to normalized family/phase fields while retaining existing `ClassifiedError` details.
- [x] 3.2 Verify middleware, sandbox, allowlist, and action-gate paths cannot bypass source-owned authorization or synthesize retry/resume decisions.
- [x] 3.3 Verify parallel tool outcomes preserve call-id association and deterministic input ordering in normalized terminal and diagnostic projections.
- [x] 3.4 Add panic/cleanup coverage to ensure post-start tool failures finalize resources and retain prior valid outcomes.

## 4. Orchestration And Recovery Projection

- [x] 4.1 Map scheduler, mailbox, workflow, and composer admission/recovery outcomes into normalized pre-execution or post-start phases without changing source lifecycle ownership.
- [x] 4.2 Classify snapshot/reconciliation conflicts as `recovery_conflict`, preserve the first terminal outcome, and record deterministic late conflicts.
- [x] 4.3 Verify retry/resume creates or references a new causal attempt according to existing recovery contracts and never mutates a terminal Run back to `working`.
- [x] 4.4 Verify memory/file backend parity and Run/Stream parity for restored terminal outcomes and conflict handling.

## 5. Observability And Replay

- [x] 5.1 Add additive terminal projection fields to diagnostics and OTel mappings through `observability/event.RuntimeRecorder`, with cardinality and payload bounds.
- [x] 5.2 Add replay fixtures for success, policy denial, pre-start failure, post-start provider failure, tool failure, timeout, cancellation, retry exhaustion, recovery conflict, duplicate terminal publication, and late conflict.
- [x] 5.3 Add drift assertions for terminal-state overwrite, synthesized retry/resume, lost partial facts, missing correlation, and Run/Stream semantic divergence.
- [x] 5.4 Update `docs/runtime-config-diagnostics.md` and `docs/mainline-contract-test-index.md` with additive fields, taxonomy mapping, fixture names, and gate ownership.

## 6. Examples And Gates

- [x] 6.1 Update existing Run/Stream, realtime, tool-loop, and recovery examples with additive terminal family/state assertions; keep the example impact as `修改示例`.
- [x] 6.2 Add or extend the dedicated contract gate with shell/PowerShell parity and a source-ownership/control-plane-absent assertion.
- [x] 6.3 Run focused package tests, integration/replay suites, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `scripts/check-docs-consistency.ps1`, and `scripts/check-quality-gate.ps1`.
- [x] 6.4 Record verification evidence, residual risks, rollback procedure, and archive readiness in the final change review.

## Example Impact Assessment

修改示例

Existing Run/Stream, realtime, tool-loop, and recovery examples require additive terminal classification assertions; no new standalone example is required unless an existing fixture cannot express a boundary case.

## Final Change Review

- Verification evidence: focused affected-package tests, `go test ./... -count=1`, `go test -race ./... -count=1`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-docs-consistency.ps1`, `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`, `pwsh -File scripts/check-agent-mode-migration-playbook-consistency.ps1`, `pwsh -File scripts/check-go-file-line-budget.ps1`, and `pwsh -File scripts/check-quality-gate.ps1` were executed. The final quality-gate harnessability report recorded `pass=true` and `failed_checks=[]`; generated `.artifacts/a64/*` reports were restored because they are gate side effects.
- Residual risks: terminal payload session correlation remains empty in the nested `run.finished.terminal_outcome` object when reconstructed only from a legacy `RunResult`; flat terminal fields and `RunResult.TerminalOutcome` retain correlation. OTel intentionally omits causation IDs to keep cardinality bounded.
- Rollback: disable/ignore additive terminal fields and remove the dedicated gate registration; existing source-specific errors, lifecycle states, and legacy replay fixtures remain valid.
- Archive readiness: all 24 tasks are complete, OpenSpec artifacts validate, docs/example impact declarations are present, and no new external dependency or control-plane service was introduced.
