## ADDED Requirements

### Requirement: Runtime SHALL support minimal non-streaming multi-provider model adapters
The model layer MUST support at least OpenAI, Anthropic, and Gemini providers through the same runtime model contract for non-streaming execution.

#### Scenario: Runner executes same prompt via different providers
- **WHEN** the same minimal prompt is executed using OpenAI, Anthropic, and Gemini adapters in non-stream mode
- **THEN** each adapter can return a valid final answer consumable by the same runner flow

### Requirement: New providers SHALL prefer official SDKs
Anthropic and Gemini adapters MUST prioritize official SDK usage to minimize future upgrade and migration cost.

#### Scenario: Provider adapter is implemented
- **WHEN** Anthropic or Gemini adapter code is introduced
- **THEN** implementation uses the corresponding official SDK as primary integration path

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
