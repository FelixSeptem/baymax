## MODIFIED Requirements

### Requirement: Runtime SHALL support minimal non-streaming multi-provider model adapters
The model layer MUST support at least OpenAI, Anthropic, and Gemini providers through the same runtime model contract for non-streaming execution.

This capability is extended in M2: the same three providers MUST also support aligned streaming semantics under the shared model event contract.

This capability is further extended in M3: before each model step, the runtime MUST evaluate requested capabilities against the active provider capability set and MUST attempt provider fallback according to configured priority when capabilities are not satisfied.

#### Scenario: Runner executes same prompt via different providers
- **WHEN** the same minimal prompt is executed using OpenAI, Anthropic, and Gemini adapters in non-stream mode
- **THEN** each adapter can return a valid final answer consumable by the same runner flow

#### Scenario: Runner streams same prompt via different providers
- **WHEN** the same minimal prompt is executed using OpenAI, Anthropic, and Gemini adapters in stream mode
- **THEN** each adapter emits stream events consumable by the same runner flow with aligned semantic outcomes

#### Scenario: Active provider lacks required capability before model step
- **WHEN** a request requires a capability not supported by the active provider for the selected model
- **THEN** runtime selects the next configured provider candidate that satisfies requested capabilities before issuing model invocation

### Requirement: New providers SHALL prefer official SDKs
Anthropic and Gemini adapters MUST prioritize official SDK usage to minimize future upgrade and migration cost.

M3 capability discovery MUST also prioritize official SDK-supported metadata or discovery methods, and MUST avoid static capability hardcoding as the primary source of truth.

#### Scenario: Provider adapter is implemented
- **WHEN** Anthropic or Gemini adapter code is introduced
- **THEN** implementation uses the corresponding official SDK as primary integration path

#### Scenario: Provider capability discovery is executed
- **WHEN** runtime requests capability discovery for an adapter
- **THEN** adapter resolves capabilities through official SDK-supported methods or metadata before considering static fallback defaults

## ADDED Requirements

### Requirement: Provider fallback SHALL be deterministic and fail fast when exhausted
The runtime MUST evaluate provider candidates in configured order for each model step and MUST terminate immediately with normalized error classification when no provider can satisfy requested capabilities.

#### Scenario: Fallback chain contains a valid candidate
- **WHEN** first provider in chain fails capability preflight and a later candidate satisfies requirements
- **THEN** runtime uses the first valid candidate and continues execution with consistent run/stream semantics

#### Scenario: Fallback chain is exhausted
- **WHEN** no provider candidate satisfies requested capabilities for the current model step
- **THEN** runtime aborts the run with fail-fast error and MUST NOT execute partial provider calls

### Requirement: Streaming fallback scope SHALL remain step-boundary only
The runtime MUST NOT switch provider after stream emission has started for a model step.

#### Scenario: Streaming step has already emitted events
- **WHEN** provider-side capability mismatch or unsupported feature is detected after first stream event emission
- **THEN** runtime terminates current step according to fail-fast semantics instead of switching to another provider mid-stream
