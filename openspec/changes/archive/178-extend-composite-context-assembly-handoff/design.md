## Context

See `proposal.md` for the problem and scope. Existing direct projection admission/resolution is in `context/hostprojection`; direct assembly and handoff correlation are integrated through runner/context assembler. Composite execution and restore are owned independently by Composer child, teams, workflow, and mailbox. Those owners now have continuation envelopes, but the public context projection is not bound to their exact native node or continuation lineage. Deployment remains same-host, single-writer, sequential reopen.

## Goals / Non-Goals

**Goals:**

- Preserve one descriptor/resolver contract while binding its use to the native source owner and exact node/continuation identity.
- Make all correlation validation precede resolver invocation, checkpoint claim, member/step activation, message delivery mutation, and context injection.
- Declare support independently by source form and deployment evidence; allow an owner to remain unsupported without blocking other independently supported owners.
- Keep source bodies transient and authorization current at each resolve/restore boundary.

**Non-Goals:**

- A shared composite context store, context cache, source journal, new graph executor, host-side continuation state machine, or planbridge resume loop.
- Automatic projection inheritance to sibling nodes or replacement deliveries.
- Multi-writer/distributed storage, new authorization model, new budget ledger, or business commit integration.
- Promoting composite context restore for any owner without a working source binding and fresh-process evidence.

## Decisions

### Code-backed baseline and owner-specific restore boundaries

The existing projection contract is implemented only on the direct runner assembly path. `types.RunRequest` and `types.ContextAssembleRequest` accept the opt-in descriptor/resolver, while `handoff.HostContextProjectionCorrelation` currently preserves the projection descriptor but has no composite source/node/continuation identity. The direct handoff restore path validates existing handoff/source references and resolves current host context, but that does not bind the result to a composite continuation.

| Form | Existing source-owned state and safe point | Projection status before this change | Restore constraint |
| --- | --- | --- | --- |
| direct runner | request-scoped projection at the existing assembler admission point; runner native checkpoint restore is a separate path | direct Run/Stream assembly and handoff correlation exist; native continuation explicitly rejects requests containing a host projection | do not treat direct projection as checkpoint-bound context restore |
| child/fanout | Composer child task/attempt continuation, persisted through `ChildContinuationStore`; resume delegates to the existing child continuation executor | no projection field or resolver in child continuation/resume APIs | task/attempt and lease identity must be validated before any future projection resolution or claim |
| team parallel/vote | team continuation envelope retains graph/member state and resumes through the team owner | existing owner path now binds/resolves context only for pending local members; remote context injection remains unsupported | bind a single member and graph generation; do not share material with siblings or reactivate settled members |
| workflow | workflow continuation retains step/node state and resumes through the workflow engine | no projection field or resolver in workflow continuation/resume APIs | bind an exact workflow step and checkpoint generation before resuming that graph |
| mailbox host plan | mailbox continuation retains message/delivery/lease/attempt and `ResumeHostPlanContinuation` returns the existing message outcome | no projection field, resolver, or handler execution safe point; resume does not execute the host effect | context injection remains unsupported; lifecycle outcome projection is not context assembly or execution restore, and lease loss must remain owner-decided |

The audit exercised the current focused suites for `context/hostprojection`, `context/handoff`, `context/assembler`, `core/runner`, Composer, teams, workflow, mailbox, and planbridge. All passed. Relevant source evidence includes `core/runner/native_continuation.go` (rejects host-projection requests for native continuation), the owner continuation definitions in `orchestration/{composer,teams,workflow,mailbox}`, and the projection integration points in `context/assembler` and `context/handoff`. Until an owner has a non-consuming continuation validation/read phase followed by current resolver authorization and its existing atomic source claim/restore boundary, its composite context cell remains unsupported; the proposal does not assume a resolver-before-claim order that the current APIs cannot provide.

### Bind at the existing source owner boundary

Each supported source owner accepts an optional detached host-projection binding in its existing request/checkpoint API and validates it against its native continuation envelope. The owner owns node scoping: Composer child attempt/task, team member, and workflow step. Mailbox remains unsupported until its existing worker/delivery owner exposes a context-aware safe point bound to message/delivery/lease/attempt; `ResumeHostPlanContinuation` is not that safe point because it only projects lifecycle outcome. Context packages provide bounded normalization and current resolver behavior but do not import orchestration packages. This avoids a dependency cycle and keeps restore arbitration source-owned.

Alternatives considered: a planbridge-level projection router would couple context semantics to one optional integration surface and could not cover direct owner APIs. A global context registry would create a second persistence owner and blur authorization boundaries. Both are rejected.

### Extend descriptors additively and make inheritance explicit

Keep `host_context_projection.v1` compatible for direct callers. Add an optional source binding/correlation object with a declared form, opaque source identity, native node identity, continuation reference/generation, and parent projection/capability evidence. Existing descriptor limits remain fixed; any serialized additions count against the existing byte limit. No source references or resolved bodies are copied into a checkpoint except bounded opaque references/digests.

Each owner must explicitly supply or persist a binding for a target node. No default inheritance across workflow steps, team members, child attempts, or mailbox deliveries is allowed. A successor may reuse a projection only if the owner writes a successor binding into its next checkpoint and the host reauthorizes on resolution.

