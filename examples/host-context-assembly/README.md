# Host context assembly and handoff

状态：进行中 OpenSpec `establish-host-context-assembly-handoff-contract` 的已交付离线示例。它使用真实 runner、fake model/tool、fake current-authority resolver 和 synthetic opaque reference；不使用 live provider、business database、network 或持久的 source body。dedicated replay 和双平台 contract gates 已交付；标准全量最终验证仍待该 active change 完成。运行：

```powershell
go run ./examples/host-context-assembly
```

## Semantic anchor

目标合同是显式 opt-in、provider-neutral、reference-only 的 `host_context_projection.v1`：宿主先决定角色可见性、来源信任、business reference/tool allowlist 和 private memory/knowledge access；Baymax 只验证有界 projection identity/version/digest、opaque source references、capability 收窄和可选 host-plan correlation，然后复用既有 Context Assembler、只读预算投影、compaction、`handoff.v1` 与 reference-first restore owner。

digest 是一致性证据，不是当前 source access、业务授权或 business commit。source body、credentials、private memory/knowledge 原文、effect payload 与 raw resolver error 不属于 descriptor、handoff correlation、fixture 或诊断。

## Runtime path

```text
host pre-filter + current authorization
  -> host_context_projection.v1 (bounded opaque references/digests)
  -> validation before resolver
  -> host-owned current-source resolver
  -> existing Context Assembler / budget projection / compaction
  -> existing handoff.v1 correlation
  -> existing source validation + host reauthorization before restore injection
```

当前 direct Run/Stream 的 assembly 入口已能显式传入瞬态 descriptor/resolver；此 opt-in path 要求既有 Context Assembler 已启用。若 assembler 禁用，保留既有 bypass 语义，projection 不会被 admit 或 resolve。固定 API 限制为 32 KiB descriptor、64 references、256 UTF-8 bytes/reference 和 128 UTF-8 bytes/digest。宿主必须继续使用既有 Context Assembler 与 handoff APIs，不能把本 README、OpenSpec artifact、replay 或 handoff digest 当作可用 resolver/restore authorization。`host_context.*` 是有限 fail-closed 分类，不暴露 source body 或 raw callback error。

## Current support matrix

| Path | Status | Boundary |
| --- | --- | --- |
| Direct `Run` context assembly | supported when Context Assembler is enabled | `RunRequest` passes an opt-in detached descriptor and host resolver into the existing assembler; when the assembler is disabled, its existing bypass semantics apply and the projection is not admitted or resolved. No projection means legacy behavior. |
| Direct `Stream` context assembly | supported when Context Assembler is enabled | Uses the same assembly entry path and material ordering as direct `Run`; when disabled it preserves assembler bypass without descriptor admission/resolution, and it fabricates no callback or terminal behavior. |
| Legal existing `handoff.v1` cuts | supported | Existing handoff logic retains only additive, nullable, reference-only correlation; no resolved body is persisted. |
| Correlated restore validation and material resolution | supported with explicit resolver | Existing reference-first validation remains authoritative and `runner.RestoreHandoffWithHostProjection` requires fresh current authorization. It returns transient resolution for caller-owned injection only; the runtime neither injects, resumes, nor runs itself. |
| Cross-process continuation | unsupported | A descriptor or digest is not a durable continuation or current authorization. |
| Mailbox/workflow/team recovery | unsupported | No composed recovery owner, mailbox worker, workflow/team bridge, or shared state is added. |
| Dynamic-action continuation | unsupported | Existing dynamic-action/checkpoint owners retain their own admission and restore semantics. |

This matrix is intentionally narrower than `orchestration/planbridge`: optional plan correlation is evidence only; it cannot compile, dispatch, resume, or authorize a plan.

## Expected markers

离线 fake model/tool 示例会验证并输出以下 markers：

- `host_context.projection_admitted`：已验证、已授权的预过滤 projection 进入现有 assembly path。
- `host_context.capability_expansion`：child 扩大 parent capability 时在 resolver 前拒绝。
- `host_context.restore_revalidation_failed`：handoff restore 的 source reference/digest/current authorization 漂移时，在 context injection 前拒绝。
- `host_context.run_stream_equivalent`：等价 Run/Stream 的规范化 assembly、budget、handoff/restore evidence 相同。
- `host_context.legacy_rollback`：停止提交 descriptor/resolver 后，既有 assembler 路径仍只接收原始 caller messages。

这些是离线示例 markers，不是 runtime event、诊断 schema 或承诺的 wire vocabulary。对应 smoke 测试为 `go test ./examples/host-context-assembly -run TestOfflineExampleEmitsContractMarkers -count=1`。

## Boundaries and rollback

本 change 不创建 context database、global projection registry、budget ledger、Provider SDK dependency、Host command、outbox/mailbox worker、business store 或新的 execution/terminal/recovery state machine。预算、业务授权、effect/revision CAS 和 commit 继续由既有 owner 或宿主负责；诊断仍只能经 `observability/event.RuntimeRecorder`。

回滚方式是停止提交未来的 opt-in projection bridge 并使用现有宿主预过滤 + assembler 路径。不得删除 checkpoint/journal/application record，也不能把回滚表述为撤销已经发生的业务 effect。完整计划见归档 [proposal](../../openspec/changes/archive/165-establish-host-context-assembly-handoff-contract/proposal.md) 和 [design](../../openspec/changes/archive/165-establish-host-context-assembly-handoff-contract/design.md)。
