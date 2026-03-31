## ADDED Requirements

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
