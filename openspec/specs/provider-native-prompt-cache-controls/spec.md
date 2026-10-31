# provider-native-prompt-cache-controls Specification

## Purpose
This capability lets a host explicitly request provider-native prompt-cache behavior through a bounded model request directive. It preserves provider-specific semantics and never implies that a request was cached or that cost or latency improved.

## Requirements

### Requirement: Model requests SHALL accept bounded provider-native prompt-cache directives

`ModelRequest` MUST support an optional, additive provider-tagged prompt-cache directive that contains exactly one provider-specific variant. The provider discriminator MUST be one of `openai`, `anthropic`, or `gemini` and MUST match the selected adapter. The OpenAI Responses variant MUST contain one non-empty opaque prompt-cache key of at most 64 UTF-8 bytes. The Anthropic Messages variant MUST contain a zero-based boundary index into the interpreted canonical message sequence and a TTL of `5m` or `1h`; the index MUST identify an existing message. The Gemini variant MUST contain one non-empty existing cached-content resource reference of at most 512 UTF-8 bytes. Provider SDK types MUST remain inside `model/<provider>` packages.

When the directive is absent, adapters MUST preserve the existing request behavior. A directive whose provider tag does not match the selected adapter, whose variant is missing or conflicting, or whose values are malformed or out of bounds MUST fail before provider invocation. Adapters MUST NOT silently ignore a directive or translate it into a different provider's cache behavior. Error text MUST NOT echo opaque key or resource reference values.

#### Scenario: Omitted directive preserves default request behavior
- **WHEN** a host sends a `ModelRequest` without a prompt-cache directive
- **THEN** each adapter builds the same native request shape and behavior as before this capability, with no cache-control field added by this feature

#### Scenario: Supported directive maps to its provider-native request field
- **WHEN** a host sends a valid directive matching the selected provider adapter
- **THEN** the adapter maps only that directive to the documented native request field and preserves the original structured message and tool-result projection

#### Scenario: Provider mismatch and malformed directive fail before invocation
- **WHEN** the directive provider differs from the selected adapter, variants conflict, or a value is invalid or exceeds its bound
- **THEN** the adapter returns a deterministic typed request-shape error and the provider SDK invocation count remains zero

### Requirement: Prompt-cache controls SHALL remain opt-in and avoid cache-resource lifecycle ownership

The runtime MUST NOT automatically create, update, delete, discover, persist, or resolve provider cache resources. A Gemini cached-content directive MUST reference an already-existing resource supplied by the caller. An Anthropic breakpoint MUST identify a valid boundary in the structured request and MUST NOT alter message content or ordering. An OpenAI key MUST be treated as an opaque caller-provided routing key, not as proof of cache identity or authorization. Cache controls MUST NOT change model selection, retries, tool behavior, or terminal semantics.

#### Scenario: Gemini uses only a caller-supplied resource reference
- **WHEN** a valid Gemini directive names an existing cached-content resource
- **THEN** the adapter attaches the reference to the request and performs no cache create/get/list/update/delete operation

#### Scenario: Cache directive does not change execution behavior
- **WHEN** a valid prompt-cache directive is supplied to equivalent Run and Stream requests
- **THEN** both paths use equivalent cache controls and retain the same terminal, retry, tool, and structured-context semantics

#### Scenario: Cache directives are not evidence of cache hits
- **WHEN** a request includes a valid prompt-cache directive but provider usage omits cache accounting
- **THEN** the response reports cache usage as unavailable/default and does not infer a hit, saved cost, or latency improvement

### Requirement: Prompt-cache directive values SHALL be protected from diagnostics and conformance artifacts

The public runtime MUST NOT write raw cache keys, Anthropic breakpoint values, Gemini resource names, credentials, or prompt content to diagnostics, replay fixtures, or normalized request projections. Projection evidence MAY include bounded provider identity, directive kind, boundary index, TTL class, and a one-way digest of an opaque reference when necessary for parity; it MUST NOT contain the original key or resource reference. The digest MUST NOT be used as a cache key or as proof of cache isolation.

#### Scenario: Projection and diagnostics omit caller secrets
- **WHEN** a request contains an opaque cache key or provider resource reference
- **THEN** projection, diagnostics, and replay artifacts expose only allowed bounded classifications and never contain the original value

#### Scenario: Cache intent and observed cache usage remain separate facts
- **WHEN** a cache directive is present but no provider cache usage is observed
- **THEN** conformance reports request intent independently and retains unavailable cache usage without synthesizing a hit
