## Example Impact Assessment

无需示例变更（附理由）：任务只交付离线审计合同、fixture、replay、gate 和文档，不改变 `examples/agent-modes`；若实现阶段发现需要修改示例，必须先完成 `MATRIX.md` 与对应 README 的文档基线并重新评估。

## 1. Baseline and contract setup

- [x] 1.1 Audit existing Action Gate, tool lifecycle, security, timeline, RuntimeRecorder, diagnostics replay, and Run/Stream parity owners; record the exact source-owned fields and verify the audit has no second state owner with `openspec list --specs`, relevant `openspec show` outputs, and a checked-in owner mapping note.
- [x] 1.2 Define the versioned `action_capability_audit.v1` snapshot, bounded field limits, nullable/default compatibility rules, privacy exclusions, and stable verdict/drift taxonomy; verify malformed, unsupported-version, missing-field, and overflow cases are specified before implementation.
- [x] 1.3 Add the new capability contract under `specs/action-capability-risk-idempotency-evidence-audit/spec.md` to the mainline contract index mapping; verify the index points to the change spec and no existing capability path is duplicated.

## 2. Canonical normalization and evidence audit

- [x] 2.1 Implement snapshot parsing and deterministic normalization for stable action identity, source, namespace/tool, version/digest, effect, side-effect, risk, reversibility, idempotency, preconditions, timeout/retry, owner, and scope; verify equivalent input ordering yields one canonical digest.
- [x] 2.2 Implement bounded evidence-reference normalization for intent, issued, confirmed, Preview, Approve, Commit, Verify, Policy, Sandbox, Tool lifecycle, and Action timeline facts; verify raw payloads, credentials, reasoning, full command output, and unbounded responses are rejected.
- [x] 2.3 Implement evidence-sufficiency and declared-versus-observed correlation checks; verify missing idempotency/risk/verify facts produce `gap` or `insufficient_evidence`, issued-but-unconfirmed operations never become compliant, and conflicting duplicate evidence fails deterministically.
- [x] 2.4 Implement the four verdicts `compliant`, `gap`, `insufficient_evidence`, and `not_applicable` plus canonical drift codes; verify each action receives exactly one verdict and no missing evidence is treated as safe, reversible, retryable, or successful.

## 3. Replay and fixture coverage

- [x] 3.1 Add canonical fixtures covering complete low-risk read, high-risk Preview/Approve/Commit/Verify, missing metadata, non-idempotent retry ambiguity, issued-without-confirmation, approval scope drift, declared/observed drift, duplicate conflict, and privacy/boundary violations; verify each fixture has deterministic expected output.
- [x] 3.2 Add Run/Stream paired fixtures for allow, deny, timeout, pre-execution rejection, started interruption, confirmed result, and unconfirmed result; verify semantic parity is accepted while confirmation or verdict divergence returns `action_capability_run_stream_evidence_parity_drift`.
- [x] 3.3 Add historical compatibility fixtures that omit the new extension fields; verify archived lifecycle, action-gate, security, timeline, and replay fixtures remain parseable with documented defaults and no false audit failure.
- [x] 3.4 Verify replay is offline, read-only, idempotent, and bounded by running repeated replays and negative side-effect probes; confirm no provider/tool/network/registry/credential/RuntimeRecorder/source-store call occurs.

## 4. Gate, diagnostics boundary, and documentation

- [x] 4.1 Integrate the audit replay into `tool/contributioncheck` with shell and PowerShell parity; verify taxonomy drift, fixture drift, library-first boundary violations, and output mismatch fail with stable classifications.
- [x] 4.2 Keep audit output outside RuntimeRecorder, RunRecord, runtime config, Policy, Tool registry, and business Outcome; verify diagnostics cardinality and privacy tests show no raw payload or high-cardinality evidence dimension is emitted.
- [x] 4.3 Update `docs/mainline-contract-test-index.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, and `docs/development-roadmap.md`; verify each document records owner boundaries, no new config, rollback, and `Example Impact Assessment` without marking the candidate as runtime-complete.
- [x] 4.4 Verify `examples/agent-modes/MATRIX.md` and affected mode READMEs are unchanged; if any example change is discovered, stop and complete the required docs-first baseline before checking this task.

## 5. Validation and handoff

- [x] 5.1 Run focused tests for the audit/replay package, diagnostics replay, contributioncheck, and affected Action Gate/tool lifecycle packages; verify positive, negative, boundary, privacy, and Run/Stream parity cases pass.
- [x] 5.2 Run `openspec validate --all`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and the corresponding shell gates; verify no `roadmap-status-drift`, `missing-example-impact-declaration`, `invalid-example-impact-value`, or taxonomy drift is reported. The default 900-second full-gate budget first expired before the final race/lint steps; the gate's documented `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=1800` setting was then used without skipping any check, and all 72 steps passed in 1168.64 seconds.
- [x] 5.3 Run repository completion gates `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml`; record output, risk points, rollback point, and any intentionally unexecuted command in the handoff. All three commands passed (lint: `0 issues`). The prior lint blocker was seven existing Go files checked out as CRLF under `core.autocrlf=true`; the repository now enforces LF for `*.go` through `.gitattributes`, and the seven files were mechanically gofmt-normalized. Risk remains limited to the offline audit's fixture/normalizer/gate surface; rollback removes the new audit capability and this EOL rule while retaining existing runtime owners. No required completion command remains unexecuted.
- [x] 5.4 Re-check `openspec list --json`, roadmap current-state text, and `git diff --check`; verify the change remains the only active change, all tasks have evidence, and no runtime behavior or example semantics changed. Existing unrelated workspace edits, caches, and temporary artifacts remain untouched.
