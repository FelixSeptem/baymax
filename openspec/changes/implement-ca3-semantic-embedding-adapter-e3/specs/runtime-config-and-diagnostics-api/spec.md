## ADDED Requirements

### Requirement: Runtime config SHALL expose CA3 embedding scorer controls
Runtime config MUST expose CA3 embedding scorer controls including enablement flag, provider/model selector (OpenAI/Gemini/Anthropic), optional independent embedding credentials, timeout, cosine metric selector, and hybrid score weight fields with fail-fast validation.

#### Scenario: Startup with valid embedding scorer config
- **WHEN** runtime starts with valid CA3 embedding scorer configuration
- **THEN** effective config includes embedding scorer controls and CA3 can evaluate hybrid scoring path

#### Scenario: Hot reload with invalid embedding scorer config
- **WHEN** runtime receives invalid CA3 embedding scorer configuration update
- **THEN** runtime rejects update and preserves previous valid config snapshot

#### Scenario: Default embedding scorer config
- **WHEN** runtime loads default CA3 embedding scorer settings
- **THEN** effective defaults use cosine metric, `rule_weight=0.7`, `embedding_weight=0.3`, and shared quality threshold strategy

#### Scenario: Independent embedding credentials configured
- **WHEN** runtime config includes provider-specific embedding credentials
- **THEN** effective config uses independent embedding credentials for CA3 embedding calls

### Requirement: Diagnostics API SHALL include CA3 embedding scoring fields
Runtime diagnostics MUST include additive CA3 embedding scoring fields for adapter status, similarity contribution, and fallback reasons without breaking existing field semantics.

#### Scenario: Embedding scoring success
- **WHEN** CA3 completes embedding scoring successfully
- **THEN** diagnostics include embedding contribution fields and adapter status markers

#### Scenario: Embedding scoring fallback
- **WHEN** CA3 falls back from embedding scoring to rule-only path
- **THEN** diagnostics include explicit embedding fallback reason and fallback mode markers

#### Scenario: Provider path observability
- **WHEN** CA3 embedding scorer executes
- **THEN** diagnostics include which provider adapter path was selected for the scoring attempt
