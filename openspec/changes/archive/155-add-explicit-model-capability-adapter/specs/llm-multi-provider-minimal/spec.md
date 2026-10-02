## MODIFIED Requirements

### Requirement: Runtime SHALL support minimal non-streaming multi-provider model adapters
The model layer MUST support at least OpenAI, Anthropic, and Gemini providers through the same runtime model contract for non-streaming execution.

This capability is extended in M2: the same three providers MUST also support aligned streaming semantics under the shared model event contract.

This capability is further extended in M3: before each model step, the runtime MUST evaluate requested capabilities against the active provider capability set and MUST attempt provider fallback according to configured priority when capabilities are not satisfied. A candidate that does not implement the capability discovery contract MUST be rejected before invocation with a diagnostic that identifies the missing `ModelCapabilityDiscovery` interface, including its `ProviderName` and `DiscoverCapabilities` methods; this condition MUST remain distinct from a discovered provider that explicitly lacks a requested capability.

#### Scenario: Runner executes same prompt via different providers
- **WHEN** the same minimal prompt is executed using OpenAI, Anthropic, and Gemini adapters in non-stream mode
- **THEN** each adapter can return a valid final answer consumable by the same runner flow

#### Scenario: Runner streams same prompt via different providers
- **WHEN** the same minimal prompt is executed using OpenAI, Anthropic, and Gemini adapters in stream mode
- **THEN** each adapter emits stream events consumable by the same runner flow with aligned semantic outcomes

#### Scenario: Active provider lacks required capability before model step
- **WHEN** a request requires a capability not supported by the active provider for the selected model
- **THEN** runtime selects the next configured provider candidate that satisfies requested capabilities before issuing model invocation

#### Scenario: Candidate lacks capability discovery
- **WHEN** a strict Run or Stream request reaches a candidate that implements `types.ModelClient` but not `types.ModelCapabilityDiscovery`
- **THEN** runtime skips the candidate before model invocation and reports a diagnostic naming the missing interface and its required methods
