## Context

The proposal introduces an offline audit capability over metadata already owned by extension lifecycle, adapter manifests, Skill loading, memory lifecycle, context assembly, and evaluation corpus contracts. Those owners expose related identity and version facts, but no shared runtime source of truth should be introduced. The projection therefore needs to be reference-only, bounded, deterministic, and usable by replay and gates without loading provider SDKs or changing Run/Stream execution.

## Goals / Non-Goals

**Goals:**

- Define one versioned `capability_asset_provenance.v1` projection for asset identity, version/digest, owner, scope, source, dependencies, consumers, and verification metadata.
- Normalize equivalent inputs into one stable digest and deterministic ordering so drift is evidence-based rather than serialization-based.
- Classify malformed metadata, identity/version drift, scope violations, dependency conflicts, consumer-reference conflicts, withdrawal impact, and replacement incompatibility.
- Provide bounded fixtures, offline replay, Run/Stream parity assertions, and shell/PowerShell gate parity.
- Keep all outputs reference-only and safe for historical payloads that omit new optional fields.

**Non-Goals:**

- No runtime asset registry, marketplace, dynamic download, hosted persistence, release controller, or automatic activation/rollback.
- No replacement of extension, adapter, memory, context, Skill, evaluation, or `RuntimeRecorder` ownership.
- No raw prompt, transcript, reasoning, credential, workspace content, or unbounded payload in a projection or fixture.
- No changes to `examples/agent-modes` in this first audit phase.

## Decisions

### Versioned projection, not a shared fact store

The audit consumes owner-specific descriptors and emits a versioned projection with a bounded asset kind, stable identity, optional version, version range, digest and compatible digest set, owner, scope, source reference, dependency references, consumer references, dependent references, and verification timestamp. Normalized references contain identifiers/digests only. A projection digest covers canonical fields and sorted reference arrays.

An in-memory or file-backed registry was rejected because it would create a second authoritative source and introduce lifecycle, deletion, and migration semantics that the roadmap explicitly defers. Existing owner records remain authoritative.

### Deterministic canonicalization

Inputs are validated before normalization. Asset kinds, identifier syntax, scope shape, reference limits, and digest formats use explicit bounded rules. Identifier-bearing fields accept only ASCII letters, digits, dot, underscore, colon, slash, and hyphen. Optional fields use documented null/default values; unknown additive fields are ignored. Arrays of dependencies and consumers are deduplicated and sorted by canonical identity before digesting. Repeated primary descriptors with the same identity and scope but incompatible owner or dependency data are all marked conflicting; no case is selected as authoritative.

This is preferred over preserving source order because source ordering differs between manifest producers and would create false drift. The canonicalizer must never silently repair an invalid required field.

### Separate audit findings from release decisions

The audit returns findings and bounded impact sets, including candidate replacement compatibility, but never activates, withdraws, or rewrites an asset. Withdrawal impact is computed from declared consumer references and dependency edges only; missing references produce an explicit incomplete-evidence finding rather than an inferred consumer set.

Cases that require an observed descriptor set `require_observed=true`; an omitted descriptor then emits the bounded `capability_asset_missing_evidence` finding. Drift findings carry normalized, bounded expected and observed descriptors so evidence remains auditable without storing bodies. Cross-scope references are retained only long enough to classify `capability_asset_scope_violation`, then excluded from the authorized projection and impact set.

Withdrawal impact is the bounded, sorted union of declared consumers and dependents; missing either required edge set produces incomplete evidence. This keeps policy and activation with existing owners and makes rollback a repository change that can remove the projection/replay/gate without data migration.

### Replay-first verification

Fixtures cover valid assets, missing/nullable metadata, duplicate references, digest/version drift, scope mismatch, dependency conflict, consumer conflict, withdrawal with complete and incomplete references, and compatible/incompatible replacement. Replay runs the same normalization and audit path repeatedly and compares stable output digests. Equivalent Run and Stream input envelopes must produce equivalent audit results even though the capability itself is offline.

Shell and PowerShell wrappers invoke the same deterministic contract command and fail on output or classification drift. No network or live provider is used.

## Risks / Trade-offs

- [Risk] Existing owners may expose different identity/version vocabularies. → Use explicit source adapters and classify unmappable inputs; do not silently invent identity.
- [Risk] A bounded projection can omit a real consumer or dependency. → Enforce reference limits and emit `evidence_incomplete`/impact-incomplete findings; never claim a complete impact set without complete references.
- [Risk] Canonicalization rules become another compatibility surface. → Version the projection, fixture the rules, preserve unknown additive fields, and require replay/gate coverage for any rule change.
- [Risk] Teams may treat audit output as an activation decision. → Keep output names and documentation reference-only, and test that no runtime or policy mutation is performed.
- [Risk] The full repository gate is expensive. → Add a focused contract gate and replay fixture first, then wire it into the existing quality gate with bounded inputs and no network dependency.

## Migration Plan

1. Add the new capability spec, projection/replay fixtures, focused tests, and shell/PowerShell gate without changing runtime owners.
2. Run focused contract/replay checks, then the repository quality and documentation gates required by the change process.
3. Publish the projection as an audit artifact only after deterministic replay and parity checks pass.
4. Roll back by removing the new projection adapters, fixtures, replay entry points, gate wiring, and documentation; no persisted runtime data or migration is required.

## Open Questions

None. Projection version, bounded fields, finding categories, ownership boundaries, and rollback behavior are fixed by this design; implementation may choose local Go type names without changing the contract.
