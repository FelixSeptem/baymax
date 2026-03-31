# llm-multi-provider-minimal Specification

## Purpose
TBD - created by archiving change bootstrap-multi-llm-providers-m1. Update Purpose after archive.
## Requirements
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

### Requirement: M1 SHALL not expose streaming placeholder APIs
The M1 multi-provider change MUST remain non-streaming only and MUST NOT add streaming placeholder APIs for Anthropic or Gemini.

#### Scenario: API surface is reviewed after M1
- **WHEN** maintainers inspect exported model adapter APIs
- **THEN** no new streaming placeholder interface or method is added for Anthropic/Gemini in this change

### Requirement: Provider errors SHALL map to baseline error classes
Anthropic and Gemini provider failures MUST map to baseline `types.ErrorClass` categories for operational consistency.

#### Scenario: Provider returns authentication or rate-limit errors
- **WHEN** provider-specific auth/limit errors occur
- **THEN** runtime returns mapped baseline error classes consistent with existing model error semantics

### Requirement: M2 follow-up TODOs SHALL be explicit
The repository MUST include explicit TODO markers in implementation and documentation for post-M1 work on fine-grained provider error mapping and streaming semantic alignment.

#### Scenario: Maintainer reviews M1 completion artifacts
- **WHEN** change artifacts and relevant docs are reviewed
- **THEN** explicit TODO items for M2 provider streaming and fine-grained error mapping are present

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

### Requirement: Provider adapters SHALL normalize tool-calling request contract for ReAct loop
OpenAI, Anthropic, and Gemini adapters MUST normalize provider-specific tool-call output into canonical runtime contract fields that can be consumed uniformly by runner loop logic.

Normalized tool-call request contract MUST include at minimum:
- `tool_call_id`
- `tool_name`
- canonical arguments payload
- step correlation metadata.

#### Scenario: OpenAI adapter emits tool call in ReAct step
- **WHEN** OpenAI response contains provider-specific function/tool-call structure
- **THEN** adapter returns canonical tool-call request fields consumable by shared runner loop

#### Scenario: Anthropic and Gemini adapters emit equivalent tool calls
- **WHEN** Anthropic and Gemini responses represent semantically equivalent tool-call intents
- **THEN** adapters map them into semantically equivalent canonical tool-call contract fields

### Requirement: Provider adapters SHALL normalize tool-result feedback contract for next model step
Provider adapters MUST accept canonical tool-result feedback from runner and MUST map it to provider-native request shape without semantic drift.

#### Scenario: Runner sends canonical tool result to OpenAI adapter
- **WHEN** runner passes canonical tool-result payload after dispatch
- **THEN** adapter maps payload into provider-native follow-up message and preserves correlation to original tool call

#### Scenario: Equivalent tool-result feedback through Anthropic and Gemini adapters
- **WHEN** equivalent canonical tool-result payload is sent to Anthropic and Gemini adapters
- **THEN** both adapters continue model step with semantically equivalent outcome classification

### Requirement: Provider tool-calling failures SHALL map to canonical error taxonomy
Provider-specific tool-calling failures MUST map to canonical error classification without leaking provider-only error semantics into runner contracts.

Minimum canonical classes for this milestone:
- capability unsupported,
- request shape invalid,
- tool result feedback invalid,
- provider rate or auth failure.

#### Scenario: Provider returns unsupported tool-calling capability error
- **WHEN** adapter receives provider-specific unsupported-tool-call error
- **THEN** adapter maps it to canonical capability-unsupported classification

#### Scenario: Provider rejects malformed tool-result feedback
- **WHEN** provider returns feedback payload validation error
- **THEN** adapter maps error to canonical feedback-invalid classification

### Requirement: Provider fallback in tool-calling flows SHALL remain step-boundary deterministic
Fallback across providers MUST happen before model step invocation begins. Once a step has started emitting Stream events, runtime MUST NOT switch provider for that step.

#### Scenario: Active provider lacks tool-calling capability before step invocation
- **WHEN** pre-step capability evaluation detects missing tool-calling support
- **THEN** runtime selects next configured provider candidate deterministically before issuing model call

#### Scenario: Stream step has started and provider fails mid-step
- **WHEN** provider failure occurs after stream emission starts for the current step
- **THEN** runtime terminates the step with canonical fail-fast classification instead of mid-step provider switch

