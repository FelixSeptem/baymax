## Context

`model/openai`, `model/anthropic`, and `model/gemini` own provider protocol
adaptation. `adapter/capability` already owns required/optional capability
negotiation, while `runtime/config.Manager.ReadinessPreflight()` owns the
canonical `ready|degraded|blocked` aggregation. These components can report
local facts, but a host cannot yet evaluate a selected provider/model and its
credentials through one deterministic, replayable admission contract.

The change must preserve Baymax's library-first boundary. Catalog data is
provided by configuration or the embedding host; it is not fetched from
provider APIs. Credentials remain host-owned and are never serialized into
config, diagnostics, replay fixtures, or events.

## Goals / Non-Goals

**Goals:**

- Define a bounded, host-injected catalog of provider/model descriptors and
  catalog versions.
- Reuse `adapter/capability` for required/optional negotiation rather than
  defining a model-specific capability taxonomy.
- Provide deterministic credential-preflight evidence and map the aggregate
  outcome through existing readiness/admission semantics.
- Preserve config precedence, fail-fast validation, atomic reload rollback,
  Run/Stream parity, and RuntimeRecorder-only diagnostics writes.
- Make catalog, credential, fallback, and reason outcomes replayable without
  leaking tokens, endpoints, raw provider responses, or secrets.

**Non-Goals:**

- Remote model discovery, background refresh, a hosted registry, credential
  storage, OAuth/device flows, secret rotation, or multi-tenant authorization.
- Provider SDK use outside `model/<provider>`, especially in `context/*`.
- A second readiness, admission, recovery, event, or terminal-state machine.
- Automatic model switching beyond configured deterministic optional fallback.

## Decisions

### 1. Use a catalog value object instead of a global registry

A new library-owned catalog package will expose immutable descriptors keyed by
normalized `(provider, model)` and a host-chosen catalog version. Descriptors
contain only bounded metadata: provider/model identity, context window, declared
capabilities, optional compatibility labels, and declared fallback targets.

The catalog is supplied through runtime configuration or an explicit host
dependency snapshot. It is validated, sorted, copied, and then treated as
immutable for an admission evaluation. This avoids global mutable state and
makes fixture/replay inputs self-contained.

Alternative considered: provider-owned dynamic directory clients. Rejected
because availability, caching, credentials, consistency, and network failure
would create a remote control-plane dependency outside the 0.x boundary.

### 2. Separate credential evidence from credential material

The host supplies a `CredentialEvidence` result per provider, containing only a
stable status (`available|missing|invalid|unverified`) and bounded canonical
reason code. `available` permits the normal candidate; `missing` and `invalid`
block required use; `unverified` follows explicit strict/non-strict policy.
Neither raw credentials nor a validation request is represented in persisted
types.

Alternative considered: have catalog code probe providers directly. Rejected
because each probe creates network side effects, couples shared code to provider
SDKs, and risks exposing credential-bearing transport data.

### 3. Reuse capability negotiation and readiness aggregation

The evaluator builds an `adapter/capability.Request` from required and optional
model capabilities, negotiates it against the descriptor, combines it with
credential evidence, and emits a single source-owned admission projection.
Required capability gaps, unknown descriptors, and invalid/missing required
credentials are blocking. Optional gaps can become deterministic degraded
fallback only when the descriptor declares an eligible target and its capability
and credential evidence independently pass.

The manager converts catalog outcomes into canonical readiness findings and
continues using existing strict-mode escalation and primary-reason arbitration.
It does not introduce separate readiness statuses or duplicate primary-reason
fields.

### 4. Freeze admission input per Run or Stream turn

An admission stores catalog version, selected descriptor identity, capability
outcome, credential status, fallback identity, and reason codes as a bounded
snapshot. Run and Stream use the same evaluator and snapshot projection. A
catalog hot reload validates a complete candidate first and atomically swaps it
only for later admissions; an invalid reload keeps the old active catalog.

Alternative considered: resolving catalog data for every provider dispatch.
Rejected because it allows one Run to mix catalog generations and makes replay
and incident investigation ambiguous.

### 5. Keep observability additive and single-writer

`runtime/diagnostics.RunRecord` receives optional bounded catalog/preflight
fields. Event parsing maps `provider.catalog` lifecycle/admission events only
through `observability/event.RuntimeRecorder`; callers cannot mutate run records
directly. Fixtures use redacted status/reason data and verify additive nullable
defaults for older consumers.

## Risks / Trade-offs

- [Static catalog becomes stale] -> Catalog version and deterministic reload
  make staleness visible; live discovery remains deliberately out of scope.
- [Credential evidence can be inaccurate] -> Evidence records `unverified`
  distinctly and strict mode can block it; the host remains accountable for
  probing policy.
- [Fallback masks a production issue] -> Fallback requires explicit descriptor
  declaration, preserves the original reason, emits diagnostics, and cannot
  satisfy missing required capabilities.
- [Catalog expands diagnostics cardinality] -> Identities and capability/reason
  lists are normalized, deduplicated, sorted, and capped before recording.
- [Config change races with execution] -> Candidate catalogs are validated then
  atomically swapped, while in-flight admissions retain their own snapshot.

## Migration Plan

1. Add disabled-by-default `runtime.provider_catalog.*` controls and descriptor
   validation. Existing provider selection behavior remains unchanged when the
   catalog is disabled.
2. Add catalog evaluator and host-supplied credential evidence seam, then
   project findings through existing readiness and admission paths.
3. Add additive diagnostics/event parsing, replay fixtures, conformance tests,
   shell/PowerShell gate, and agent-mode documentation markers.
4. Enable the feature only when a host supplies a valid catalog; invalid startup
   configuration fails fast and invalid hot reload retains the previous catalog.

Rollback is configuration-first: disable the catalog or restore the preceding
valid runtime snapshot. The evaluator leaves no provider action side effect
before admission, so rollback does not need to cancel or rewrite active Runs.

## Example Impact Assessment

修改示例

The existing `adapter-onboarding-manifest-capability` agent-mode documentation
will receive catalog admission and credential-preflight markers before its
corresponding runtime example is changed.

## Open Questions

- The first implementation will use host-injected `CredentialEvidence`; whether
  a standard in-process verifier interface is needed should be decided from
  provider client integration tests, not assumed in this proposal.
- The initial catalog will express context window as a positive token count;
  provider-specific token accounting remains owned by `model/<provider>` and is
  not standardized here.
