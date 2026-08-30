## Why

Baymax already exposes classified errors, ReAct termination reasons, provider streaming error categories, and protocol Run states, but these contracts were defined by different runtime domains. As Run/Stream, realtime interrupt/resume, scheduler, mailbox, recovery, and diagnostics evolve together, the same underlying condition can be reported as a Go error, a failed tool result, a terminal Run state, or an event without a stable cross-domain interpretation. This change establishes a narrow, source-owned failure taxonomy and terminal-outcome projection so callers can distinguish rejection, cancellation, retryable failure, recovery conflict, and completed work without forcing every failure into a model-visible result.

Why now: the roadmap's PDF comparison identified failure classification and authoritative terminal outcome as the highest-leverage unresolved seam. The repository is otherwise at a clean baseline with no active change, making this a suitable contract-first audit and additive extension before further event-stream or tool-lifecycle work.

## What Changes

- Define a canonical, additive failure family and terminal-outcome projection for runtime-facing Run/Stream results and mapped protocol events.
- Specify the boundary between build/admission failures, policy denials, runtime failures, timeout, cancellation, retry exhaustion, recovery conflicts, and successful completion.
- Preserve source ownership: scheduler, mailbox, workflow, provider, tool, policy, and recovery subsystems retain their execution decisions; the contract only normalizes and projects them.
- Make Run and Stream expose semantically equivalent terminal classification, causal correlation, and budget/attempt metadata for equivalent work.
- Preserve partial valid facts and source-owned diagnostics when a stream or tool fails after producing observable work.
- Add deterministic rules for retry/resume eligibility, terminal-state idempotency, and conflict recording without mutating a terminal Run back to `working`.
- Add negative and boundary replay fixtures plus integration/gate coverage for cross-domain mapping and drift.
- **BREAKING**: none intended; all new fields are additive, nullable, or defaultable. Existing error classes and source-specific reason codes remain valid during the compatibility window.

## Capabilities

### New Capabilities

None. The change extends existing runtime protocol and execution contracts rather than introducing a parallel capability namespace.

### Modified Capabilities

- `agent-runtime-protocol-contract`: clarify terminal outcome projection, failure-family mapping, idempotent terminal writes, and the distinction between source-owned failures and protocol projection.
- `multi-provider-streaming-error-taxonomy`: align stream-established versus pre-stream failures, partial-output preservation, retry boundary, and normalized terminal projection.
- `react-loop-and-tool-calling-parity-contract`: align ReAct/tool termination mapping with the shared failure family while preserving existing canonical ReAct reasons and Run/Stream parity.
- `multi-agent-session-recovery`: clarify recovery-conflict and retry/resume projection without changing source scheduler or recovery ownership.

## Impact

- Affected code: `core/types`, `core/runner`, `model/*`, `tool/local`, `orchestration/*`, `runtime/diagnostics`, `observability/event`, and protocol mapping/replay helpers.
- Affected public projections: additive fields on `RunResult`, protocol Event/Run mappings, diagnostics query records, and terminal reason metadata. Existing fields remain readable and existing source error classes are not removed.
- Affected tests: provider streaming tests, ReAct Run/Stream parity, tool dispatch failure cases, recovery conflict/retry cases, protocol replay fixtures, diagnostics idempotency, and shared quality gates.
- Affected documentation: `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, and relevant example READMEs.
- Dependencies: reuse existing `ClassifiedError`, protocol Run states, reason taxonomy, `RuntimeRecorder`, policy precedence, snapshot/recovery, and replay infrastructure. No new external service or hosted control plane.

## Example Impact Assessment

修改示例

Existing Run/Stream, realtime, tool-loop, and recovery examples need additive terminal classification assertions; no new standalone example is required unless an existing fixture cannot express a boundary case.
