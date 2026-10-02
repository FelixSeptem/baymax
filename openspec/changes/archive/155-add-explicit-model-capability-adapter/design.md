## Context

`types.ModelClient` is intentionally minimal, while `core/runner` uses `types.ModelCapabilityDiscovery` for provider naming, strict capability admission, and fallback ordering. The adapter package already owns external integration contracts, so an explicit model wrapper belongs there and must remain independent of provider SDKs.

## Goals / Non-Goals

**Goals:**

- Provide a small, reusable wrapper for local/test model clients that explicitly declares provider identity and supported capabilities.
- Preserve existing model delegation semantics and strict preflight/fallback behavior.
- Make the missing discovery contract actionable in runner errors and examples.

**Non-Goals:**

- Inferring capabilities from Go method sets or model behavior.
- Changing the `types.ModelClient` interface or provider SDK adapters.
- Adding runtime configuration, persistence, diagnostics storage, or a second fallback mechanism.

## Decisions

1. **Package boundary:** Add `adapter/modelcapability` with a `Config` and `Adapter`. It imports `core/types` only and does not depend on `core/runner`, provider SDKs, or runtime configuration.
2. **Explicit declaration:** `Config.Provider` is required; `Config.Model` is optional; `Config.Capabilities` is normalized, validated against the canonical `types.ModelCapability` values, deduplicated, and sorted. Discovery returns `ProviderCapabilities` with supported entries for declared capabilities and unsupported entries for known capabilities not declared, using `Source: explicit_adapter`.
3. **Delegation:** `Generate` and `Stream` directly call the wrapped model. The wrapper is a transparent execution boundary and never remaps model errors.
4. **Runner diagnostics:** Keep the existing classified model error and fallback loop. For candidates that fail the type assertion, record `capability_discovery_unavailable`, include `missing_interfaces`, and use an actionable message. Discovered-but-missing capabilities continue to use `capability_unsupported`.
5. **Example/documentation:** Update the model template and adapter docs to show `modelcapability.Wrap` with `streaming`, making the explicit opt-in visible at the first integration point.

Alternatives considered: expanding `ModelClient` would be breaking; implicit streaming inference would bypass strict preflight and make fallback nondeterministic; a runner-local registry would duplicate adapter ownership and create hidden global state.

## Risks / Trade-offs

- [Explicit declarations can be inaccurate] -> Keep the adapter opt-in and document that declarations are assertions; production provider adapters remain authoritative.
- [Changing error details may affect consumers] -> Preserve the existing error class and top-level classification, adding only bounded detail fields and a clearer message.
- [Static capability reports can become stale] -> Mark the source as `explicit_adapter` and scope this wrapper to local/test integrations.

## Migration Plan

Existing model clients continue to compile. Integrators that need strict Stream or capability-gated Run behavior wrap their client and declare capabilities. Rollback consists of removing the wrapper and reverting the additive diagnostic/example changes; no persisted data or configuration migration is required.

## Example Impact Assessment

修改示例

## Open Questions

None.
