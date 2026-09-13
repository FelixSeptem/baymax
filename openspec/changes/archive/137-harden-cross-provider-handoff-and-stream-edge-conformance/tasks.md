## Example Impact Assessment

修改示例。Update the existing agent-mode documentation baseline before changing example code; examples use provider-independent semantic markers and never require live credentials.

## 1. Baseline and Conformance Matrix

- [x] 1.1 Audit OpenAI, Anthropic, and Gemini adapter boundaries, existing toolcontract/error taxonomy, and Run/Stream provider tests; record the current normalized behavior and any reproducible drift cases in the proposal evidence.
- [x] 1.2 Define the bounded `provider_handoff_stream_edge.v1` fixture schema, canonical normalized projection, correlation fields, nullable usage/reasoning fields, and stable drift vocabulary; verify malformed and unknown-version validation cases.
- [x] 1.3 Add documentation-first updates to `examples/agent-modes/MATRIX.md` and the selected provider-independent mode README with semantic anchor, runtime path, expected markers, rollback notes, and replay/gate mapping; verify the doc-first gate before example tasks proceed.

## 2. Provider Handoff Normalization

- [x] 2.1 Add or refine adapter tests for equivalent OpenAI/Anthropic/Gemini tool-call normalization, preserving bounded arguments, tool-call identity, step/run correlation, and optional thinking projection.
- [x] 2.2 Add canonical tool-result feedback tests for success, provider-native error, missing correlation, malformed shape, and oversized payload; verify rejection occurs before provider invocation.
- [x] 2.3 Add provider error taxonomy tests for capability unsupported, feedback invalid, authentication/rate-limit, overflow, and transport/abort outcomes; verify no provider-only class leaks into Runner contracts.

## 3. Stream Edge and Fallback Fence

- [x] 3.1 Add deterministic stream fixtures/tests for start, partial, completion, empty content, U+2028/U+2029 Unicode, usage present/absent, and bounded overflow across all supported adapters.
- [x] 3.2 Add abort and post-start failure tests proving the current step terminates canonically and never switches provider after the first semantic stream event.
- [x] 3.3 Add pre-step capability/request-shape fallback tests proving deterministic candidate selection, causal correlation preservation, and fail-fast exhaustion without partial provider calls.
- [x] 3.4 Add Run/Stream parity coverage for handoff, empty/Unicode content, usage/abort, overflow, fallback fence, and terminal outcome; verify only permitted event-order differences remain.

## 4. Replay and Contract Gates

- [x] 4.1 Implement `provider_handoff_stream_edge.v1` replay parsing, bounded normalization, side-effect-free execution, historical fixture compatibility, and idempotency tests.
- [x] 4.2 Add replay drift classification tests for schema, tool-call correlation, tool-result feedback, thinking projection, stream boundary, abort/usage, overflow, Unicode/empty content, fallback fence, and Run/Stream parity.
- [x] 4.3 Add shell and PowerShell conformance gates with equivalent checks for adapter ownership, no raw payload diagnostics, no mid-stream fallback, bounded fixtures, taxonomy stability, parity, and replay idempotency.
- [x] 4.4 Register the gate in `scripts/check-quality-gate.*`, update `docs/mainline-contract-test-index.md`, and verify missing fixture/gate evidence blocks the quality gate.

## 5. Diagnostics and Documentation

- [x] 5.1 Verify `RuntimeRecorder` and OTel projections remain bounded/additive and never store raw provider payloads, reasoning bodies, credentials, or unbounded stream content; add positive and negative diagnostics tests.
- [x] 5.2 Update `README.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/development-roadmap.md`, and `docs/pi-agent-comparison-and-adoption-study.md` with provider owner boundaries, no-new-config decision, rollback path, and deferred routing exclusions.
- [x] 5.3 Update provider and replay READMEs or cross-reference indexes as needed; verify docs consistency and roadmap status gates pass.

## 6. Integrated Verification and Handoff

- [x] 6.1 Run focused provider, toolcontract, runner parity, host-independent example, and diagnostics replay suites with positive, negative, boundary, and idempotency evidence.
- [x] 6.2 Run `openspec validate --all`, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and both provider conformance gate scripts; record exact results and environment-blocked commands. `go test ./...` passes on rerun with dedicated GOCACHE; quality gate reached all provider/conformance checks, with its parallel host-jsonl subprocess cleanup encountering a Windows ACL race on a temporary executable. `go test -race` remains environment-blocked by `cgo.exe`/`cc1.exe`.
- [x] 6.3 Perform final scope review against non-goals and architecture boundaries; verify no new provider, credential store, remote catalog, hosted gateway, context/provider SDK dependency, raw-payload diagnostic field, or mid-stream fallback path was introduced.
