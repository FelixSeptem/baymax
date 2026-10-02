## Context

The existing `model/openai` adapter constructs OpenAI Responses requests locally, supports `BaseURL`, exposes cache usage when the official response contains it, and keeps CountTokens explicitly unsupported. Existing request-projection and provider cache fixtures already provide SDK-neutral canonical facts and historical replay behavior. The new contract must widen validated reach without turning an endpoint URL into an implicit provider identity or treating superficial API similarity as proof of compatibility.

## Goals / Non-Goals

**Goals:**

- Make endpoint compatibility an explicit, bounded, host-owned profile.
- Validate the profile before a request is admitted and route all projection through the existing OpenAI adapter owner.
- Prove structured context, tool-result, ordering, stream boundary, cache usage, and Run/Stream parity through deterministic fixtures and replay.
- Preserve privacy, module boundaries, historical fixture compatibility, and explicit unsupported capability semantics.

**Non-Goals:**

- No automatic endpoint probing, model catalog discovery, provider guessing, or fallback router.
- No new credential store, runtime control plane, shared provider wire protocol, or provider SDK dependency in `context/*`.
- No promise that every endpoint advertising OpenAI compatibility supports every Responses feature.
- No live-provider requirement for the conformance gate and no automatic fixture or source mutation.

## Decisions

### 1. Explicit profile over URL inference

The profile contains a bounded profile id, endpoint identity, model identity, API shape version, and declared capabilities. A non-empty `BaseURL` alone never authorizes compatibility. This avoids silently applying OpenAI semantics to an endpoint with different tool or stream behavior.

Alternative: infer compatibility from URL or a probe request. Rejected because it is nondeterministic, can leak credentials or prompts, and creates an implicit discovery contract.

### 2. Reuse the OpenAI adapter owner

The profile is consumed at the OpenAI adapter/config boundary; canonical facts remain provider-neutral and native SDK construction remains in `model/openai`. `context/*` and runtime orchestration do not import OpenAI SDK types.

Alternative: create a generic compatibility adapter or shared wire layer. Rejected because it would duplicate request construction and flatten provider-specific capability boundaries.

### 3. Capability declarations are conservative

The profile declares support for structured input, native tool results, streaming, cache usage, and CountTokens independently. Missing or unknown declarations resolve to unavailable/unknown and cannot be fabricated from a successful HTTP response. CountTokens remains unsupported unless an explicit, tested capability is added later.

### 4. Conformance is fixture-first and review-only

Versioned fixtures carry bounded canonical facts, expected event sequences, cache usage, and parity references without raw prompts, reasoning, credentials, or full SDK payloads. Shell and PowerShell gates run the same offline replay and fail on undeclared differences; they never modify source or fixtures.

## Risks / Trade-offs

- [Endpoint claims compatibility but diverges at runtime] -> Require profile admission plus conformance fixtures and keep live behavior opt-in and host-owned.
- [Profile fields become a second provider catalog] -> Keep identity and capability declarations local to the request/config boundary; do not add remote persistence or global registration.
- [Cache or stream support is overclaimed] -> Model each capability independently and default absent usage to unavailable; gate fabricated usage and parity drift.
- [Third-party endpoint adds unsupported extensions] -> Ignore unknown additive fields in replay and require a new reviewed fixture for any behavior Baymax intends to consume.

## Migration Plan

1. Add the profile schema and validation without changing the official OpenAI default path.
2. Add canonical conformance fixtures for official Responses behavior and one explicit compatible-profile example using only sanitized facts.
3. Wire replay, package tests, and paired shell/PowerShell gates; preserve historical projection fixtures.
4. Document admission, capability limits, rollback, and unexecuted live-provider checks.
5. Roll back by disabling profile admission and removing only new fixtures/gates; the existing OpenAI adapter remains unchanged.

## Open Questions

无。The profile is explicitly host-supplied, conformance is offline, and unsupported capabilities remain unavailable; these choices define the scope for implementation.
