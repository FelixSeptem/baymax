## MODIFIED Requirements

### Requirement: Provider adapters SHALL normalize tool-result feedback contract for next model step

Provider adapters MUST accept canonical tool-result feedback and map it to provider-native request shape without semantic drift. Feedback MUST preserve original call identity, success/error meaning, bounded payload rules, and next-step eligibility; invalid feedback MUST fail before provider invocation.

For OpenAI, Anthropic, and Gemini, a valid feedback result MUST be represented as a provider-native tool-result or function-response request part wherever the official SDK supports that form. The adapter MUST NOT treat a serialized text envelope as semantically equivalent to an available native result part. If the adapter cannot safely construct the native part, it MUST fail deterministically before provider invocation rather than send a downgraded request.

#### Scenario: Valid feedback uses a native provider request part
- **WHEN** a provider adapter receives canonical feedback with a valid call identity, tool identity, and bounded result
- **THEN** the provider-boundary request carries an associated native tool-result or function-response part

#### Scenario: Native construction cannot be completed
- **WHEN** a provider adapter cannot safely construct the required native feedback part
- **THEN** it returns a deterministic request-shape failure before sending a provider request

#### Scenario: Invalid feedback remains fail-fast
- **WHEN** canonical feedback omits its call identity or tool identity
- **THEN** the adapter returns the existing feedback-invalid classification before provider invocation

#### Scenario: Equivalent tool-result feedback continues the loop
- **WHEN** equivalent canonical tool-result payloads are sent to OpenAI, Anthropic, and Gemini adapters
- **THEN** each adapter carries the original call correlation through a provider-native request part and continues with semantically equivalent next-step classification

#### Scenario: Runner sends canonical tool result to OpenAI adapter
- **WHEN** runner passes canonical tool-result payload after dispatch
- **THEN** adapter maps payload into an OpenAI-native follow-up request part and preserves correlation to original tool call

#### Scenario: Equivalent tool-result feedback through Anthropic and Gemini adapters
- **WHEN** equivalent canonical tool-result payload is sent to Anthropic and Gemini adapters
- **THEN** both adapters map it to their native request parts and continue the model step with semantically equivalent outcome classification

#### Scenario: Invalid feedback fails before invocation
- **WHEN** canonical feedback is missing required identity or exceeds bounds
- **THEN** the adapter returns canonical feedback-invalid classification and does not issue a provider request
