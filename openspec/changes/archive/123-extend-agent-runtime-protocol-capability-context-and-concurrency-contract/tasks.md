## 1. Contract Model and Validation

## Example Impact Assessment

修改示例

The existing agent-runtime-protocol-projection example is updated with descriptor, bounded context, action availability, and admission projection markers.

- [x] 1.1 Add additive protocol descriptor, profile version, capability declaration/request, negotiation result, and canonical host-action DTOs in the existing cross-module protocol owner.
- [x] 1.2 Reuse adapter capability negotiation's required/optional classification, `fail_fast|best_effort` strategy, profile version handling, and deterministic missing/downgrade reason taxonomy; add direct compatibility tests.
- [x] 1.3 Add bounded Session context projection types for participant/agent references, context scope, and allowlisted scalar metadata, including deterministic count, length, and serialized-size validation.
- [x] 1.4 Add host-action availability validation that distinguishes descriptor availability from authorization and rejects unsupported actions without source state mutation.
- [x] 1.5 Add concurrent-Run policy and admission outcome types with `reject|serialize|branch|optimistic|unknown`, compatible outcome validation, conflict references, stable reason codes, and side-effect-free projection helpers.

## 2. Source Mapping and Runtime Parity

- [x] 2.1 Add opt-in descriptor/context mapping for Runner, Workflow/Composer, Teams, Scheduler, A2A, and Realtime sources while preserving source ownership and nullable v1 fields.
- [x] 2.2 Map source-owned same-Session admission outcomes without implementing a second queue, lock, branch engine, cancellation policy, or persistence store.
- [x] 2.3 Add Run/Stream parity tests for capability negotiation, context validation, action availability, and admission classification under equivalent effective configuration.
- [x] 2.4 Add negative tests for unknown capabilities, incompatible profile versions, unsupported actions, invalid metadata, incompatible policy/outcome pairs, and unknown-policy defaulting.

## 3. Replay, Diagnostics, and Gates

- [x] 3.1 Extend `agent_runtime_protocol.v1` fixtures with descriptor negotiation, bounded context, action support, and concurrent-admission cases while keeping existing fixtures valid.
- [x] 3.2 Add deterministic drift fixtures for capability reason changes, profile mismatch, context-limit changes, policy/outcome mismatch, and missing correlation.
- [x] 3.3 Add additive protocol correlation fields to diagnostics/OTel projection where required, keeping `RuntimeRecorder` as the single diagnostics write entry and preserving cardinality limits.
- [x] 3.4 Extend the protocol contract gate and shell/PowerShell parity checks with capability, context, action, admission, control-plane-absence, and Run/Stream assertions.
- [x] 3.5 Update `docs/mainline-contract-test-index.md` with the new fixtures and gate-to-requirement mapping.

## 4. Example and Documentation Delivery

- [x] 4.1 **Example Impact Assessment: 修改示例** - update `examples/agent-modes/MATRIX.md` and `agent-runtime-protocol-projection/README.md` first with semantic anchor, runtime path, expected markers, unsupported-capability behavior, concurrency policy, and rollback notes.
- [x] 4.2 Update the `agent-runtime-protocol-projection` implementation and example contract tests only after the documentation baseline is complete.
- [x] 4.3 Update README, runtime configuration/diagnostics documentation, module-boundary notes, and roadmap references without introducing provider-specific context fields.
- [x] 4.4 Add example smoke coverage proving v1 lifecycle compatibility plus descriptor/context/admission projection behavior.

## 5. Verification and Delivery

- [x] 5.1 Run affected package unit and integration suites, including positive, negative, boundary, replay, and Run/Stream parity cases.
- [x] 5.2 Run `go test ./...` and `go test -race ./...`.
- [x] 5.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 5.4 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`, confirming no roadmap status drift or missing example-impact declaration.

Verification note: with `GOTOOLCHAIN=go1.26.6+auto` and the updated third-party dependencies, `govulncheck ./...` reported zero callable vulnerabilities. `pwsh -File scripts/check-quality-gate.ps1` completed all 54 steps successfully, including the example-impact declaration and roadmap-status checks; `pwsh -File scripts/check-docs-consistency.ps1` also passed.
- [x] 5.5 Review module-boundary, additive compatibility, rollback, and control-plane-absence evidence before marking the change ready for archive.
