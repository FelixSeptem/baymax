## Why

Baymax already dispatches tools through validation, policy and sandbox boundaries, middleware, retries, panic recovery, and diagnostics. Those behaviors are implemented and tested in separate layers, but callers do not yet have one contract that proves which logical lifecycle stage produced a result, how a rejection differs from execution failure, or that every started call reaches deterministic cleanup and feedback on both Run and Stream paths.

Why now: the failure-taxonomy and terminal-recovery contracts have established the shared outcome and recovery boundaries. The next high-value gap identified by the roadmap is to make the existing tool path auditable end to end without inventing a parallel Tool/MCP model or terminal state machine.

## What Changes

- Add a transport-neutral tool lifecycle and failure-isolation contract covering logical `prepare`, `validate`, `authorize`, `execute`, and `finalize` outcomes for each source-owned tool call.
- Define deterministic projection and isolation rules for unknown tools, argument/schema failures, policy and allowlist denial, sandbox denial or execution failure, middleware short-circuit or failure, panic, timeout, cancellation, retry exhaustion, and returned tool errors.
- Preserve stable `call_id`, source identity, input ordering, partial valid facts, and existing first-terminal-wins semantics; parallel completion timing MUST NOT reorder normalized model feedback.
- Require every call that enters execution processing to reach one idempotent finalization projection, including panic, cancellation, timeout, and middleware failure paths, without bypassing policy, sandbox, egress, or allowlist constraints.
- Add additive, nullable, defaultable diagnostics and replay facts through `observability/event.RuntimeRecorder`; retain existing reason taxonomy and prohibit high-cardinality payloads in OTel metric dimensions.
- Add canonical and negative replay fixtures, unit/integration Run/Stream parity tests, executable agent-mode assertions, and a shell/PowerShell parity contract gate wired into the quality gate.
- **BREAKING**: none. The change reuses existing dispatcher, middleware, policy, sandbox, terminal-outcome, and ReAct ownership. Existing public fields and source-specific reason codes remain compatible.

## Capabilities

### New Capabilities

- `tool-lifecycle-and-failure-isolation`: Defines the lifecycle-stage projection, failure isolation, correlation, finalization, replay, and Run/Stream parity requirements for source-owned tool invocation.

### Modified Capabilities

None. Existing ReAct, tool middleware, security, sandbox, failure taxonomy, and terminal-recovery requirements retain their scope; this change adds a composition contract that references their established behavior rather than redefining it.

## Impact

- Affected implementation owners: `tool/local`, `core/types`, `core/runner`, `runtime/diagnostics`, `observability/event`, `tool/diagnosticsreplay`, and their focused integration tests.
- Affected contracts and gates: reuse `react-loop-and-tool-calling-parity-contract`, `agent-lifecycle-hooks-and-tool-middleware-contract`, security/sandbox contracts, terminal-outcome projection, and quality-gate conventions without creating a second execution or terminal owner.
- Affected documentation: `README.md`, `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/mainline-contract-test-index.md`, and the selected agent-mode tool example documentation.
- No external service, hosted tool runtime, transport listener, global queue, or platform control plane is introduced. Runtime module boundaries, `env > file > default`, fail-fast configuration validation, and atomic hot-reload rollback remain unchanged.

## Example Impact Assessment

修改示例

The tool-loop/agent-mode documentation baseline will be updated before implementation to show policy denial, panic or timeout finalization, and parallel `call_id` ordering assertions. The change extends an existing executable example rather than adding a new standalone tool runtime example.
