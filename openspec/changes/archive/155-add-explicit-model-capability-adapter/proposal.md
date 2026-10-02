## Why

`types.ModelClient` intentionally remains a small Generate/Stream contract, but the runner's strict model-step preflight also needs provider identity and capability discovery. A minimal client that can stream therefore fails with the opaque `no provider satisfies required capabilities` error. An explicit adapter gives local/test models a deterministic opt-in path while preserving capability negotiation for production providers.

## What Changes

- Add a model capability adapter that wraps any `types.ModelClient` and explicitly declares provider identity, model identity, and supported model capabilities.
- Keep the adapter opt-in: it must never infer streaming or tool-call support from the wrapped client's method set.
- Improve runner preflight details when a model does not implement `ModelCapabilityDiscovery`, naming the missing interface and the adapter path.
- Update the model adapter template and external-adapter documentation to show the strict Stream preflight requirements.
- Add unit and integration coverage for adapter validation, delegation, capability discovery, Stream admission, and diagnostic parity.

## Example Impact Assessment

修改示例

## Capabilities

### New Capabilities

- `explicit-model-capability-adapter`: Explicitly wraps a minimal model client with provider identity and declared capabilities for runtime preflight.

### Modified Capabilities

- `llm-multi-provider-minimal`: Clarify the required preflight diagnostic when a candidate lacks `ModelCapabilityDiscovery` while preserving fallback and fail-fast semantics.

## Impact

- Affected code: new `adapter/modelcapability` package and `core/runner` preflight diagnostics.
- Affected documentation/examples: `adapter/README.md`, external adapter template index, and model adapter template.
- Public API: additive Go package; existing `types.ModelClient` implementations remain source-compatible.
- No provider SDK, transport, persistence, or runtime configuration changes.
