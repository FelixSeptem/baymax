## Why

Baymax already exposes OpenAI, Anthropic, and Gemini through a shared model contract, deterministic capability fallback, tool-result feedback, and Run/Stream parity. The remaining risk is boundary drift: equivalent tool calls, reasoning content, stream aborts, usage metadata, Unicode, empty content, and overflow can still be normalized differently across providers or after a fallback/context handoff. This is now worth closing because the embedded host and runtime steering contracts make these differences observable to downstream hosts, while no single replayable conformance matrix currently blocks regressions.

## What Changes

- Define a versioned, provider-neutral conformance contract for tool-call/tool-result handoff and stream edge semantics across OpenAI, Anthropic, and Gemini.
- Add canonical normalization expectations for thinking/reasoning projections, stream start/partial/abort boundaries, usage retention, empty content, Unicode, overflow, and provider error classes.
- Require deterministic step-boundary fallback and causal correlation preservation across provider handoff without switching provider after stream emission begins.
- Add positive, negative, boundary, replay, and Run/Stream parity fixtures for the conformance matrix; keep replay offline and side-effect-free.
- Add shell/PowerShell parity gates that detect provider-specific leakage, mid-stream fallback, taxonomy drift, fixture drift, and missing evidence.
- Update provider, diagnostics, contract-index, roadmap, and contribution documentation with the new source ownership and rollback path.
- No new provider, runtime configuration key, credential store, remote catalog, hosted gateway, or provider-agnostic wire protocol is introduced.

## Capabilities

### New Capabilities

- `cross-provider-handoff-and-stream-edge-conformance`: Versioned observable contract and conformance matrix for cross-provider handoff, stream edge normalization, causal correlation, and replay/gate evidence.

### Modified Capabilities

- `llm-multi-provider-minimal`: Extend the existing tool-call, tool-result, and streaming fallback requirements with canonical edge normalization and post-start fallback-fence semantics while preserving all existing scenarios.
- `diagnostics-replay-tooling`: Extend replay requirements with the provider handoff/stream-edge fixture namespace and deterministic drift classifications.

## Example Impact Assessment

修改示例

Update the relevant agent-mode documentation baseline and add provider-independent markers only; examples MUST NOT require live provider credentials.

## Impact

- Affected implementation owners: `model/openai`, `model/anthropic`, `model/gemini`, `model/providererror`, `model/toolcontract`, and the existing runner/provider admission path.
- Affected tests and evidence: provider adapter unit tests, integration Run/Stream parity tests, diagnostics replay fixtures, and shell/PowerShell contract gates.
- Affected documentation: `README.md`, `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `docs/pi-agent-comparison-and-adoption-study.md`, and provider module READMEs as needed.
- Rollback: remove the new conformance gate and fixture namespace, revert provider normalization changes, and retain the pre-existing provider contract and replay fixtures. No persisted data or configuration migration is required.

## Baseline Audit Evidence (Tasks 1.1–1.3)

- OpenAI, Anthropic, and Gemini SDK translation remains owned by
  `model/openai`, `model/anthropic`, and `model/gemini`; each exposes the same
  `types.ModelClient` `Generate`/`Stream` boundary and maps provider stream
  events into `types.ModelEvent`. No provider SDK is imported by `core/*` or
  `context/*`.
- Canonical tool-result feedback is built once by
  `model/toolcontract.CanonicalInput`/`WithCanonicalInput`; missing call/name
  correlation is rejected as `feedback_invalid` before a provider function is
  invoked. Provider failures are classified through
  `model/providererror.Classified` (`capability_unsupported`,
  `feedback_invalid`, `auth`, `rate_limit`, `request_invalid`, `timeout`, and
  `server`/`unknown`) and retain stream phase (`pre_execution` or
  `post_start`).
- Existing integration evidence covers equivalent tool-call Run/Stream loops,
  capability fallback before invocation, and the post-start fallback fence:
  `integration/model_multi_provider_contract_test.go` verifies OpenAI,
  Anthropic, and Gemini tool calls and asserts that a provider that fails after
  emitting text cannot switch to the next provider. A known boundary to close
  is that these tests do not yet share a versioned, bounded fixture projection
  for nullable usage/reasoning, empty/Unicode content, or overflow.
- The bounded provider-neutral fixture baseline is implemented in
  `model/conformance`; it validates `provider_handoff_stream_edge.v1`, provider
  and mode ownership, causal correlation, event bounds, optional usage and
  reasoning, and deterministic canonical projection digests. Malformed and
  unknown-version cases are covered by `model/conformance/stream_edge_test.go`.
- Documentation-first evidence is anchored in
  `examples/agent-modes/realtime-interrupt-resume/README.md` and its
  `MATRIX.md` row. The semantic anchor, provider-owned runtime path, expected
  markers, rollback notes, replay namespace, and shell/PowerShell gate mapping
  are present before provider example code changes.
