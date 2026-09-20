## ADDED Requirements

### Requirement: Supported adapter request projection SHALL preserve native semantic facts

For OpenAI, Anthropic, and Gemini, a `ModelRequest` whose `Messages` contains system, user, or assistant entries MUST preserve the same ordered role sequence at the provider SDK boundary whenever that provider SDK can express the role. The adapter MUST preserve the source-relative order of messages and the final user instruction, and MUST NOT silently replace a non-empty structured request with a single user text input.

A canonical tool result with a non-empty call identity and tool name MUST reach the provider SDK boundary using that provider's native tool-result or function-response shape when expressible. The provider request MUST preserve the result-to-call association and tool identity. Text-envelope projection is not a conforming success path for a supported native shape.

#### Scenario: Structured roles remain observable at the SDK boundary
- **WHEN** a request contains ordered system, user, and assistant messages together with a final user instruction
- **THEN** the normalized SDK-boundary projection retains the corresponding ordered role facts and does not collapse them into one user-text fact

#### Scenario: Tool result reaches a native request part
- **WHEN** a request contains a canonical tool result with valid call identity and tool name
- **THEN** the normalized SDK-boundary projection marks the result as native, retains the call/tool association, and does not mark it as a text-envelope result

#### Scenario: Unsupported native representation fails before invocation
- **WHEN** an adapter cannot safely express a canonical role or valid tool result in its provider-native request shape
- **THEN** it returns a deterministic request-shape failure before the provider request is sent and does not emit a partial text-envelope fallback

### Requirement: Request projection parity SHALL cover Run Stream and token accounting

For the same canonical request facts, each supported adapter's Run, Stream, and CountTokens paths MUST use semantically equivalent role ordering, final-input placement, and tool-result association rules. Differences that are solely required by a provider's SDK call shape MUST normalize to equivalent provider-neutral projection facts.

#### Scenario: Run and Stream preserve equivalent request semantics
- **WHEN** the same request is executed through Run and Stream on one supported adapter
- **THEN** the normalized provider-boundary role sequence, final-input placement, and tool-result association are equivalent

#### Scenario: Token accounting matches generation projection facts
- **WHEN** the same request is counted and then executed through Run or Stream on one supported adapter
- **THEN** the token-count request and generation request normalize to equivalent role and tool-result projection facts, apart from fields unavailable to the token-count API

### Requirement: Resolved request projection gaps SHALL be migrated explicitly

When a runtime repair resolves a previously declared role, tool-result-native, ordering, or Run/Stream projection gap, the versioned fixture MUST replace that `declared_gap` with an explicit no-gap native expectation in the same change. Replay and gate verification MUST fail if a resolved gap remains declared, if a newly observed gap is undeclared, or if the updated native expectation is not deterministic.

#### Scenario: Resolved gap cannot remain silently declared
- **WHEN** a fixture still declares a gap that is no longer observed after native request projection is implemented
- **THEN** replay fails with the existing request-projection contract drift classification until the fixture expectation is explicitly updated
