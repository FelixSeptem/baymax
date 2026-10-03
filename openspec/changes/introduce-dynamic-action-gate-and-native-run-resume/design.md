## Context

The current Runner evaluates Action Gate before tool dispatch and then treats tool outcomes as model-loop input. A business adapter can therefore discover a PendingAction after a tool has run, but Baymax has no typed hand-off that pauses the source Run before the next model step. Existing Realtime interrupt/resume is cursor-based and active-run scoped; `RetryRun` and follow-up promotion deliberately create new causal Runs. See `proposal.md` for the observed Application/Bole boundary and target evidence level.

The design must preserve library-first ownership, Run/Stream equivalence, additive compatibility, `RuntimeRecorder` as the single diagnostic writer, and the existing separation between source-owned runtime state and host/application business state.

## Goals / Non-Goals

**Goals:**

- Represent a tool-produced, resumable action as a bounded opaque registration owned by the source Run.
- Pause the same Run at the tool-result boundary with a native `input_required` projection before any subsequent model or executor work.
- Restore the same Runner execution from a versioned checkpoint after one valid external decision.
- Provide deterministic idempotent, stale, mismatch, timeout, denial, and terminal-resume outcomes.
- Preserve equivalent Run and Stream semantics and provide replayable diagnostics evidence.
- Allow Application/adapter layers to retain PendingAction business payload and executor ownership.

**Non-Goals:**

- Storing or interpreting Application PendingAction bodies, approval policy, credentials, or business state in Baymax.
- Replacing or merging Realtime interrupt/resume, retry, or follow-up promotion semantics.
- Introducing a hosted checkpoint service, global action registry, provider SDK dependency, or transport server.
- Re-executing completed tools or reconstructing provider-internal hidden state that was not represented in the checkpoint contract.
- Guaranteeing cross-process resume without an explicitly configured durable checkpoint owner.

## Decisions

### 1. Use an opaque dynamic action registration at the tool-result boundary

Add an additive carrier to the tool-result/dispatch contract with a bounded opaque token or reference, action kind, resumability, and producing correlation. The carrier is a capability signal, not a business action schema. Runner validates shape and correlation but never infers an action from arbitrary `Content` or `Structured` fields.

Alternative considered: inspect structured tool output for fields such as `pending_action` by convention. Rejected because it makes business schemas part of Runner behavior and cannot guarantee stable security or replay semantics.

### 2. Pause before the next model step, not during executor dispatch

After dispatch returns tool outcomes, Runner first normalizes dynamic registrations, evaluates their gate decision, and either:

```text
tool result
  -> action registration validation
  -> dynamic gate evaluation
  -> checkpoint write
  -> input_required
```

or continues the existing model loop. A valid resumable registration is a hard boundary: no next model call, no later executor call, and no final `completed` outcome for that Run until resume/deny/timeout settles the action.

Alternative considered: allow the model to observe the PendingAction and ask for confirmation itself. Rejected because it permits additional model/executor work before the source policy has admitted the action and cannot produce a source-owned `input_required` boundary.

### 3. Store only Runner recovery metadata and opaque references

The checkpoint format is versioned and bounded. It includes:

- Run/session identity and Run/Stream mode;
- iteration and loop phase boundary;
- normalized model-request/message digest or reference;
- completed tool call/result references and ordering;
- opaque action token/reference and action registration digest;
- gate decision state and resume attempt/idempotency identity;
- checkpoint version, schema, digest, and source correlation.

The checkpoint store is injected and source-owned. In-memory operation remains valid for single-process hosts; durable operation is an explicit adapter capability. The store must support compare-and-set or equivalent idempotent admission so duplicate resumes cannot continue twice.

Alternative considered: persist the entire `RunRequest`, tool output bodies, and PendingAction payload in Runner. Rejected because it duplicates business data, increases sensitive-data exposure, and violates source ownership.

### 4. Resume through a distinct same-Run API

Introduce a resume admission path that validates Run/session/token/checkpoint digest/idempotency and then restores the saved loop state. It must not call `RetryRun`, `PromoteRuntimeFollowUp`, or synthesize a new causal Run. Existing APIs keep their current meanings:

- Realtime resume restores a valid Realtime cursor;
- retry starts a new Run after failed/canceled source state;
- follow-up promotion starts a distinct causal Run at an idle/terminal boundary;
- dynamic action resume continues the paused Run from its checkpoint.

### 5. Separate command admission from runtime outcome in the host contract

The host adapter accepts a versioned action-resume command and returns only accepted/rejected/duplicate admission. The source Runner emits the subsequent pause/resume/deny/timeout/terminal facts through existing runtime events and terminal projection. Host disconnect must not synthesize a business decision or terminal state.

### 6. Preserve additive observability and replay

New diagnostics fields are nullable/defaultable and written through `RuntimeRecorder`. Suggested bounded fields include dynamic action registration count, action token digest/reference, checkpoint version/digest, pause reason, resume attempt, resume admission result, and source correlation. Replay fixtures normalize token references and digests without exposing business payloads or invoking live components.

## Risks / Trade-offs

- [Opaque references can outlive their business action] → Require source-owned expiry/validation, checkpoint digest matching, and deterministic stale rejection.
- [Checkpoint metadata can become sensitive] → Store references/digests rather than action bodies or full provider transcripts; enforce bounded sizes and redaction rules.
- [A process restart can lose in-memory checkpoints] → Advertise resume durability as an explicit source capability; fail closed when a durable checkpoint is unavailable instead of pretending same-Run recovery exists.
- [Run/Stream implementations may drift] → Share normalization/checkpoint/resume primitives and require parity fixtures for both paths.
- [Dynamic pause can conflict with existing terminal/follow-up races] → Apply first-terminal/source-owner arbitration and reject resume after a settled terminal outcome.
- [Tool output APIs gain a new optional field] → Keep the field additive and absent by default; legacy tools continue with current behavior.

## Migration Plan

1. Add the optional dynamic action carrier and checkpoint/resume interfaces without changing existing tools or `ModelClient` implementations.
2. Implement Runner pause/resume behind explicit source capability/admission; legacy Runs without registrations retain current behavior.
3. Add host command/event projection and replay fixtures; adapters opt in only after they can provide token validation and checkpoint ownership.
4. Roll back by disabling dynamic registration/resume admission. Existing pre-dispatch Action Gate, Realtime interrupt/resume, retry, and follow-up paths remain available; no persisted business data migration is required.

## Example Impact Assessment

新增示例

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | core/runtime maintainers | docs consistency |
| components | 新增文档 | `examples/dynamic-action-resume/README.md` | runner/adapter owners | example smoke run |
| configuration | 无需文档变更（附理由） | — | runtime/config | no configuration change |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md` | contract owners | focused tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability owners | diagnostics tests |
| examples | 新增文档 | `examples/dynamic-action-resume` | example owners | `go run` |
| CLI/integration | 修改文档 | host action-resume contract | host owners | host tests |
| best practices | 修改文档 | runtime boundary docs | maintainer | docs gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | release owner | roadmap parity |
