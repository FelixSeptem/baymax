## Why

Dynamic-action continuation is currently retained in an Engine-local map, so a new Engine cannot restore the same native Run after process loss. Composite child/team/workflow/mailbox forms also have no uniform pause/restore capability contract, making it possible for callers to confuse existing checkpoint-like metadata with source-owned continuation.

This change establishes a durable direct-run baseline and an explicit form-by-pause-reason support matrix. It does not claim that composite forms are recoverable until their own source owners can prove restoration of the same graph.

## What Changes

- Add an injectable durable runner checkpoint store, versioned public checkpoint references, atomic/idempotent resume claims, and a new-Engine restore entry point for direct native Runs.
- Preserve the same logical run/session, causal predecessor, pause reason, and Run/Stream semantics across checkpoint persistence and process restart. Host revalidation remains an admission callback only; it cannot restore source state.
- Require direct support and cross-process verification for `input_required`, `approval_required`, and `budget_exhausted`. During rollout, unverified cells reject as `unsupported_capability`; completion of this direct baseline requires all three cells and cannot be claimed by leaving one unsupported.
- Declare child/fanout, team, workflow, mailbox, and planbridge resumed execution unsupported for every pause reason in this phase. Existing workflow checkpoints and event-history replay are not runner continuation restore.
- Define fail-closed behavior for corrupt, foreign, expired, unavailable, or already-claimed checkpoints and for cancellation/timeout/retry/resume races. No new parent Run, retry, replay side effect, or best-effort fallback may stand in for restore.
- Add crash-window contract fixtures, direct Run/Stream and new-Engine integration coverage, a documentation-first offline example, and dedicated contract gates.

## Capabilities

### New Capabilities
- `durable-native-continuation-restore`: Source-owned persistent direct-run checkpoints, cross-process restore, atomic resume claims, typed rejection, and explicit unsupported boundaries for composite execution forms.

### Modified Capabilities
- `dynamic-action-gate-and-native-run-resume`: Extend the existing same-Engine continuation contract with a separately versioned durable profile, preserving legacy input_required behavior.
- `host-resume-admission-revalidation`: Preserve host revalidation as a pre-admission check while adding durable source claim/restore ordering and new-Engine identity binding.
- `host-runtime-capability-manifest`: Require source-evidenced checkpoint and restore capability by execution form and pause reason; keep unverified composite cells unsupported.
- `diagnostics-replay-tooling`: Add read-only fixtures for checkpoint lifecycle and crash-window outcomes without executing a continuation during replay.
- `host-execution-plan-bridge`: Keep resumed plan execution rejected unless a matching source-native restore capability and binding are explicitly declared.

## Impact

- Public API and source state: `core/types`, `core/runner`, and host resume integration. Preserve the existing metadata-only `RunCheckpointStore` signatures; add a durable extension contract and actual Engine injection path rather than silently changing existing implementers.
- Capability projection: `orchestration/planbridge` manifest; no planbridge restore implementation or new scheduler is implied.
- Evidence: runner/type/host tests, fixtures and replay, contract scripts, documentation and an offline example.
- Existing workflow persistence, mailbox ownership, native event history, and callback adapters remain separate source contracts.
- Roadmap BXM-4 composite graph implementation remains a named follow-up candidate with its original acceptance criteria; this proposal closes the direct baseline and explicit unsupported matrix only. Danny coverage: 1.3, 4.3–4.6 (capability/rejection boundary), 5.2–5.5, 7.1–7.2.

## Example Impact Assessment

新增示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/runner/README.md`、`core/types/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：runner/types maintainers、host integration maintainers、documentation and examples maintainers
- 验证证据：`openspec validate persist-native-continuations-and-restore-composite-graphs --strict`、OpenSpec documentation/example/roadmap gates、`scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime architecture maintainers | source store and restore ownership boundaries documented |
| components | 修改文档 | `core/runner/README.md`, `core/types/README.md` | runner/types maintainers | public DTO and Engine responsibility documented as planned contract |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime diagnostics maintainers | typed checkpoint failure and diagnostic boundary documented |
| contract/API | 修改文档 | `docs/guides/integration.md` | host integration maintainers | direct restore contract and unsupported planbridge scope documented |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime diagnostics maintainers | checkpoint lifecycle facts and single-writer boundary documented |
| examples | 新增文档 | `examples/durable-continuation-restore/README.md` | examples maintainers | documentation-first offline scenario and expected markers defined |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host integration maintainers | integration preconditions and failure behavior documented |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | restore versus retry/replay distinction documented |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | active Spec 3 scope/status linked to this change |