The binding uses the owner-preexisting logical node/attempt identity and generation, not a self-referential hash of the checkpoint envelope. Existing checkpoints derive their digest from all persisted fields, including the projection binding; embedding that derived digest back into the same binding would form a circular hash. The owner therefore persists the projection correlation together with a stable native node/attempt or delivery identity, validates the actual checkpoint reference/digest separately at its existing load/claim boundary, and then requires both validations before resolver invocation. This retains checkpoint-digest coverage without a second ID allocator or an unverifiable fixed-point hash.

Alternatives considered: inheriting a parent projection to all descendants is ergonomic but widens visibility implicitly and breaks per-node authorization. Requiring explicit node bindings is selected.

### Restore ordering is source validation, projection validation, host reauthorization, then assembly

The order is: validate manifest and source availability; load and validate native continuation/checkpoint without consuming it; match projection identity/digest/capability/source-node binding; revalidate current host authorization through the resolver; atomically claim/restore according to the existing source contract; assemble transient context at the owner's existing safe point. Team resolution is limited to pending local members and rechecks the original deadline after the resolver. If existing owner APIs require claim before resolver, change them to use a non-consuming validation/read phase first; claim is permitted only after projection and current authorization checks. Mailbox context stays unsupported until its actual handler execution safe point participates in this order. Failures must leave no partially injected context and must preserve duplicate/idempotent source outcomes.

Cancellation, timeout, lease loss, retry, and terminal arbitration continue to be decided by each source owner. Projection rejection never changes those facts and never triggers retry or a replacement Run.

### Capability promotion is owner-local and evidence-backed

Extend the versioned manifest with a context assembly/handoff cell for each form, including owner namespace, supported native node scope, projection/continuation API versions, deployment, retention/expiry, resolver requirement, and evidence reference. Direct support remains unchanged. Composite cells stay unsupported until their own tests prove context binding, current authorization, restart restore where claimed, failure ordering, and Run/Stream parity. Planbridge can validate and delegate to these cells but cannot promote them based on type/package presence.

Alternatives considered: a single `composite=true` bit would permit capability borrowing and cannot explain which graph node is bound. Per-form/per-owner cells are selected, with unsupported dimensions explicit.

### Replay remains fixture-only and read-only

Extend versioned fixtures with synthetic identities/digests and expected ordering/classifications. Replay must not call real resolver, source store, model, tool, Run/Stream, checkpoint claim, or terminal mutation. Real integration tests separately prove owner behavior with fake model/tool and fresh-process source stores.

## Risks / Trade-offs

- [Risk] Source API changes across four owners are broad → implement adapters incrementally but keep one shared correlation DTO and common conformance vectors; unsupported cells remain explicit until each owner's tests pass.
- [Risk] Resolver callback before source claim creates a time-of-check/time-of-use window → source owner rechecks generation/fence at atomic claim after resolver, and refuses if native facts changed; no execution begins on mismatch.
- [Risk] Context fragments could appear in serialized state → persisted types accept only descriptor/correlation references, with tests scanning checkpoints, handoffs, replay fixtures, diagnostics, and errors for source-body sentinels.
- [Risk] Existing direct behavior could regress → retain no-projection compatibility, add direct regression parity, and use additive/nullable fields with bounded defaults.
- [Risk] Some composite owner may not safely support context at its current pause boundary → report `unsupported_capability`/unavailable and do not invent fallback behavior. Mailbox lifecycle continuation currently has no handler execution safe point and is excluded from context injection in this change.

## Migration Plan

1. Add contract fields and validation helpers while keeping existing direct descriptor construction source-compatible.
2. Add owner-specific optional bindings and capability cells; default them to unsupported.
3. Promote only source forms with completed integration/restart evidence and update the example and documentation in the same change.
4. Roll back by omitting the optional bindings and manifest evidence; continue using existing owner-specific execution without composite projection. Retain all persisted checkpoints and source history.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `context/hostprojection/README.md`, `context/handoff/README.md`, `orchestration/teams/README.md`, `orchestration/mailbox/README.md`, `examples/host-context-assembly/README.md`
- 维护 owner：context projection/assembler owners, each composite source owner, planbridge, documentation and examples maintainers
- 验证证据：strict OpenSpec validation, owner unit/integration/restart tests, Run/Stream parity, reference-only replay, shell/PowerShell gates, `openspec validate --all`, full Go/race/lint/quality/docs verification recorded in `verification.md`

## Example Impact Assessment

修改示例

理由：the existing offline host-context assembly example must gain an owner-bound composite scenario and a cross-node/stale-continuation rejection marker; it must not use live services or define a parallel runtime contract.

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | context and source owners | ownership/no-second-store contract tests |
| components | 修改文档 | `context/hostprojection/README.md`, `context/handoff/README.md` | context owners | projection/handoff tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config owner | fixed limits and no-new-config contract gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | context and source owners | per-form integration tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay owner | replay/gate mapping tests |
| examples | 修改文档 | `examples/host-context-assembly/README.md` | examples owner | offline smoke and marker checks |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | planbridge/source owners | pre-activation rejection tests |
| best practices | 修改文档 | `docs/guides/integration.md`, `context/hostprojection/README.md` | context and source owners | per-owner authorization, fail-closed and rollback contract tests |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | roadmap status consistency gate |
