## Context

See `proposal.md` for motivation and the delta specs for observable behavior. Baymax already has a Context Assembler, prefix/journal lifecycle, read-only `budget_projection.v1`, bounded `handoff.v1`, reference-first restore, host-plan identity and host-resume revalidation. Those pieces deliberately have separate owners: the assembler does not own business access decisions; budget projection has no write-back path; handoff resolves existing owner references instead of storing source bodies; and resume identity/digest is not current authorization.

The integration observation is credible enough to require a contract, but the Danny source implementation has not been independently audited. The first implementation task must therefore establish the Baymax gap with a minimal synthetic fixture before adding runtime behavior.

## Goals / Non-Goals

**Goals:**

- Give hosts one versioned, provider-neutral, reference-only descriptor for already-filtered context input.
- Validate descriptor identity, bounds, canonical digest, host-plan correlation and capability narrowing before any resolver/assembler side effect.
- Reuse existing assembler, budget, compaction, handoff and runner owner paths; preserve direct legacy use when the bridge is absent.
- Carry only bounded correlation through handoff and require current host authorization again before restore injection.
- Make privacy, unsupported inputs, drift and Run/Stream parity replayable and gateable on both platforms.

**Non-Goals:**

- Moving role policy, source trust, private memory, knowledge access, business revision, write scope, effect key, commit gate or application execution record into Baymax.
- Persisting source bodies, creating a context database/cache, building a global projection registry, or adding a Provider SDK to `context/*`.
- Adding configuration keys, changing budget facts/defaults, QueryRuns fields, host wire commands, outbox/mailbox delivery, retry, new Run creation, dynamic-action continuation, or a cross-owner terminal/recovery coordinator.
- Treating a descriptor, handoff, plan digest or replay transcript as current business authorization.

## Decisions

### 1. Use a thin, explicit, opt-in reference-only projection boundary

The bridge will define a pure domain DTO and validator, provisionally under `context/hostprojection`, with `host_context_projection.v1`, immutable identity/version/digest, opaque references, visibility declarations, capability digest/subset declaration and optional host-plan correlation. Its default limits are 32 KiB serialized descriptor, 64 references, 256 UTF-8 bytes per reference and 128 UTF-8 bytes per digest. It copies mutable input before retaining it and canonicalizes ordered fields deterministically. These are constructor/API limits only; there is no env/file/hot-update surface.

The host-facing integration will pass an optional detached snapshot into the existing context assembly request. Absence preserves the current assembler behavior. A host-owned resolver is called only after validation and returns material for the current assembly step; neither the descriptor nor the bridge persists resolved material.

Alternatives considered:

- Pass host-prepared content directly through existing `ModelRequest`: minimal code but no verifiable identity, drift proof, source provenance, child subset check or replay contract.
- Store host context bodies in a new Baymax context database: appears convenient for restore but transfers source/authorization ownership and creates a parallel recovery system.
- Put the contract in `orchestration/planbridge`: plan identity may be referenced, but a context projection applies to direct paths too and must not make plan compilation an authorization prerequisite.

### 2. Validate before resolve; preserve host access ownership at resolve and restore time

The required validation order is: descriptor structure/version/limits; canonical digest; detached identity and optional plan correlation; parent-child capability subset; current source resolution; existing assembly. The resolver receives only a validated descriptor snapshot and remains responsible for live source trust and business authorization. It must not be invoked for malformed, drifted or unauthorized descriptors.

For restore, the order is existing handoff validation and reference resolution, then projection-correlation validation and host resolver reauthorization, then the normal reference-first assembler injection. The implementation must not call host code under a source/assembler lock, cache an accepted authorization decision, or turn resolver output into durable state. Errors are mapped to a finite `host_context.*` taxonomy without raw callback errors or source contents.

Alternatives considered:

- Resolve before digest/limit validation: gives attackers an oracle and violates no-side-effect-before-rejection.
- Trust the descriptor digest during restore: does not detect revoked access or changed business revision.
- Reuse the host-resume verifier: its checkpoint decision semantics and effects are intentionally different; only its principle of current host revalidation is reused.

### 3. Reuse existing budget, handoff, event and terminal owners

The bridge reads a supplied existing `budget_projection.v1` or its existing source facts but adds no counters, write-back, admission cache or terminal decision. Existing assembler pressure, prefix/journal and compaction choose their normal behavior. The optional plan identity is correlated as an immutable evidence reference only; it cannot dispatch, recompile or resume a plan.

`handoff.v1` receives an additive nullable projection-correlation value containing only schema/identity/version/digest, capability digest, optional plan correlation and opaque reference evidence. Existing handoff source reference ownership, stable cuts, idempotent restore store and RuntimeRecorder path remain in force. No new terminal/event type, global sequence, durable cursor, QueryRuns field or direct diagnostic writer is introduced.

Alternatives considered:

- Add a context budget owned by the bridge: duplicates `budget_projection.v1` and could change terminal semantics.
- Copy projection bodies into `handoff.v1`: violates reference-first privacy and creates a competing source store.
- Create projection-specific completion/restore events: duplicates existing source event and terminal ownership.

