## Why

Explicitly injected runner, child, team, and workflow journals currently discard append failures, so execution may appear complete even when its advertised source history is incomplete. Separately, a workflow FileStore reopen converts an abandoned `claimed` resume to `recovery_required` but leaves an abandoned `running` resume stuck as `in_progress`; the store also lacks an inter-process single-writer fence despite advertising sequential single-writer scope. These gaps make recovery status ambiguous after persistence failures or process crashes.

## What Changes

- Give explicitly configured source owners a typed, source-identified health outcome for callback-history append failures. These callback projections report degraded/unavailable retained history; they do not gate effects or claim atomicity with execution.
- Keep effect uncertainty at the native continuation/source boundary: workflow claim/result persistence failures fail closed, and reopened incomplete workflow resume claims become `recovery_required` without redispatch.
- Keep history opt-in and source-owned. Do not create a second journal, claim atomicity with a consumer projection, or reinterpret callback replay as execution restore.
- On workflow store reopen, classify every incomplete persisted resume attempt (`claimed` or `running`) without a committed result as `recovery_required`; never automatically redispatch an uncertain step.
- Enforce the advertised same-host single-writer/sequential-reopen workflow FileStore scope with an owner fence and typed rejection of concurrent or stale writers.
- Preserve exact duplicate resume projection, typed conflict/unavailable outcomes, original workflow identity, and Run/Stream terminal parity.
- Move the repository's strict security-validation toolchain floor to Go `1.26.10`, the fixed version for the reachable `GO-2026-6809` standard-library advisory that blocks this change's required quality gate.
- Add deterministic failure/crash-window tests and update the public contract, workflow/store documentation, roadmap, and contract-test mapping.

## Capabilities

### New Capabilities

- `source-history-failure-semantics`: defines how source owners report failed history writes without claiming complete history or replayability.

### Modified Capabilities

- `native-execution-event-history`: source history append failures and source availability must be explicit; observation history remains distinct from effect execution and consumer projection.
- `durable-native-continuation-restore`: abandoned workflow resume attempts and workflow FileStore ownership/fencing must resolve to typed fail-closed outcomes after reopen.
- `host-runtime-capability-manifest`: the workflow FileStore may advertise its existing single-writer sequential-reopen profile only when that writer fence is enforced.

## Impact

Affected owners: `core/history`, `core/runner`, `orchestration/composer`, `orchestration/teams`, `orchestration/workflow`, and workflow continuation tests. Public behavior changes concern source-history health/failure outcomes and workflow resume recovery outcomes; no new execution form is enabled. Update `core/types/README.md`, `orchestration/workflow/README.md`, and `docs/development-roadmap.md`. Danny-side cross-store projection transactionality remains out of scope.

## Example Impact Assessment

无需示例变更（附理由）：此 change 不增加可正常执行的调用形态或配置入口；它定义写入故障与 abandoned resume 的 typed failure/recovery 行为。故障注入和重启测试比增加正常路径示例更能验证这些边界。

## Documentation Impact Assessment

| Area | Outcome | Document | Owner | Evidence |
|---|---|---|---|---|
| architecture | 修改文档 | `orchestration/workflow/README.md` | Workflow owner | 明确 incomplete claim、recovery-required 与单写者 fence 行为 |
| components | 修改文档 | `core/types/README.md` | Runtime contracts owner | 明确显式 history 注入、append failure 与 capability health 边界 |
| configuration | 修改文档 | `core/types/README.md`, `CONTRIBUTING.md`, `docs/versioning-and-compatibility.md`, `docs/runtime-config-diagnostics.md` | Runtime contracts owner | 明确本 change 不新增 runtime 配置键或默认值，并将 strict security toolchain floor 同步为 Go 1.26.10 |
| contract/API | 修改文档 | `core/types/README.md` | Runtime contracts owner | 记录 typed history failure 和 workflow recovery 语义 |
| diagnostics | 修改文档 | `core/types/README.md` | Runtime contracts owner | 明确不增加诊断 schema 或 recorder 字段 |
| examples | 修改文档 | `core/types/README.md` | Runtime contracts owner | 说明负向/崩溃窗口由测试覆盖，不增加示例 |
| CLI/integration | 无需文档变更（附理由） | — | Runtime contracts owner | 不改变 CLI 或 integration 调用流程 |
| best practices | 无需文档变更（附理由） | — | Runtime contracts owner | 由 owner package 文档说明 failure/recovery 约束，不引入新惯例 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | Runtime roadmap owner | 标记本 change 为 active 并保留剩余 source-owner gaps |
| release status snapshot | 修改文档 | `README.md` | Runtime contracts owner | 将 active OpenSpec 状态与 `openspec list --json` 对齐 |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `core/types/README.md`, `orchestration/workflow/README.md`, `CONTRIBUTING.md`, `docs/versioning-and-compatibility.md`, `docs/runtime-config-diagnostics.md`
- 维护 owner：Runtime contracts owner and Workflow owner
- 验证证据：`openspec validate --all`; `pwsh -File scripts/check-docs-consistency.ps1`; targeted history/workflow tests
