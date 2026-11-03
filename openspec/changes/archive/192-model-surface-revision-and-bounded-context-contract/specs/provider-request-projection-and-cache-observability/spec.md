## ADDED Requirements

### Requirement: Request projection SHALL expose an explicit bounded model surface

`provider_request_projection.v1` SHALL support an optional
provider-neutral model-surface projection for the configuration, prompt, and
tool-schema inputs that a host explicitly supplies as model-visible evidence.
For each of those three surfaces, the projection SHALL record only
`available`, a bounded revision, a canonical SHA-256 digest, and a byte length.
It MUST NOT record raw configuration, prompt text, tool schema, credentials,
reasoning, provider payload, opaque cache value, or a body-derived excerpt.

Each supplied revision and digest MUST be valid UTF-8 and at most 256 UTF-8 bytes; each supplied
length MUST be nonnegative and no greater than 256 KiB; the serialized
model-surface projection MUST be at most 4 KiB.  An absent surface SHALL
canonicalize to `available=false` with empty revision/digest and zero length.
The projection is opt-in: it MUST NOT infer a configuration, prompt, or
tool-schema fact from roles, parts, tool order, capabilities, cache controls,
or any other field.  Invalid or over-limit evidence MUST fail fast with a
stable request-projection drift classification and MUST NOT produce a partial
canonical projection.

#### Scenario: Explicit model-surface facts are deterministic and private
- **WHEN** a caller explicitly supplies valid configuration, prompt, and
  tool-schema revision/digest/length facts
- **THEN** canonical request projection and digest are deterministic, retain
  those safe facts, and contain none of the underlying raw values

#### Scenario: Omitted surface stays unavailable without inference
- **WHEN** an otherwise valid request omits any model-surface input
- **THEN** that surface canonicalizes to unavailable/default and no role,
  capability, tool-order, cache-control, or provider-specific fact is used to
  manufacture a replacement value

#### Scenario: Invalid model-surface evidence is rejected atomically
- **WHEN** a supplied surface has a negative or over-limit length, an invalid
  UTF-8 or over-limit revision/digest, an invalid digest shape, or a serialized
  model-surface projection beyond its bound
- **THEN** projection validation fails with a stable classification before any
  provider invocation and returns no truncated or partially canonical result

### Requirement: Model-surface evidence SHALL participate in replay and parity

Request facts and provider-bound observations SHALL preserve equivalent
explicit model-surface summaries for semantically equivalent Run and Stream
paths.  Offline replay SHALL compare the summaries, detect a safe-fact
revision/digest/availability/length mismatch with a stable drift
classification, and remain deterministic, read-only, and independent of
provider SDKs, a host resolver, diagnostics, network, or execution.

Historical fixtures without a model-surface field MUST parse with every surface
unavailable/default.  Unknown future fields MUST be ignored without affecting
the current canonical digest.  The projection SHALL remain reference-only and
MUST NOT authorize model selection, context access, terminal transitions,
source recovery, or business effects.

#### Scenario: Run and Stream preserve the same explicit summary
- **WHEN** the same explicitly supplied model-surface facts reach equivalent
  Run and Stream request paths
- **THEN** replay reports semantically equal summaries and their canonical
  projections agree on the model-visible evidence

#### Scenario: Replay finds a summary mismatch
- **WHEN** a fixture records different available revision, digest, or length
  facts between an expected and observed model surface
- **THEN** replay returns the stable model-surface drift classification without
  calling a provider, resolver, tool, RuntimeRecorder, or effect owner

#### Scenario: Historical fixture remains compatible
- **WHEN** replay reads a valid fixture written before model-surface evidence
  existed
- **THEN** it defaults all three surfaces to unavailable and preserves the
  historical request-projection semantics
