## Context

The existing provider adapters already share canonical model, tool-call, tool-result, capability, error, and streaming contracts. Provider-specific request/response translation remains owned by `model/<provider>`, while Runner owns step boundaries, fallback admission, terminal arbitration, and Run/Stream equivalence. Diagnostics replay is offline and `RuntimeRecorder` is the only diagnostics write path. See proposal.md for the motivation and scope.

## Goals / Non-Goals

**Goals:**

- Define one normalized semantic matrix for OpenAI, Anthropic, and Gemini handoff and stream-edge behavior.
- Keep provider-specific translation inside each provider adapter and make drift observable before it reaches Runner or Host.
- Preserve causal IDs, tool-call IDs, usage semantics, error classes, terminal outcomes, and Run/Stream parity across step-boundary fallback.
- Make every conformance claim reproducible through bounded fixtures, offline replay, and equivalent shell/PowerShell gates.
- Preserve backward compatibility for existing provider and replay fixtures by using additive fields and nullable/default behavior.

**Non-Goals:**

- Adding providers, changing provider SDK ownership, or introducing a shared provider wire protocol.
- Switching providers after a stream has emitted its first semantic event.
- Building runtime model discovery, a credential store, remote routing, hosted persistence, or a gateway.
- Persisting raw provider payloads, reasoning bodies, credentials, or unbounded stream content in diagnostics.

## Decisions

### 1. Normalize at adapter boundaries, not in Runner

Each `model/<provider>` adapter will map provider-native tool calls, tool results, thinking/reasoning fragments, usage, aborts, empty content, Unicode, and overflow into the existing canonical model/toolcontract types. Runner will consume only canonical values and retain ownership of fallback, step boundaries, and terminal arbitration.

**Alternative rejected:** a new shared provider-neutral translation layer. It would duplicate adapter ownership and risk leaking provider-specific assumptions into `core/runner`.

### 2. Use a versioned conformance matrix and fixture digest

The change introduces a versioned fixture namespace, recommended as `provider_handoff_stream_edge.v1`. Each case records provider, mode, semantic input, normalized expected output, causal correlation, usage/abort outcome, and an explicit drift class. Fixture payloads are bounded and redactable; raw provider responses are not persisted.

**Alternative rejected:** snapshotting complete provider responses. Raw payloads are high-cardinality, may contain secrets, and make fixtures brittle across harmless SDK changes.

### 3. Treat stream start as the fallback fence

Capability and request-shape checks happen before provider invocation. A provider may be replaced only before the first semantic stream event. After stream start, errors are normalized and the current step terminates according to existing fail-fast/terminal policy; no mid-stream provider switch is allowed.

**Alternative rejected:** mid-stream recovery by concatenating providers. It breaks causal ordering, usage accounting, deduplication, and Run/Stream equivalence.

### 4. Classify drift, do not infer it from provider names

Canonical drift classes cover handoff schema, tool-call correlation, tool-result feedback, thinking projection, stream boundary, abort/usage, overflow, Unicode/empty content, error taxonomy, fallback fence, and Run/Stream parity. Gate output uses stable classes and does not expose provider-only error strings as runtime contract.

### 5. Replay and gates are first-class evidence

Diagnostics replay validates the fixture schema, normalizes cases deterministically, compares expected digests, and proves side-effect freedom. A PowerShell and shell gate run the same matrix and structural assertions: adapter ownership, no raw payload diagnostics, no mid-stream fallback, bounded fixtures, and parity with existing provider tests.

## Risks / Trade-offs

- **[Risk] SDK upgrades alter benign response details.** → Compare canonical normalized projections and bounded digests rather than raw payloads; keep fixture versions additive.
- **[Risk] Provider behavior differs in ways the matrix does not model.** → Require positive, negative, boundary, and unknown-field cases, and fail closed on unsupported fixture versions.
- **[Risk] Usage/abort semantics are not available from every provider.** → Represent unavailable fields as nullable/default values with an explicit capability/reason classification; never synthesize precise usage.
- **[Risk] Conformance tests require live credentials.** → Use deterministic adapter fakes and recorded bounded semantic inputs; keep optional live-provider probes outside the required gate.
- **[Risk] Existing fixtures regress when replay support is added.** → Run mixed historical and new fixture suites and require idempotent normalized output.

## Migration Plan

1. Add the spec delta, normalized matrix types, fixtures, and replay support without changing existing defaults.
2. Add focused provider, integration, parity, and gate tests; initially report any discovered drift with stable classifications.
3. Correct only adapter-local normalization defects proven by fixtures; do not alter Runner ownership or introduce configuration keys.
4. Enable the conformance gate as a required quality step after the mixed-fixture suite and shell/PowerShell parity pass.
5. Roll back by removing the new gate/fixture namespace and reverting adapter-local changes; no persisted state or configuration migration is involved.

## Example Impact Assessment

修改示例。Only provider-independent semantic markers may be added to an existing agent-mode README/MATRIX baseline; no example may require live provider credentials or define a second provider contract.
