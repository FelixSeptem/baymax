## MODIFIED Requirements

### Requirement: OpenAI Streaming SHALL use native SDK event stream
`model/openai` MUST consume OpenAI Go official SDK Responses streaming events as the source of truth for stream output, instead of compatibility-only generate fallback behavior.

Under M2 multi-provider alignment, OpenAI streaming output MUST continue to use native SDK events while conforming to the shared cross-provider external semantic contract.

#### Scenario: OpenAI stream participates in cross-provider contract tests
- **WHEN** cross-provider streaming contract tests run
- **THEN** OpenAI stream remains native-SDK-driven and passes the same semantic assertions as Anthropic/Gemini adapters
