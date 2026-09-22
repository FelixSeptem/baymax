## Context

See `proposal.md` for motivation and the modified specs for behavior. The current runner forwards `RunRequest.Input`, `RunRequest.Messages`, and pending `ToolCallOutcome` values together in `ModelRequest`. The three provider adapters then call `toolcontract.CanonicalInput`, which chooses a trimmed `Input` (falling back to only the final message when input is absent) and serializes tool results into a text envelope. Consequently Generate/Stream lose structured request semantics, while parts of CountTokens already map `Messages` differently.

The repository must preserve its layering constraints: provider SDK protocol details remain in `model/<provider>`, `context/*` cannot import provider SDKs, diagnostics remain `RuntimeRecorder`-owned, and Run/Stream cannot introduce parallel decision or termination semantics. Archive 142 already supplies bounded source/observed projections, fixture parsing, replay, classification, and SDK-boundary test seams; this change reuses rather than replaces them.

## Goals / Non-Goals

**Goals:**

- Give every supported adapter one canonical, SDK-neutral source interpretation for role ordering, final input placement, and tool-result validation, then map it independently to each provider's native SDK request types.
- Preserve source order without implicit deduplication: ordered `Messages` are emitted first; a non-empty `Input` is appended as the final user instruction. This mirrors the runner's ownership of both fields and gives callers a deterministic result when they intentionally provide both.
- Make a valid canonical `ToolCallOutcome` native at the SDK boundary, or fail before invocation if native construction is not safe or supported.
- Make Generate, Stream, and CountTokens prove equivalent projection facts through the existing conformance family and `provider_request_projection.v1` fixture/replay/gate.

**Non-Goals:**

- No cross-provider SDK request struct, no generic provider wire protocol, no change to model selection/fallback/terminal semantics, and no `context/*`, scheduler, ReAct, tail-recap, or runtime configuration change.
- No prompt-cache policy, `TokenUsage`/diagnostics cache field evolution, remote catalog, local routing, credential storage, or raw-payload observability.
- No text-envelope fallback for a valid result when an SDK-native representation is required; legacy text-only request behavior is not retained as a silent compatibility path.

## Example Impact Assessment

无需示例变更（附理由）

设计把变更限定在 provider adapter 的 SDK 请求构造、离线 projection fixture 与 gate；它不改变 agent-mode 的配置键、runtime path 或 expected markers。实现若证明例程的可观察 tool-result、usage 或输出 marker 改变，必须先完成 `examples/agent-modes/MATRIX.md` 与受影响模式 README 的文档基线，并将评估改为“修改示例”，然后才能继续相应实现任务。

## Decisions

### 1. Canonical request interpretation is SDK-neutral; SDK construction remains provider-owned

`model/toolcontract` will own a small canonical request interpretation/validation layer whose input is `types.ModelRequest`. It will preserve bounded references to source messages, final input, and valid tool-result associations, rather than serializing them. It contains no provider enum, SDK type, network call, or provider-specific part definition.

Each `model/openai`, `model/anthropic`, and `model/gemini` adapter will own a private mapper from that interpretation to its official SDK request types. This separates shared source semantics from provider protocol details without creating a shared outbound request wire format.

Alternatives considered:

- Continue using `CanonicalInput`: rejected because its string result cannot retain role or native result-part identity.
- Introduce a common provider request AST with SDK-like parts: rejected because it becomes a second provider protocol owner and invites lowest-common-denominator text fallback.
- Duplicate all source validation in every adapter: rejected because Run/Stream/CountTokens divergence would recur.

### 2. Source ordering and final input placement are explicit and immutable

The canonical sequence is: retain non-empty ordered `Messages` exactly in their source order; append non-empty `Input` once as the final user instruction; then attach valid tool results using the provider's native correlation mechanism. The layer does not deduplicate same-content items, merge roles, reorder tool outcomes, or infer a role from text. Empty/whitespace-only messages and input are excluded by the same normalization rule used for all three paths.

This is deliberately source-preserving. A caller that supplies equal content in both fields has supplied two source facts; deletion would create a hidden behavior change. Tests will pin the ordering and count so a later consolidation can only occur through an explicit contract change.

