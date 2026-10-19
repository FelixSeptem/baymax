## Why

归档 162 的 `host_execution_plan.v1` 已建立 bounded compile、native-owner delegation 与 pre-dispatch rejection，但它正确地将 mailbox delivery 视为 `unsupported_capability`。Danny 所需的 `mailbox_async` 不能通过重跑 parent、direct-run fallback 或 host 自建队列来模拟；它需要复用 Baymax mailbox 的 message、delivery/lease、retry、DLQ、authority 与因果事实。

调用方也不能从 package、DTO 或过往拒绝行为推断一个 form 是否能 admission、保留 native event、创建 checkpoint、跨进程恢复或支持某个 pause reason。现在先在同一个有界 change 中发布 source-owned capability manifest，并将其中明确可用的 `mailbox_async` 计划桥接到既有 mailbox owner，为后续 source history/cursor replay 和 durable continuation 建立可靠的 admission 基线。

## What Changes

- 新增版本化、可追溯的 host runtime capability manifest/API。它按 execution form 声明 admission、native events、durable history、after-cursor replay、checkpoint creation、cross-process restore、`input_required`、`approval_required`、`budget_exhausted`、cancellation、timeout、causal retry、mailbox continuation 和 retention policy，并关联 Baymax contract version、build/module revision、public API version、fixture/contract evidence 与 unsupported reason。
- 修改 `host_execution_plan.v1` 与 `orchestration/planbridge`：当 manifest 明确允许时，接受 `mailbox_async` delivery/binding；在编译与 admission 阶段校验 mailbox configuration/contract version、message identity、parent run、causation、authority digest、plan digest、write scope、effect namespace 和 resource limits。
- 新增或扩展 mailbox 的公开 binding/outcome DTO，以 source-owned 方式暴露 message ID、delivery/lease ID、attempt/retry causation、ack/nack/retry/DLQ 结果、原始 run/parent run/host correlation 和 terminal outcome。相同逻辑消息的重复 dispatch 必须收敛为一个 logical admission，且保留原 authority、occurrence 与 effect namespace。
- planbridge 只委托现有 Baymax mailbox owner、binding 和 delivery lifecycle；不得创建 queue、worker/execution loop、scheduler、第二个 mailbox owner、terminal reducer、event journal 或 host business-commit state。
- 明确 lease loss、timeout、retry、DLQ 与 terminal/attempt evidence 的 source-native 关联。mailbox 不可用、manifest/contract version 不兼容、identity/binding 不匹配或 form 未声明时，必须在副作用前返回 typed unavailable/`unsupported_capability`，不得降级到 direct Run 或通过重新 Run parent 恢复。
- 提供 mailbox continuation 的 Run/Stream terminal parity、duplicate dispatch、lease-loss/retry/DLQ、unavailable/version mismatch 的 unit/integration/replay/gate coverage，以及离线 host-neutral example 和同步维护文档。

## Capabilities

### New Capabilities

- `host-runtime-capability-manifest`: 定义 versioned、evidence-linked、source-owned 的 execution capability document 与 admission 查询，明确 form/pause/history/restore 支持或拒绝原因，禁止调用方从实现存在性猜测能力。

### Modified Capabilities

- `host-execution-plan-bridge`: 将 mailbox delivery 从无条件拒绝改为仅在 manifest 与 native mailbox binding 都明确兼容时的 source-owner delegation，并固定 pre-dispatch validation、identity、result 与 no-fallback 合同。
- `multi-agent-mailbox-contract`: 扩展公开 mailbox binding/outcome 语义，使 host-plan 计划可引用既有 mailbox lifecycle，同时保留 message/delivery/lease/retry/DLQ 与 terminal ownership。

## Impact

预计改动 `orchestration/planbridge`、`orchestration/mailbox` 及其 integration boundary、对应公开 DTO、`tool/diagnosticsreplay`、`tool/contributioncheck`、专用 shell/PowerShell contract gate 和 `examples/host-execution-plan`。`Run`/`Stream`、既有 direct/teams/workflow/child bridge 路径和 direct mailbox API 保持兼容；它们不会因 manifest 存在而取得新的 continuation/replay/restore 语义。

