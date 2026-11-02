## Context

See proposal.md. Existing `ValidateHostResumeResult` runs in live resume and durable restore before the owner claim. It already validates authority/plan/revision/scope and bounded references. We extend this seam instead of adding another authorization callback or store.

## Goals / Non-Goals

Goal: expose original admission evidence and reject stale policy/listen scope with bounded fields. Non-goals: permission evaluation, cross-source identity mapping, business commit, checkpoint schema migration, and diagnostic body storage.

## Decisions

1. Add nullable `Provenance *HostAdmissionProvenance` to Binding and Current with `omitempty`. The three required fields when present are `AdmissionReference` (256 bytes), `PolicyRevision` (256 bytes), and `SourceScopeDigest` (128 bytes). A source without subscription uses an explicit host-owned scope digest. Both absent preserves existing v1 behavior; one absent or an invalid field rejects invalid_binding.
2. Admission references must match for any decision (binding_mismatch). Confirm additionally compares policy revision (authority_denied) and source scope (scope_drift), after existing identity and cancellation handling. Deny/timeout still require cancellation authority and ignore execution-policy/scope drift. Existing reasons avoid parallel error vocabularies.
3. Trusted paused provenance belongs to the host execution record, never to a client resume request. Native source refs retain their existing binding checks. Use the same pure validator in runtime and replay; do not serialize secrets or log callback bodies.
4. Test via JSON fixture mutation as an initial failing reproduction, then typed bounds and both runtime modes. Reopen a FileStore and fresh Engine to prove denial occurs before any resume claim. An invalid-then-valid attempt uses the same idempotency key to demonstrate preserved eligibility.

Alternatives: folding provenance into AuthorityDigest would hide policy/source scope attribution; requiring fields for all legacy hosts would break v1 without adding source evidence; a new authorization service would duplicate the host owner.

## Risks / Trade-offs

Validation prerequisite: three existing PowerShell contract scripts unconditionally override `GOCACHE`. Preserve an already configured shared cache in deployment-scope, team-mailbox and scenario-simulation gates (matching the existing main-worktree fixes) so full quality verification does not create per-worktree caches. No contract assertion or runtime configuration changes.

- A host can omit or fabricate facts → provenance is evidence only; hosts opting in must load a trusted paused record and perform fresh checks, with CAS at the actual effect boundary.
- Optional legacy mode cannot prove provenance → document its limited evidence; never infer provenance for old paused records.
- References reveal identifiers → opaque bounded references only, no bodies or credentials.

## Migration Plan

Deploy the new validator before hosts persist provenance. Populate all three fields from trusted admission records and current policy/source facts together. For rollback pause provenance-enabled resumes; retain source checkpoint and host facts instead of stripping the fields. No new configuration or QueryRuns schema.

## Example Impact Assessment

修改示例

更新 `examples/host-resume-admission`，展示可信暂停记录的 admission/policy/source-scope provenance；不新增执行入口。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/host-resume-admission.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`examples/host-resume-admission/README.md`
- 维护 owner：`core/types`、`core/runner` maintainers
- 验证证据：`pwsh -File scripts/check-host-resume-admission-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | — | 保留 source/host ownership，未引入模块依赖或新状态 owner |
| components | 修改文档 | `docs/guides/host-resume-admission.md` | core maintainers | host resume contract gate |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | 说明新增字段默认省略，无配置键变化 |
| contract/API | 修改文档 | `docs/guides/host-resume-admission.md` | core maintainers | provenance validation and restore tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | 复用有限 reason，不增加诊断写入口 |
| examples | 修改文档 | `examples/host-resume-admission/README.md` | examples maintainers | offline example and markers |
| CLI/integration | 无需文档变更（附理由） | — | — | 既有 direct/Host guard 复用同一校验，不新增 wire command |
| best practices | 修改文档 | `docs/guides/host-resume-admission.md` | core maintainers | trusted records and effect CAS notes |
| roadmap | 修改文档 | `README.md`、`docs/development-roadmap.md` | documentation maintainers | roadmap status consistency |
