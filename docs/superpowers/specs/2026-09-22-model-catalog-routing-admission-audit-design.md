# Model Catalog Routing Admission Audit Design

## Problem and intent

Baymax already owns a host-supplied provider/model catalog, descriptor normalization,
capability negotiation, credential evidence, fallback admission, readiness projection,
and atomic catalog reload. The unresolved question is narrower: can the existing
exact-identity admission contract express a real host need to choose among multiple
local or host-supplied model candidates?

This design therefore starts with an offline, replayable audit. It must produce
evidence before any routing behavior is added. A candidate resolver is permitted only
when a reproducible fixture proves that exact-identity admission is not expressive
enough for an explicit host requirement.

## Chosen approach

The change introduces a versioned, provider-neutral
`model_catalog_routing_admission.v1` audit projection. Its inputs are supplied by the
host/config snapshot: catalog generation, normalized candidate identities, bounded
descriptor facts, required and optional capabilities, credential evidence status,
readiness policy, fallback declaration, and (if explicitly supplied) bounded priority
hints.

The audit normalizes identity and candidate order, evaluates each candidate through
the existing capability/credential/readiness/fallback semantics, and emits only
bounded facts: selected or blocked identity, ordered skip reasons, capability and
credential outcomes, fallback identity, generation, and canonical digests. It does
not perform discovery, network I/O, credential probing, provider calls, file access,
clock reads, or runtime mutation.

If the audit finds no expressiveness gap, the delivered result is the audit contract,
fixtures, replay, privacy/bounds checks, and gates; the resolver remains inactive and
the no-gap evidence is recorded. If a gap is proven, the follow-up implementation is
a pure function in `model/catalog` that consumes only the supplied snapshot,
candidates, and policy. It uses deterministic ranking and tie-break rules, rejects
ambiguous ties, reuses existing admission semantics, and remains opt-in. It does not
become a global mutable router or a second source of truth.

## Boundaries and ownership

- `model/catalog` owns canonical identity, descriptor normalization, catalog
  generation, and any conditional pure candidate ordering.
- `adapter/capability` remains the owner of capability negotiation facts.
- `runtime/config` remains the owner of snapshot publication, readiness policy,
  environment/file/default precedence, fail-fast validation, and atomic reload
  rollback.
- `observability/event.RuntimeRecorder` remains the single diagnostic write path.
- `tool/diagnosticsreplay` compares normalized audit facts offline and has no runtime
  side effects.
- `tool/contributioncheck` and shell/PowerShell gates enforce module/privacy and
  fixture invariants.

The change does not add remote discovery, background refresh, credential storage or
probe APIs, a provider-neutral wire protocol, a global router, provider SDK imports
to `context/*`, new terminal semantics, or automatic prompt/policy/memory/tool
mutation.

## Data flow and failure handling

```text
host snapshot + candidate identities + policy
                  |
                  v
       bounded canonical normalization
                  |
                  +--> duplicate/priority/generation/privacy validation
                  |
                  v
      existing capability/readiness admission facts
                  |
                  +--> exact-identity audit result
                  |
                  +--> conditional pure resolver (only with proven gap)
                  |
                  v
       selected/blocked identity + reasons + digest
```

Invalid input fails before provider action and produces no partial-success result.
Duplicate identities, conflicting priority metadata, ambiguous ties, unknown source
values, overflow, invalid generations, and privacy violations have stable bounded
reason codes. Existing strict/non-strict readiness, credential degradation/denial,
fallback, and atomic reload semantics remain authoritative.

Run and Stream use the same normalized facts and must produce equivalent selection,
admission, and parity digests. No mid-stream provider switching is introduced.

## Verification strategy

The first layer is contract and normalization tests: positive, negative, boundary,
privacy, historical-default, and idempotency cases. The fixture family covers success,
blocked, capability denial, credential degradation/denial, fallback, ambiguous
selection, reload rollback, and Run/Stream parity. Fixtures contain no endpoints,
credentials, raw provider responses, prompts, or unbounded bodies.

Replay runs each fixture twice and compares canonical digest, reason order, generation,
selected/skipped correlations, and compatibility defaults. Contribution checks prove
that audit/resolver code cannot import provider SDKs or introduce discovery,
credential-store, global-router, shared-wire, or `context/*` dependencies. Shell and
PowerShell gates run the same schema, taxonomy, privacy, replay, rollback, and parity
checks. Full repository tests and quality/docs gates remain required before archive.

## Alternatives considered

1. **Immediate local router** — rejected because it would freeze ranking, fallback,
   and ownership semantics without reproducible host demand.
2. **Remote catalog/discovery SPI** — rejected because it expands lifecycle,
   networking, credential, and security ownership beyond the library-first boundary.
3. **Selection in runtime readiness or provider adapters** — rejected because catalog
   facts belong to `model/catalog`; readiness observes/admit policy and provider SDK
   details remain provider-owned.

## Example impact

`无需示例变更（附理由）`: the audit phase changes neither agent-mode configuration,
runtime paths, nor expected markers. If a later resolver changes observable example
selection or output, `examples/agent-modes/MATRIX.md` and the affected mode README
must be updated first and the assessment changed to `修改示例`.

