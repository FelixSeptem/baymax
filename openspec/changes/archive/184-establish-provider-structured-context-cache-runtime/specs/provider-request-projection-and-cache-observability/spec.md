## ADDED Requirements

### Requirement: Request projections SHALL record prompt-cache intent without exposing opaque values

The `provider_request_projection.v1` contract MUST add an additive/default request-side cache-control projection that records whether a directive was present, its bounded provider-specific kind, and only the safe structural facts needed to verify mapping. It MUST exclude raw OpenAI cache keys, Gemini cached-content resource names, prompt content, and provider SDK payloads. Historical fixtures that omit the new projection MUST continue to parse as no directive. Run and Stream projections MUST agree on cache-control intent and observed native mapping. Request-side cache intent MUST remain separate from response-side cache usage and MUST NOT assert that a provider cache hit occurred.

#### Scenario: Historical fixture without cache intent remains compatible
- **WHEN** replay reads a pre-change request projection fixture without cache-control fields
- **THEN** replay interprets the fields as absent/default and preserves its existing canonical behavior

#### Scenario: Cache directive mapping is projected without raw values
- **WHEN** an adapter receives a valid provider-specific cache directive
- **THEN** the normalized projection records the bounded directive kind and mapping facts but omits the original key or resource reference

#### Scenario: Run and Stream cache-control projection differs
- **WHEN** equivalent Run and Stream requests produce different cache-control presence or native mapping
- **THEN** replay fails with a stable request projection parity classification
