## Why

Archive 172 makes workflow graph continuation source-owned and durable, but
child/fanout, team parallel/vote, mailbox host-plan continuation, and planbridge
resumed execution still reject or force a host to create a new logical run. The
existing scheduler recovery snapshot and mailbox lifecycle history are not
continuation material: neither can prove an atomic resume claim, restore the
original graph, or prevent duplicate effects.

This is the next roadmap candidate because the remaining composite forms have
distinct source owners and must either provide their own durable facts or remain
explicitly unavailable. The change closes those gaps source by source without
turning planbridge, history replay, or a host projection into another executor.

## What Changes

- Add bounded source-owned continuation records and restore APIs for the first
  source-owner phase: composer local/A2A child and fanout execution.
- Preserve original graph/run/session/parent/causation/authority/occurrence/
  effect identities, source-native task/member/message/attempt/lease facts,
  pending and terminal state, budgets/deadlines, completed effects, pause
  boundaries, and predecessor/successor lineage.
- Require each source to use an atomic, full-identity resume claim; equivalent
  duplicates return the retained projection while foreign, stale, conflicting,
  expired, corrupt, unavailable, or in-progress requests fail closed.
- Publish form-by-form capability evidence for the child phase. Team
  parallel/vote, mailbox host-plan continuation, and planbridge `resumed` remain
  explicit unsupported cells in this phase; their later source-owner phases
  must provide compatible bindings before they can be enabled.
- Add reference-only replay fixtures, Run/Stream parity/integration coverage,
  source-owner tests, and shell/PowerShell contract gates for every form that is
  actually enabled. A form or pause reason that lacks evidence remains
  `unsupported_capability`.

## Capabilities

### New Capabilities

- `composite-continuation-restore`: source-owned child/fanout continuation
  state, atomic restore admission, lifecycle races, and explicit unsupported
  boundaries for the remaining composite forms.

### Modified Capabilities

- `durable-native-continuation-restore`: extend durable restore requirements
  from direct/workflow to independently evidenced composite owners.
- `workflow-graph-composability`: describe graph/member/task continuation
  lineage without allowing workflow capability to imply other forms.
- `host-runtime-capability-manifest`: publish independently evidenced
  child/fanout, team, mailbox, and resumed-plan continuation cells.
- `host-execution-plan-bridge`: retain strict rejection of resumed plans in this
  phase and define the source-binding boundary needed by a later resumed owner.
- `diagnostics-replay-tooling`: validate bounded composite continuation facts
  read-only, including source lifecycle races and unsupported forms.

## Impact

Affected owners are `orchestration/composer` and `orchestration/scheduler`, with
capability/replay/planbridge rejection surfaces updated for teams, mailbox, and
resumed plans. Source-specific stores, capability projections, replay fixtures,
gates, integration tests, and public integration guidance are updated together.
No distributed store, multi-writer claim, second scheduler, mailbox owner,
terminal reducer, host business store, provider dependency, or
application-level replay restore is introduced. The initial deployment profile
remains same-host, single-writer, sequential reopen.

## Example Impact Assessment

修改示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/composer/README.md`、`orchestration/planbridge/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：composer/scheduler、planbridge、diagnostics/replay、documentation maintainers
- 验证证据：`openspec validate restore-composite-continuations-and-graph-state --strict`；`pwsh -File scripts/check-openspec-documentation-impact.ps1`；`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| 领域 | 结论 | 维护路径 | owner | 验证证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | composer/scheduler | documentation impact gate |
| components | 修改文档 | `orchestration/composer/README.md` | composer/scheduler | package contract tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | composer/scheduler | documentation impact gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | composer/planbridge | strict OpenSpec validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay | replay gate |
| examples | 修改文档 | `examples/durable-continuation-restore/README.md` | documentation maintainers | example impact gate |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | planbridge | contract gate |
| best practices | 修改文档 | `docs/guides/integration.md` | documentation maintainers | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status gate |
