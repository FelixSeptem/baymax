## Why

Baymax's `host_context_projection.v1` is currently admitted only by direct Run/Stream, while child, team, workflow, and mailbox owners now have separate continuation paths. Hosts cannot bind a filtered context projection to the exact composite node and restored continuation, so independently valid context and checkpoint evidence can drift or be incorrectly reused across members, steps, messages, or attempts. The roadmap's conditional requirement is now triggered by the need to use one reference-first context/handoff contract across these already-supported composite owners.

## What Changes

- Extend the existing reference-only projection contract with explicit source-form and native node/continuation correlation for child, team member, and workflow step paths. Mailbox context injection remains unsupported because its continuation API only projects lifecycle outcomes and has no handler execution safe point.
- Require each owner to validate projection identity/digest, graph or message lineage, parent capability subset, authority/scope/causation, and the matching restore/checkpoint reference before resolving any context source.
- Revalidate current host authorization through the existing resolver after source restore validation and before context material is assembled or injected; keep resolved content transient.
- Publish independent context-assembly capability cells per owner and deployment profile. Missing owner binding, evidence, or current resolver remains typed unsupported/unavailable before source activation or restore claim.
- Preserve existing direct Run/Stream behavior, existing owner checkpoint/history stores, `handoff.v1` semantics, and same-host single-writer deployment limits.
- Add owner-specific integration for implemented paths, read-only replay, applicable Run/Stream parity, privacy, drift, and shell/PowerShell contract evidence; keep mailbox explicitly unsupported in capability/replay evidence and track a mailbox-owned handler safe point separately.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `host-context-assembly-handoff`: define composite source/node binding, context capability manifest cells, restore-time correlation validation, and per-form unsupported semantics.
- `composite-continuation-restore`: bind projection identity and capability digest to the source-owned graph/member/step or message/delivery continuation and its predecessor/successor lineage.
- `host-runtime-capability-manifest`: declare composite context assembly/handoff independently by form, owner, deployment profile, and evidence reference.

## Impact

- Affected owners: `context/hostprojection`, `context/assembler`, `context/handoff`, `core/types`, `core/runner`, `orchestration/composer`, `orchestration/teams`, `orchestration/workflow`, `orchestration/mailbox`, and `orchestration/planbridge`.
- Public API and runtime semantics change additively. Existing direct descriptor behavior and no-projection behavior remain unchanged. Composite admission remains fail-closed unless the matching owner binding and capability evidence are supplied.
- Host filtering, current authorization, source-body resolution, business revision/effect authorization, and business commit remain host-owned. No shared context database, alternate journal, queue, worker, scheduler, provider SDK, or composite execution loop is introduced.
- Deployment claims remain same-host, single-writer, sequential reopen; this change does not imply multi-writer or cross-machine restore.

## Example Impact Assessment

修改示例

理由：extend the existing offline `examples/host-context-assembly` to cover at least one supported owner-bound composite path, plus a rejected cross-member or stale-continuation case. The example remains offline and uses synthetic opaque references, fake resolver/model/tool, and the real owner API.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `context/hostprojection/README.md`, `context/handoff/README.md`, `orchestration/teams/README.md`, `orchestration/mailbox/README.md`, `examples/host-context-assembly/README.md`
- 维护 owner：context projection/assembler owners, each composite source owner, planbridge, documentation and examples maintainers
- 验证证据：strict OpenSpec validation, owner unit/integration/restart tests, Run/Stream parity, reference-only replay, shell/PowerShell gates, `openspec validate --all`, full Go/race/lint/quality/docs verification recorded in `verification.md`

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | context and source owners | ownership/no-second-store contract tests |
| components | 修改文档 | `context/hostprojection/README.md`, `context/handoff/README.md` | context owners | projection/handoff tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config owner | fixed limits and no-new-config contract gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | context and source owners | per-form integration tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay owner | replay/gate mapping tests |
| examples | 修改文档 | `examples/host-context-assembly/README.md` | examples owner | offline smoke and marker checks |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | planbridge/source owners | pre-activation rejection tests |
| best practices | 修改文档 | `docs/guides/integration.md`, `context/hostprojection/README.md` | context and source owners | per-owner authorization, fail-closed and rollback contract tests |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | roadmap status consistency gate |

## Risks and Rollback

- Incorrect composite correlation could resolve context under another member's, step's, or delivery's identity. Every binding must be checked by its source owner against native continuation material before resolver invocation.
- Reusing an old digest could bypass changed source authorization. Restore must call the host resolver for current authorization; historical digest is consistency evidence only.
- Partial resolver results could leak context into a partially restored graph. A composite projection is all-or-nothing for its declared node; any missing/denied reference rejects before injection or execution continues.
- Rollback stops supplying composite projection bindings and returns to the already-supported per-owner behavior without host projection. It does not delete checkpoints, histories, source records, or reverse completed business effects.
