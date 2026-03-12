## Context

Baymax runtime currently supports OpenAI/Anthropic/Gemini through unified Generate/Stream contracts, but provider capability differences (tool-call behavior, structured output support, model-specific features) are still handled implicitly and can fail at execution time. The roadmap and v1 limitations already defer capability detection and provider-level fallback to R3 M3.

A static capability table is not sufficient because provider model capabilities evolve quickly. This change therefore prioritizes dynamic discovery via official SDK-accessible metadata/endpoints, with deterministic fail-fast semantics when no fallback candidate satisfies requested capabilities.

## Goals / Non-Goals

**Goals:**
- Introduce a provider capability model used by runner before each model step.
- Discover capabilities dynamically via official SDK methods where available; only use static defaults as minimal fallback baseline.
- Add model-step-level provider fallback chain with deterministic ordering and fail-fast termination when no candidate is valid.
- Keep external Run/Stream semantics consistent (no mid-stream provider switch).
- Extend diagnostics and docs so fallback decisions are observable and behavior/docs remain aligned.

**Non-Goals:**
- No token-level or mid-stream provider switching.
- No cost/latency optimization routing beyond capability satisfaction.
- No new provider onboarding in this change.
- No CLI surface for diagnostics (library API only remains unchanged).

## Decisions

### Decision 1: Add a capability discovery contract in model adapters
- Choice: each provider adapter exposes capability discovery through a shared interface (for example `DiscoverCapabilities(ctx, model)`), internally using official SDK-supported metadata/discovery methods.
- Rationale: keeps provider logic local and future-proof to fast model changes; avoids central static matrix churn.
- Alternative considered: maintain a repository-wide static capability registry. Rejected because it requires frequent manual updates and creates stale behavior risk.

### Decision 2: Execute capability check before model invocation
- Choice: runner performs preflight capability matching for each model step before calling Generate/Stream.
- Rationale: avoids predictable provider rejections and allows deterministic fallback before request execution.
- Alternative considered: retry-on-error fallback only. Rejected because provider error payloads are inconsistent and may arrive too late in streaming paths.

### Decision 3: Fallback scope is model-step only and fail-fast
- Choice: fallback is attempted only before a step begins. If current provider cannot satisfy requested capabilities, runner tries next configured provider. If none match, runner aborts immediately with normalized error.
- Rationale: aligns with confirmed requirement for strict fail-fast and semantic stability.
- Alternative considered: switch provider during stream. Rejected for semantic complexity and event ordering risk.

### Decision 4: Extend runtime config + diagnostics minimally
- Choice: add provider fallback policy fields (ordered candidates, discovery timeout/cache policy) under runtime config with existing precedence/validation semantics; add diagnostics fields for capability check result and fallback path summary.
- Rationale: integrates with existing hot-reload and diagnostics architecture without introducing a new config subsystem.
- Alternative considered: hardcode fallback chain in code. Rejected due to poor operability.

## Risks / Trade-offs

- [Risk] SDK discovery APIs differ across providers and may be partially unavailable. -> Mitigation: define adapter-level graceful degradation rules and normalized `unknown` capability state.
- [Risk] Dynamic discovery may add request latency. -> Mitigation: add bounded cache with TTL and timeout; expose diagnostics for cache hit/miss and discovery latency.
- [Risk] Fallback behavior may hide provider misconfiguration. -> Mitigation: keep explicit diagnostics trail and fail-fast when chain exhausted.
- [Risk] Hot reload could alter fallback chain during traffic. -> Mitigation: retain atomic snapshot swap semantics; each run step reads one consistent config snapshot.

## Migration Plan

1. Introduce capability types and adapter discovery interface behind internal feature flag path.
2. Wire preflight capability matching into runner model step for both Run and Stream entry points.
3. Add runtime config fields and validation for fallback order/discovery controls.
4. Add diagnostics schema extensions and recorder plumbing.
5. Add contract/integration tests for success fallback, chain exhaustion fail-fast, and stream semantic consistency.
6. Update README + docs roadmap/acceptance limitations to reflect M3 delivery status.

Rollback strategy: disable capability-aware fallback in runtime config and keep existing provider selection behavior while preserving compiled binaries.

## Open Questions

- Provider-specific SDK discovery availability may vary by model family; adapters should document unsupported discovery methods and fallback behavior per provider.
- Whether fallback cache TTL should be globally configured or provider-specific can be revisited after first production telemetry.