宿主继续拥有业务 authorization/revision、write scope、effect namespace/key、occurrence 与 business commit。mailbox 继续拥有 publish/delivery/lease/ack/nack/requeue/retry/DLQ；source runner 继续拥有 Run/Stream、attempt 和 terminal；`RuntimeRecorder` 继续是唯一 diagnostics writer。此 change 不实现 durable source history/cursor replay、跨进程 checkpoint restore 或 team/workflow/child pause continuation；这些能力分别留给路线图 Spec 2、Spec 3，并在 manifest 中显式声明为 unsupported，直到具备独立证据。

## Risks and Rollback

主要风险是把 delivery accepted、ack 或 DLQ 错当作 source/business terminal，把不等价 duplicate admission 成第二次执行，或让 bridge 的预检替代 source 当前 lease/authority 状态。设计以 full logical identity、source activation recheck、分离 delivery/attempt/terminal facts、Run/Stream parity 和 no-side-effect rejection 来控制这些风险。

回滚时停止构造新 mailbox plan binding，并将 manifest 中 `mailbox_async` 标为 unavailable/unsupported；已有 mailbox message、delivery、lease、attempt、retry、DLQ 与宿主 business-effect 事实保留给其原 owner 对账。回滚不删除 pending message、不撤销已发生的副作用，也不把已有 mail delivery 重新解释成 direct run。

## Example Impact Assessment

新增示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`
- 维护 owner：orchestration、mailbox、host-integration 与 documentation maintainers
- 验证证据：OpenSpec strict validation、documentation impact gate、roadmap consistency 与 mailbox plan bridge contract gate

`docs/development-roadmap.md` 已在本分支同步 Spec 1 的执行顺序、source-owner 边界、支持/拒绝条件、验收证据和 Danny 映射。实施前必须把最终 API、limits、typed errors、support matrix、rollback 和 example runtime path 同步到本 change 的受影响维护文档；未同步的语义不得勾选任务、声明实现完成或归档。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/development-roadmap.md` | orchestration maintainers | 复核 owner 分层及禁止第二 queue/worker/scheduler/state machine |
| components | 修改文档 | `orchestration/planbridge/README.md`、`orchestration/mailbox/README.md` | orchestration/mailbox maintainers | manifest、binding/outcome、owner delegation 与 support matrix 对照 source tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime configuration maintainers | mailbox availability/version 的 constructor/admission 语义；无新增配置时明确保持 `env > file > default` |
| contract/API | 修改文档 | `docs/guides/integration.md`、`docs/mainline-contract-test-index.md` | contract and host-integration maintainers | spec requirements、public DTO、Run/Stream integration、fixture/replay/gate 对照 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | source-owned bounded correlation/outcome；RuntimeRecorder 单写且无新 QueryRuns contract |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples maintainers | README 文档先行、offline marker、no-fallback 与 rollback 路径验证 |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host-integration maintainers | compile/admit/dispatch、typed rejection、duplicate convergence 与 host boundary |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | digest/ack/DLQ 不等于 business commit 或 restore 的反例核对 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | `openspec list --json` 与 roadmap status consistency |

## Validation Strategy

提案阶段执行 `openspec validate publish-capabilities-and-bridge-mailbox-async-host-plans --strict`、OpenSpec documentation/example-impact/roadmap consistency gates、`pwsh -File scripts/check-docs-consistency.ps1` 和 `git diff --check`。

实施阶段先运行受影响 package 的正向、负向、边界和并发测试；再运行 mailbox/planbridge Run/Stream integration、reference-only replay、专用 shell/PowerShell contract gate，最后执行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1` 与 docs consistency。实际命令、exit code、风险、回滚与未执行项将记录到 `verification.md`；本提案不提前宣称这些实现验证已通过。
