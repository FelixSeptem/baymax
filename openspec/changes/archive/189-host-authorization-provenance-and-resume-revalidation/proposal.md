## Why

Host resume already checks current authority, plan, revision and write scope, but it cannot distinguish a changed admission policy or source listen scope from an unchanged authority digest. Hosts need reference-only evidence linking a paused execution to its original admission and fresh provenance checks before source claim.

## What Changes

- Add optional trusted admission provenance (admission reference, policy revision, source scope digest) to paused binding and current facts.
- Fail closed on partial or malformed provenance and reject confirm drift before source claim or continuation.
- Preserve legacy omitted provenance and cancellation semantics; actual host authorization and effect CAS remain mandatory.
- Extend unit, Run/Stream, reopened-store and offline replay coverage using the existing host resume gate.

## Capabilities

### New Capabilities

### Modified Capabilities

- `host-resume-admission-revalidation`: bounded optional provenance and confirm-time drift checks.

## Impact

Owner: `core/types` and `core/runner`. Deployment remains embedded host and the existing supported source profiles; no new store, global authorization service, config key, terminal owner or diagnostics writer. Rollback stops provenance-enabled admission until the host can revalidate; do not remove trusted paused records or replay effects.

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
