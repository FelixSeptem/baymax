## Why

Direct native continuation 已有 source-owned checkpoint、atomic claim 与跨进程 restore，但 workflow 仍只能 terminal-only；宿主遇到输入、审批或预算暂停时只能重跑 parent 或把 application snapshot 当作恢复。这个缺口会丢失 graph、step、effect 和因果身份，且让 planner 无法区分真实 workflow restore 与重新执行。

## What Changes

- 为 workflow execution form 增加 source-owned graph continuation checkpoint 与 restore contract。
- 持久化 graph identity/version/digest、parent/run/session/causation、step dependency、pending/completed step、预算、pause boundary、completed effect references、terminal arbitration 以及 checkpoint predecessor/successor。
- 支持 `input_required`、`approval_required`、`budget_exhausted` 三类 source safe point；恢复同一 native workflow graph，不重跑已完成 step/effect。
- 提供新进程可调用的 restore/resume 入口，验证 binding、digest、schema、expiry、ownership、revision 与 capability subset。
- 对 checkpoint resume 执行 atomic claim；重复 resume 返回同一 projection，foreign/stale/conflicting/expired/corrupt checkpoint fail closed。
- 固定 Run/Stream 对等、cancel/timeout/retry/resume 竞争及 crash-window evidence，并增加 replay/contract gates。
- capability manifest 仅开放 workflow 对实际交付的格子；child/fanout、team parallel/vote、mailbox host plan 和 planbridge resumed 继续返回 `unsupported_capability`。
- 明确存储边界为 same-host single-writer/sequential reopen；不宣称多写者 CAS、跨机器协调、分布式 journal 或多实例 claim。

## Capabilities

### New Capabilities

无。本 change 是对现有 source-owned continuation、workflow graph、capability manifest 与 replay contract 的增量修改。

### Modified Capabilities

- `durable-native-continuation-restore`: 将 workflow 从 composite unsupported 扩展为受限的 source-owned graph checkpoint/restore。
- `workflow-deterministic-dsl`: 为 expanded workflow graph 定义三类 pause safe point、跨进程 restore 与同一 graph 的 effect 去重。
- `workflow-graph-composability`: 保存并恢复 dependency/member/step/aggregate continuation material，而非 application-level pause。
- `host-runtime-capability-manifest`: 增加 workflow checkpoint、restore 和 pause reason capability cells，并保持其它 composite form unsupported。
- `diagnostics-replay-tooling`: 增加 workflow graph continuation、claim、crash-window 与 no-side-effect replay evidence。

## Impact

涉及 workflow source owner、checkpoint store/restore API、capability manifest、diagnostic replay fixtures、integration/conformance tests、workflow README、integration and module-boundary documentation，以及 durable continuation 离线示例。不得新建 scheduler、mailbox owner 或通用 composite executor；默认持久化实现继续受 same-host single-writer 限制。

## Public Semantic Documentation Commitment


- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/workflow/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：workflow/runtime、diagnostics/replay、documentation maintainers
- 验证证据：`openspec validate restore-workflow-continuations --strict`；`pwsh -File scripts/check-openspec-documentation-impact.ps1`；`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| 领域 | 结论 | 维护路径 | owner | 验证证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | workflow/runtime | workflow owner boundary review |
| components | 修改文档 | `orchestration/workflow/README.md` | workflow/runtime | package contract review |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | workflow/runtime | configuration boundary review |
| contract/API | 修改文档 | `docs/guides/integration.md` | workflow/runtime | OpenSpec strict validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay | replay gate mapping review |
| examples | 修改文档 | `examples/durable-continuation-restore/README.md` | documentation maintainers | expected-marker review |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | workflow/runtime | admission boundary review |
| best practices | 修改文档 | `docs/guides/integration.md` | documentation maintainers | restore safety guidance review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status consistency gate |

## Example Impact Assessment

修改示例
