# openai-compatible-endpoint-profile-conformance Specification

## Purpose
This capability defines a conservative, explicit compatibility profile for OpenAI Responses-shaped endpoints so hosts can reuse Baymax's existing OpenAI projection only when structured context, tools, streaming, cache usage, and parity claims are bounded and replay-validated.

## Requirements

### Requirement: Endpoint compatibility SHALL require an explicit bounded profile

The system MUST require a host-supplied profile containing a bounded profile identity, endpoint identity, model identity, API shape version, and independent capability declarations before treating a non-default endpoint as OpenAI-compatible. A base URL or model name alone MUST NOT infer compatibility, discover capabilities, or authorize fallback.

#### Scenario: Explicit compatible profile is admitted
- **WHEN** a host supplies a valid profile with a supported Responses API shape and bounded endpoint/model identities
- **THEN** the request may use the existing OpenAI adapter path subject to the profile's declared capabilities

#### Scenario: Missing or malformed profile is rejected
- **WHEN** the profile is absent, exceeds bounds, contains sensitive material, or declares an unsupported API shape
- **THEN** admission fails fast with a stable validation reason and no provider request is sent

### Requirement: Capability declarations SHALL be conservative and independent

The profile MUST declare structured input, native tool results, streaming, cache usage, and CountTokens independently. An absent or unknown declaration MUST remain unavailable or unknown; successful HTTP transport or an unrecognized response field MUST NOT fabricate support. CountTokens MUST remain explicitly unsupported unless a separately reviewed capability is provided.

#### Scenario: Unsupported capability remains unavailable
- **WHEN** a compatible profile omits cache usage or CountTokens support
- **THEN** the corresponding capability is reported unavailable and no synthetic cache count or token count is produced

#### Scenario: Declared tool capability is bounded
- **WHEN** a profile declares native tool-result support
- **THEN** conformance requires correlated call identity, bounded arguments/results, and the declared Responses projection shape before admission succeeds

### Requirement: Conformance SHALL preserve structured projection and Run/Stream parity

The conformance contract MUST validate canonical role/part ordering, native tool-result correlation, stream event boundaries, terminal cache usage, first-semantic-event provider fencing, and equivalent Run/Stream facts for each declared capability. Historical `provider_request_projection.v1` fixtures and unavailable-cache defaults MUST remain replayable.

#### Scenario: Official Responses fixture passes conformance
- **WHEN** a sanitized fixture matches the declared profile's canonical roles, ordering, tool correlation, stream boundaries, and terminal usage
- **THEN** offline replay returns a deterministic no-drift conformance result with equivalent Run/Stream facts

#### Scenario: Undeclared projection difference blocks admission
- **WHEN** replay observes a role, tool-result, ordering, stream, cache, or parity difference not covered by the profile fixture
- **THEN** the gate fails with a stable reason and does not authorize the endpoint as compatible

### Requirement: Replay and gates SHALL be offline, deterministic, and privacy bounded

Profile validation and conformance replay MUST not call a provider, network, tool, runtime recorder, or credential store. Fixtures MUST exclude raw prompts, reasoning, credentials, full SDK payloads, and unbounded results. Shell and PowerShell gates MUST produce the same verdict, digest, reason ordering, and changed-surface summary without mutating files.

#### Scenario: Replaying the same fixture is idempotent
- **WHEN** the same profile and fixture are replayed repeatedly
- **THEN** the canonical digest, capability verdicts, reason ordering, and parity result are identical

#### Scenario: Privacy or bound violation fails fast
- **WHEN** a profile or fixture contains sensitive markers, oversized identities, unbounded tool data, or unknown unsafe fields in a required projection
- **THEN** replay returns a stable privacy/overflow/schema reason and performs no provider or filesystem mutation

### Requirement: Rollback SHALL remove only the compatibility extension

Rollback MUST disable the new profile admission and remove its conformance fixtures/gates/docs while preserving the official OpenAI adapter, existing BaseURL behavior, historical projection fixtures, and explicit unsupported CountTokens semantics.

#### Scenario: Compatibility extension is rolled back
- **WHEN** a gate regression or endpoint incompatibility requires rollback
- **THEN** the pre-change official OpenAI path and historical replay behavior remain available and no credential or runtime migration is required
