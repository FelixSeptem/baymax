## MODIFIED Requirements

### Requirement: Provider adapters SHALL normalize tool-calling request contract for ReAct loop

OpenAI, Anthropic, and Gemini adapters MUST normalize provider-specific tool-call output into canonical runtime contract fields that can be consumed uniformly by runner loop logic. Normalized output MUST preserve `tool_call_id`, `tool_name`, canonical arguments, step correlation, and equivalent semantics for thinking/reasoning metadata when present. Provider-only fields MUST remain optional and MUST NOT become required runner semantics.

#### Scenario: Equivalent tool calls preserve correlation
- **WHEN** supported providers emit semantically equivalent tool-call structures
- **THEN** adapters return equivalent canonical tool-call identity, arguments, and step/run correlation

#### Scenario: OpenAI adapter emits tool call in ReAct step
- **WHEN** OpenAI response contains provider-specific function/tool-call structure
- **THEN** adapter returns canonical tool-call request fields consumable by shared runner loop

#### Scenario: Anthropic and Gemini adapters emit equivalent tool calls
- **WHEN** Anthropic and Gemini responses represent semantically equivalent tool-call intents
- **THEN** adapters map them into semantically equivalent canonical tool-call contract fields

#### Scenario: Optional reasoning metadata is unavailable
- **WHEN** an adapter cannot expose provider-native thinking/reasoning metadata
- **THEN** the canonical projection uses nullable/default behavior without changing tool-call semantics

### Requirement: Provider adapters SHALL normalize tool-result feedback contract for next model step

Provider adapters MUST accept canonical tool-result feedback and map it to provider-native request shape without semantic drift. Feedback MUST preserve original call identity, success/error meaning, bounded payload rules, and next-step eligibility; invalid feedback MUST fail before provider invocation.

#### Scenario: Equivalent tool-result feedback continues the loop
- **WHEN** equivalent canonical tool-result payloads are sent to OpenAI, Anthropic, and Gemini adapters
- **THEN** each adapter continues with semantically equivalent next-step classification and original call correlation

#### Scenario: Runner sends canonical tool result to OpenAI adapter
- **WHEN** runner passes canonical tool-result payload after dispatch
- **THEN** adapter maps payload into provider-native follow-up message and preserves correlation to original tool call

#### Scenario: Equivalent tool-result feedback through Anthropic and Gemini adapters
- **WHEN** equivalent canonical tool-result payload is sent to Anthropic and Gemini adapters
- **THEN** both adapters continue model step with semantically equivalent outcome classification

#### Scenario: Invalid feedback fails before invocation
- **WHEN** canonical feedback is missing required identity or exceeds bounds
- **THEN** the adapter returns canonical feedback-invalid classification and does not issue a provider request

### Requirement: Streaming fallback scope SHALL remain step-boundary only

The runtime MUST NOT switch provider after stream emission has started for a model step. Stream normalization MUST preserve start, partial, completion, abort, empty content, Unicode, usage, and overflow semantics through the existing canonical event and error contracts. Unsupported optional usage MUST remain nullable/defaultable rather than fabricated.

#### Scenario: Post-start abort does not switch provider
- **WHEN** a provider fails or aborts after the first semantic stream event
- **THEN** the current step terminates with canonical classification and no second provider contributes events

#### Scenario: Streaming step has already emitted events
- **WHEN** provider-side capability mismatch or unsupported feature is detected after first stream event emission
- **THEN** runtime terminates current step according to fail-fast semantics instead of switching to another provider mid-stream

#### Scenario: Empty and Unicode stream boundaries remain equivalent
- **WHEN** equivalent providers emit empty content or valid Unicode content
- **THEN** Run and Stream normalized outputs preserve the same content and event-boundary semantics
