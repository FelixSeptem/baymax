## MODIFIED Requirements

### Requirement: Runtime SHALL support minimal non-streaming multi-provider model adapters
The model layer MUST support at least OpenAI, Anthropic, and Gemini providers through the same runtime model contract for non-streaming execution.

This capability is extended in M2: the same three providers MUST also support aligned streaming semantics under the shared model event contract.

#### Scenario: Runner executes same prompt via different providers
- **WHEN** the same minimal prompt is executed using OpenAI, Anthropic, and Gemini adapters in non-stream mode
- **THEN** each adapter can return a valid final answer consumable by the same runner flow

#### Scenario: Runner streams same prompt via different providers
- **WHEN** the same minimal prompt is executed using OpenAI, Anthropic, and Gemini adapters in stream mode
- **THEN** each adapter emits stream events consumable by the same runner flow with aligned semantic outcomes
