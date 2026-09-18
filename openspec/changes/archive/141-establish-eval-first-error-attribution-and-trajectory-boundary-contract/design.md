## Context

See `proposal.md` for motivation. The current `runtime/evalcontract` surface has deterministic corpus, Badcase, experiment, feedback and continuity comparison types, but Badcase records stop at result/reproduction status and digest correlation. The design must extend that package without turning evaluation output into a runtime fact source, and must compose with the reference-only privacy boundary established by `evaluation-continuity-comparison`.

The change is deliberately the offline completion layer after the archived continuity work: continuity proves that source-owned facts survive handoff/recovery boundaries; first-error attribution explains the earliest evaluated deviation and the decision boundary at that point. Neither contract owns transcript, reasoning, memory, tool, provider, task or checkpoint bodies.

## Goals / Non-Goals

**Goals:**

- Define one deterministic representation for the earliest evaluated trajectory deviation and its source-owned evidence.
- Make acceptable/forbidden action boundaries and required evidence replayable without storing prompt, transcript or reasoning bodies.
- Add nullable associations from Badcase, experiment and feedback while preserving historical payload behavior.
- Provide a fixture-first implementation sequence, canonical drift taxonomy and mixed-fixture gate coverage.
- Treat memory retrieval/application as evaluation scenarios using existing references rather than a new memory evaluation subsystem.

**Non-Goals:**

- Modifying the runtime loop, model decisions, tool dispatch, context assembly, memory retrieval, provider adapters, terminal semantics or Run/Stream control flow.
- Automatically applying feedback to prompt, Skill, tool, policy, memory, configuration, code, tests, gates or repository state.
- Persisting raw reasoning, transcript, provider/tool/memory/workspace bodies, credentials or arbitrary metadata maps.
- Introducing a hosted evaluation service, leaderboard, transcript/artifact resolver, new configuration keys or a second source of truth.

## Decisions

1. **Add a separate attribution contract rather than extending continuity facts.** The new schema is `eval_first_error_attribution.v1`. Continuity answers whether source-owned facts remained equivalent; attribution answers where an evaluated trajectory first deviated and what boundary applied. Reusing continuity references is allowed, but inserting attribution fields into `ContinuityProjection` was rejected because it would mix state continuity with quality judgment and reopen an archived contract unnecessarily.

2. **Represent the prefix by stable identity and digest, not by transcript content.** Attribution carries corpus/Badcase/run correlation, first-error step ID and ordinal, a prefix digest and bounded source references. The full message sequence, prompt, reasoning and tool output are never copied. Storing a full trajectory was rejected because it creates retention, privacy and ownership problems and makes fixture size unbounded.

3. **Use fixed top-level kinds with bounded cause codes.** First-error kind is `decision|tool_selection|tool_input|policy|memory_retrieval|memory_application|context|provider|termination|evidence`; root-cause owner is `model|tool|policy|memory|context|provider|runtime|host|unknown`. Primary and secondary cause codes remain bounded normalized identifiers so domains can express specific causes without arbitrary payload maps. Secondary causes carry explicit rank; last-write-wins and unordered free-form explanations are rejected.

4. **Use integer confidence and an explicit recoverability enum.** Confidence is represented as basis points in the inclusive range `0..10000`; recoverability is `recoverable|non_recoverable|unknown`. Floating-point confidence was rejected because cross-platform serialization can destabilize digests. Empty recoverability is normalized to `unknown` only for explicitly compatible input; unsupported values fail fast.

5. **Model decision boundaries as reference-only sets.** Acceptable actions, forbidden actions, required evidence and safety constraints use stable kind/ID/digest references. Set-like collections are sorted canonically; an action present in both acceptable and forbidden sets is rejected, and every required-evidence reference must exist in the attribution evidence set. Free-form policy summaries were rejected because they cannot be compared deterministically.

6. **Enforce small explicit bounds before digesting.** The initial contract limits secondary causes to 16, evidence references to 32, each action/constraint set to 32, and serialized attribution size to 64 KiB. Validation rejects overflow rather than truncating it. These limits keep replay safe and reviewable while leaving enough room for multi-cause Badcases; silent truncation was rejected because it can hide the true first-error boundary.

7. **Keep associations additive and nullable.** Badcase and experiment records receive optional attribution references, and feedback may cite an attribution plus bounded evidence. Existing versions without those fields retain current normalization and aggregation behavior. Attribution drift is reported independently and cannot rewrite reproduction status, metric totals, continuity outcome or feedback approval state.

8. **Keep feedback evidence-linked but review-only.** Validation may prove that a recommendation is correlated with a reviewed attribution and evidence set, but no package consumes that recommendation as an execution instruction. Any later behavior change requires a separate OpenSpec change. An automatic prompt/Skill/tool/policy/memory optimizer was rejected as outside the library-first contract and unsafe without independent governance.

9. **Use fixture-first TDD and a shared pure evaluator.** A failing `eval_first_error_attribution.v1` fixture and focused contract tests establish the current gap first. The runtime package then supplies pure normalization/comparison logic consumed by `tool/diagnosticsreplay`; replay does not duplicate taxonomy or normalization. Live runtime integration was rejected because the approved scope is offline contract closure.

10. **Use semantic reason names without proposal identifiers.** Canonical reasons cover schema, step, kind, owner, cause, prefix, action-boundary, required-evidence, evidence, recoverability, confidence, correlation, privacy and parity drift. Code, files, fixture namespaces and diagnostics contain no proposal sequence number.

## Risks / Trade-offs

- [Risk] Evaluators may disagree on what constitutes the first error. → Mitigation: require a stable step/ordinal, prefix digest, canonical top-level kind, evidence references and rubric/corpus correlation; disagreement becomes explicit drift rather than hidden prose.
- [Risk] Cause codes may proliferate across domains. → Mitigation: keep a fixed top-level kind/owner taxonomy, bounded normalized cause identifiers and contract tests for unknown/invalid values; do not accept arbitrary maps.
- [Risk] Confidence can look more objective than the evidence supports. → Mitigation: treat confidence only as an integer comparison field paired with evidence and reviewer context; it never drives runtime policy or automatic application.
- [Risk] Adding references to Badcase/experiment could accidentally change historical digests. → Mitigation: fields are nullable with documented defaults, historical fixtures stay in the mixed gate, and existing aggregation excludes attribution from legacy semantics unless the new version is explicitly present.
- [Risk] Memory application cases could become a parallel memory contract. → Mitigation: fixtures cite existing memory diagnostics and owner references only; no memory content, index access, lifecycle mutation or new memory source is introduced.
- [Risk] Large adversarial fixtures could consume excessive replay resources. → Mitigation: validate counts and serialized size before normalization, reject body fields and avoid resolver/network calls.

## Migration Plan

1. Add failing focused tests and a version-controlled `eval_first_error_attribution.v1` success/drift/privacy fixture that demonstrate the result-only Badcase gap.
2. Implement the pure bounded normalization/comparison contract and canonical reason taxonomy in `runtime/evalcontract`.
3. Add nullable Badcase/experiment/feedback association handling without changing historical normalization or aggregate digests.
4. Add diagnostics replay evaluation using the same contract and run mixed historical fixture coverage.
5. Wire shell and PowerShell gates, update the contract-test index and change the roadmap status only when implementation evidence is complete.

Rollback removes the new attribution parser/evaluator, nullable associations, fixture invocation and gate mapping. Historical corpus, Badcase, experiment, feedback, continuity and replay payloads require no migration and remain valid.

## Example Impact Assessment

无需示例变更（附理由）：本设计只增加离线 Eval contract、fixture、replay 和 gate，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或用户可见行为。
