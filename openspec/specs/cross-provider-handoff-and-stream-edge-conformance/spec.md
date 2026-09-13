# cross-provider-handoff-and-stream-edge-conformance Specification

## Purpose
This capability defines a bounded, replayable contract for preserving equivalent handoff and stream-edge semantics across the supported OpenAI, Anthropic, and Gemini adapters without introducing a new provider owner or routing state machine.

## Requirements

### Requirement: Provider handoff SHALL normalize to one canonical contract

The supported provider adapters MUST normalize semantically equivalent tool-call requests, canonical tool-result feedback, thinking/reasoning projections, step correlation, and error classes into the existing runtime contract. The normalized projection MUST preserve `tool_call_id`, tool identity, canonical arguments/result shape, and causal step/run correlation without exposing provider-only fields as required runtime semantics.

#### Scenario: Equivalent tool call from each provider
- **WHEN** OpenAI, Anthropic, and Gemini produce semantically equivalent tool-call intents
- **THEN** normalization yields equivalent tool identity, arguments, call correlation, and next-step admission semantics

#### Scenario: Provider-only field is absent
- **WHEN** a provider does not expose an optional reasoning or usage field
- **THEN** the normalized contract uses its documented nullable/default representation and does not synthesize a misleading value

### Requirement: Tool-result handoff SHALL preserve correlation and feedback semantics

Adapters MUST accept canonical tool-result feedback and map it to provider-native input while preserving the original tool-call correlation, success/error meaning, and next model-step eligibility. Malformed feedback MUST fail with the canonical feedback-invalid classification before a provider request is issued.

#### Scenario: Canonical feedback round-trip
- **WHEN** the runner sends equivalent canonical tool results to each supported adapter
- **THEN** each adapter produces provider-native feedback that preserves call identity and equivalent continuation classification

#### Scenario: Malformed feedback is rejected
- **WHEN** canonical feedback lacks required correlation or violates bounded shape
- **THEN** the adapter rejects it deterministically as feedback-invalid without issuing a partial provider call

### Requirement: Stream edges SHALL be deterministic and provider-independent

Streaming normalization MUST distinguish start, partial content, terminal completion, abort, empty content, Unicode content, and overflow outcomes. Once the first semantic stream event is emitted, the runtime MUST NOT switch providers for that step. Usage and abort fields MUST remain nullable/defaultable when unsupported and MUST NOT be fabricated.

#### Scenario: Stream starts and completes
- **WHEN** equivalent providers emit a valid stream with zero or more partial events followed by completion
- **THEN** normalized events preserve start/partial/terminal semantics and equivalent final outcome classification

#### Scenario: Stream aborts after emission
- **WHEN** a provider aborts after the first semantic stream event
- **THEN** the current step ends with canonical abort/terminal classification and no provider fallback occurs mid-stream

#### Scenario: Empty or Unicode content is preserved
- **WHEN** a stream contains empty content or valid Unicode including U+2028/U+2029
- **THEN** normalization preserves content semantics without dropping, escaping into a different value, or changing event order

#### Scenario: Overflow is bounded
- **WHEN** provider content or a normalized event exceeds the configured contract bound
- **THEN** the step fails fast with a stable overflow classification and does not emit unbounded diagnostics or continue with truncated causal data

### Requirement: Provider fallback SHALL remain step-boundary deterministic

Capability and request-shape checks MUST occur before provider invocation. Fallback MAY select the next configured provider only before model-step execution or before the first semantic stream event; after that fence, the current step MUST terminate through existing error and terminal arbitration.

#### Scenario: Pre-step capability fallback
- **WHEN** the active provider cannot satisfy required capabilities before invocation and a later candidate can
- **THEN** the runtime selects the later candidate deterministically and preserves causal correlation

#### Scenario: Mid-stream provider failure
- **WHEN** the active provider fails after stream emission begins
- **THEN** the runtime reports canonical failure/abort semantics without concatenating output from another provider

### Requirement: Run and Stream SHALL remain semantically equivalent

Equivalent handoff and stream-edge workloads executed through Run and Stream MUST preserve normalized tool-call/result semantics, causal references, error classes, fallback fence, usage/abort classification, and terminal outcome after permitted event-order normalization. The contract MUST NOT introduce a Stream-only or Run-only provider state machine.

#### Scenario: Equivalent Run and Stream handoff
- **WHEN** equivalent Run and Stream executions perform the same provider handoff and tool-result feedback
- **THEN** their normalized causal, error, fallback, and terminal projections are equivalent

#### Scenario: Equivalent Run and Stream abort
- **WHEN** equivalent Run and Stream executions encounter the same post-start abort condition
- **THEN** both classify the abort and terminal outcome equivalently without provider switching

### Requirement: Conformance fixtures and replay SHALL be bounded and side-effect-free

The repository MUST provide versioned `provider_handoff_stream_edge.v1` fixtures covering valid handoff, malformed feedback, thinking/usage omission, stream completion, abort, empty/Unicode content, overflow, pre-step fallback, mid-stream failure, and Run/Stream parity. Replay MUST normalize deterministically, classify drift with stable reason codes, remain offline/read-only, and preserve compatibility with historical fixtures.

#### Scenario: Canonical fixture replays successfully
- **WHEN** replay processes a valid `provider_handoff_stream_edge.v1` fixture whose normalized digest matches expectation
- **THEN** replay succeeds deterministically without invoking providers, tools, or mutating runtime state

#### Scenario: Fixture drift is classified
- **WHEN** normalized output differs in correlation, event boundary, usage/abort, fallback fence, or parity
- **THEN** replay fails with the corresponding stable drift classification and no partial success

#### Scenario: Historical fixtures remain compatible
- **WHEN** new fixtures run alongside archived provider and replay fixtures
- **THEN** historical fixtures continue to parse and normalize with documented defaults

### Requirement: Conformance gates SHALL enforce ownership and diagnostics boundaries

Shell and PowerShell gates MUST execute equivalent checks for adapter ownership, no mid-stream fallback, bounded fixtures, canonical error taxonomy, Run/Stream parity, and replay idempotency. Provider raw payloads, reasoning bodies, credentials, and unbounded stream content MUST NOT be written to diagnostics or OTel attributes.

#### Scenario: Gate detects provider-specific leakage
- **WHEN** implementation requires provider-only fields, stores raw payloads, or emits unbounded diagnostic content
- **THEN** the conformance gate fails with a deterministic boundary classification

#### Scenario: Shell and PowerShell results agree
- **WHEN** both gate implementations run against the same repository state
- **THEN** they produce equivalent pass/fail classifications for the conformance matrix