### 4. Conformance is fixture-first and capability-conservative

Implementation begins by encoding one valid direct host path and negative fixtures for malformed input, resolver denial, digest/reference drift, child widening, budget insufficiency, prefix mismatch and unauthorized restore. Only after this audit proves an extension is necessary will the bridge be attached to the minimal current owner entry point. The initial support matrix is direct Run/Stream context assembly and legal existing handoff/restore paths; cross-process continuation, composed mailbox/workflow/team recovery, resolver-independent restore and dynamic action resume remain unsupported or outside this contract.

Offline replay normalizes descriptor evidence, budget facts, handoff correlation, restore result and Run/Stream result. It never calls a resolver/provider/tool/store. A dedicated shell/PowerShell gate runs targeted tests, replay, synthetic privacy checks and deliberate drift rejection; it is connected to the standard quality gate only after the suite exists.

## Risks / Trade-offs

- [Hosts must supply a resolver rather than raw durable bodies] → This intentionally adds an integration boundary, but preserves source trust, revocation and privacy ownership.
- [A valid descriptor may later fail source resolution] → Fail closed before partial injection; hosts can refresh/filter/retry through their own application path.
- [Canonical digest/bounds can reject currently ad-hoc host inputs] → Explicit opt-in plus exact-boundary fixtures and documented migration/rollback keep legacy direct assembler paths unchanged.
- [Authorization can change after resolution but before a host business write] → This contract controls context admission only; the host retains its own effect/revision CAS at the business commit boundary.
- [Handoff correlation expands public fields] → Additive/nullable/defaultable representation, no source body, and compatibility documentation protect existing consumers.
- [Integration observation may not match a real host implementation] → The audit and fixture task is a mandatory admission checkpoint; no speculative state machine is added if the gap is not reproduced.

## Migration Plan

1. Publish the OpenSpec, active roadmap entry, documentation baseline and offline example README before implementation tasks are checked.
2. Implement pure descriptor/validator/resolver interfaces and golden vectors behind explicit opt-in; keep legacy assembler callers unchanged.
3. Add minimal assembler/handoff/restore integration only after a real Baymax gap fixture requires it; retain source-owned budget and event paths.
4. Add real-runner Run/Stream integration, reference-only replay, dual-platform gate and documentation/test-index evidence.
5. Roll back by ceasing projection submission or disabling its explicit constructor option. Do not delete checkpoint/journal/application records or claim rollback of already-committed business effects.

## Example Impact Assessment

新增示例

新增 `examples/host-context-assembly`。README 将在任何示例代码之前定义 semantic anchor、runtime path、支持/拒绝边界、expected markers 与 rollback notes；示例采用离线 fake model/tool 和 synthetic opaque references，不接入 agent-modes、live provider 或业务数据库。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`context/README.md`、`context/handoff/README.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/guides/integration.md`、`docs/mainline-contract-test-index.md`、`examples/host-context-assembly/README.md`
- 维护 owner：context、runner、host integration 与 documentation maintainers
- 验证证据：OpenSpec strict validation、documentation/example impact、roadmap/docs consistency；实施后补齐 context contract/replay、真实 runner Run/Stream parity、双平台 gate 与完整 Go/race/lint/quality/docs 门禁的实际命令和退出结果

“已同步”指本 change 已在同一功能分支建立 active proposal 的公开语义、owner、范围、文档与示例基线；它不表示运行时代码已经交付。最终 API、limits、错误、支持矩阵与实现证据仍须在任务 6 中更新并以 `verification.md` 的实际命令结果为准。没有上述维护路径的实际 Git diff、对应测试与门禁证据，任务不得勾选、change 不得归档。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md`、`context/README.md` | context maintainers | bridge 不拥有 source body、业务授权、预算或恢复状态；`context/*` 不引入 Provider SDK |
| components | 修改文档 | `context/handoff/README.md`、实施新增 bridge README | context/runner maintainers | projection → assembler → handoff/restore owner 映射与实际 API 对照 |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config maintainers | 明确 explicit opt-in、constructor-only limits、无新 env/file/hot-update 键与现有优先级不变 |
| contract/API | 修改文档 | `docs/guides/integration.md`、`docs/mainline-contract-test-index.md` | host integration/contract maintainers | spec requirement、正负边界/integration/replay/gate 逐项映射 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | bounded opaque reference/digest、RuntimeRecorder 单写、不新增 QueryRuns 字段或业务正文 |
| examples | 新增文档 | `examples/host-context-assembly/README.md`、`README.md` | examples maintainers | 文档基线先行、离线 smoke、expected markers 与 rollback 验证 |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host integration maintainers | explicit opt-in、宿主 resolver 与无新增 wire command 的 active-proposal 边界 |
| best practices | 无需文档变更（附理由） | `docs/guides/best-practices.md` | documentation maintainers | 该 contract 的集成 guardrails 已由 integration/context 文档覆盖；尚未增加通用操作建议或独立 best-practice 语义 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | active slug 与 `openspec list --json` 状态一致，归档后同步索引 |
