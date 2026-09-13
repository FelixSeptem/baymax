## Context

See `proposal.md` for motivation. Existing evaluation, handoff, checkpoint/workspace provenance, and unified snapshot contracts are independently versioned and reference-first. The design must compose those projections without transferring ownership of bodies or restore state, and must preserve additive diagnostics and Run/Stream parity constraints.

## Goals / Non-Goals

**Goals:**

- Define one deterministic, bounded normalization and comparison path for baseline/candidate continuity projections.
- Make every compared axis explicit and produce stable drift classes suitable for contract tests and replay gates.
- Provide adapters from handoff and snapshot/checkpoint/workspace references, plus fixture-driven offline replay.
- Keep continuity associations nullable and preserve current experiment aggregation, restore policy, and source ownership.

**Non-Goals:**

- Persisting transcript, reasoning, provider response, tool output, artifact, or workspace bodies.
- Creating a transcript/artifact/workspace service, resolver, or new runtime fact source.
- Changing compaction algorithms, scheduler or recovery state machines, provider/tool execution, configuration, or examples.

## Decisions

1. **Use a fixed-axis projection instead of arbitrary maps.** A `ContinuityProjection` carries schema version, run/session identity, phase, and a bounded list of typed `ContinuityFact` references. Fixed kinds prevent unbounded schemas and make privacy and duplicate checks deterministic. An arbitrary JSON map was rejected because it permits silent facts and unstable comparison semantics.

2. **Compare normalized references, never bodies.** Normalization sorts by semantic kind and stable key, canonicalizes optional owner/version/digest values, rejects duplicate conflicting keys, and enforces item/serialized-size limits inherited from existing contracts. Body fields are rejected rather than truncated. A body-aware comparator was rejected because it would duplicate artifact/transcript ownership and create retention risk.

3. **Return structured drift entries.** Comparison returns a versioned result with stable comparison identity, pass/fail status, and ordered drift entries containing class, axis, baseline key, candidate key, and bounded details. Axis-specific classes are retained even when multiple drifts occur; no last-write-wins aggregation is used. A single digest mismatch was rejected because it cannot guide replay diagnosis.

4. **Compose through adapters at existing ownership boundaries.** Handoff adapters emit facts from objective, policy/admission, pending/next actions, and references; snapshot/checkpoint/workspace adapters emit manifest, lineage, attempt/lease, and workspace references. Adapters consume source-owned projections and do not invoke resolvers or mutate sources. This preserves the existing handoff and snapshot source-of-truth contracts.

5. **Keep experiment linkage additive and nullable.** Experiment/corpus records may carry an optional continuity comparison reference/result keyed by run/session and item, while existing metric/rubric aggregation remains unchanged. Continuity drift is reported as an independent classification rather than altering pass/fail metric semantics.

6. **Implement replay as an offline fixture evaluator.** `tool/diagnosticsreplay` parses `eval_continuity_comparison.v1`, invokes the same normalization/comparison functions, and emits deterministic JSON reason codes. It never connects to live runtime, providers, tools, Git, or artifact/transcript owners. Historical fixture parsers and output remain part of the same gate.

7. **Use additive documentation and gates.** Extend the contract-test index and roadmap candidate/status references, and add shell/PowerShell gate coverage. No `examples/agent-modes` change is needed because runtime behavior and example paths are unchanged.

## Risks / Trade-offs

- [Risk] Projection fields may drift from handoff or snapshot schemas. → Mitigation: adapter contract tests consume canonical fixtures and fail on unsupported versions or missing required references.
- [Risk] Large or adversarial fixtures could exhaust replay resources. → Mitigation: enforce bounded fact count, serialized size, fixed kinds, and deterministic early rejection.
- [Risk] Continuity drift could be mistaken for metric failure. → Mitigation: keep additive nullable linkage and separate drift classifications from experiment aggregate status.
- [Risk] Run/Stream or strict/compatible restore semantics could diverge through adapters. → Mitigation: parity fixtures compare normalized projections and assert no restore mutation or policy changes.

## Migration Plan

Ship the comparator and replay evaluator behind the versioned fixture contract. Existing callers remain unchanged because continuity association is optional. Add fixtures and gates, then update roadmap/index documentation. Rollback consists of removing the new fixture invocation and comparator package; existing corpus, handoff, snapshot, and replay contracts remain valid.

## Example Impact Assessment

无需示例变更（附理由）：the change adds offline comparison and replay contracts only; it does not alter `examples/agent-modes` runtime paths, configuration, or user-visible behavior.