Alternatives considered:

- Prefer Input and discard Messages: rejected because it preserves the current proven loss.
- Treat the final Message as Input and deduplicate: rejected because it loses caller intent and cannot reliably identify duplication across roles.

### 3. Native tool-result representation is all-or-fail before provider invocation

The canonical layer first enforces the existing non-empty call ID and tool name rule. Per-provider mappers then construct the official SDK's native tool-result/function-response input part and retain call/name correlation. If a provider SDK exposes no safe native request representation for a result shape required by the contract, the mapper returns a deterministic request-shape classified failure before any SDK call. It never appends `[tool_result_feedback.v1]` to a user text block as fallback.

The implementation begins with a provider-SDK shape audit in tests: pin exact SDK constructors/types available at the resolved dependency versions before replacing construction. This guards dependency-version surprises without making SDK internals part of the cross-provider contract.

Alternatives considered:

- Keep text envelopes for one provider: rejected because it leaves the declared gap unresolved and makes tool ownership ambiguous.
- Drop unsupported result fields silently: rejected because it breaks success/error meaning and causal correlation.

### 4. One adapter-local request builder is called by Run, Stream, and CountTokens

Each adapter will expose only private helpers. Generate, Stream, and CountTokens call the same adapter-local builder or shared fact interpretation so they cannot independently choose different role/order rules. The builder produces native SDK values immediately before the respective SDK invocation; it does not create new runtime state, write diagnostics, or alter provider selection.

For APIs where CountTokens cannot accept a provider-native tool-result type, it may omit the unavailable transport detail only after recording an explicit projection capability in conformance tests; it must still preserve the same role order and call/name association facts used by generation. This is a bounded API-shape exception, not permission for a divergent text projection.

### 5. Fixture migration makes resolved gaps observable

The existing `provider_request_projection.v1` cases remain version-compatible. For the supported provider × Run/Stream cases that currently declare role, tool-result-native, and capability-related gaps, this change updates only the gaps it actually resolves to explicit native/no-gap expectations. Capabilities and cache availability remain unchanged unless a test proves a pre-existing change.

Replay performs a twice-run idempotency check over the updated fixtures. Boundary tests compare taxonomy constants, fixture expected shapes, documentation, and gate invocations; they also reject raw body fields. This prevents a repair from silently changing expected projection semantics without fixture and documentation review.

## Risks / Trade-offs

- [Provider SDK input models differ, especially for system and tool-result parts] → Pin each resolved SDK constructor/type in adapter-level tests before implementation; keep mapper code provider-local.
- [Appending Input after Messages could expose duplicate caller text] → Preserve it intentionally as source semantics; document and test it rather than silently suppressing a source fact.
- [Native mappings may affect generated model behavior] → The change is behaviorally intentional but constrained to request representation; use existing request fixtures and Run/Stream parity suites, and do not alter loop/fallback logic.
- [Tool-result payload compatibility differs by provider] → Enforce existing payload bounds before construction, include success/error and malformed cases, and fail fast if a required native mapping cannot be formed.
- [Fixture edits might accidentally erase evidence] → Keep fixture case IDs and digest checks stable where possible; require no-gap expectations and replay idempotency rather than deleting cases.
- [SDK upgrades change constructors] → Conformance captures operate at the existing SDK-boundary seams and will fail compilation/tests when mappings need intentional migration.

## Migration Plan

1. Add failing SDK-boundary/conformance tests proving the current role collapse, text-envelope result form, and Run/Stream/CountTokens mismatch for all providers.
2. Introduce canonical request interpretation/validation and provider-local native builders, then convert one provider at a time while retaining the full suite after each conversion.
3. Migrate the affected `provider_request_projection.v1` fixture cases from declared gaps to native/no-gap expectations; retain unresolved non-goal gaps unchanged.
4. Run replay, boundary, gate, full normal/race tests, lint, and docs consistency. Update the roadmap wording only after the implementation/gate evidence is complete.
5. Roll back by restoring the previous adapter request builders and corresponding declared-gap fixture expectations as a single revert. No data, config, or schema migration is required.
