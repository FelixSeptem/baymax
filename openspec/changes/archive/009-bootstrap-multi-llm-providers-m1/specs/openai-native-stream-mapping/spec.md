## MODIFIED Requirements

### Requirement: OpenAI Streaming SHALL use native SDK event stream
`model/openai` MUST consume OpenAI Go official SDK Responses streaming events as the source of truth for stream output, instead of compatibility-only generate fallback behavior.

This requirement remains scoped to OpenAI streaming behavior. Anthropic/Gemini streaming alignment is out of scope for M1 multi-provider bootstrap and MUST be handled in a follow-up change.

#### Scenario: M1 multi-provider bootstrap is delivered
- **WHEN** Anthropic/Gemini non-stream adapters are added in M1
- **THEN** OpenAI native streaming behavior remains unchanged and Anthropic/Gemini streaming support is deferred to M2
