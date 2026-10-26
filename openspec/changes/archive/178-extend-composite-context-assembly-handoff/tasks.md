## 1. Contract and Source Audit

- [x] 1.1 Audit direct projection, handoff, runner restore, child, team, workflow, mailbox continuation APIs and capability cells; map each path's exact node identity, safe point, resolver order, deployment scope, and unsupported dimensions with code-backed tests.
- [x] 1.2 Define bounded additive projection-to-source correlation and typed mismatch/unavailable errors; verify canonical digest coverage, exact node binding, existing 32 KiB/64-reference limits, and no serialized source bodies.
- [x] 1.3 Extend capability manifest DTO/validation with independently unsupported context assembly, handoff, and restore cells for direct, child/fanout, team, workflow, and mailbox; verify missing/foreign evidence rejects before side effects.

## 2. Context Contract and Owner Integrations

- [x] 2.1 Add shared detached source/node binding and restore-time validation to `context/hostprojection` and handoff correlation; retain current direct `host_context_projection.v1` compatibility and exact limits.
- [x] 2.2 Integrate Composer child/fanout context binding at task/attempt safe points; verify parent capability subset, source continuation binding, fresh resolver authorization, completed-effect preservation, and typed rejection before dispatch.
- [x] 2.3 Integrate team parallel/vote context binding per member; verify sibling isolation, aggregate state continuity, current authorization, and no member reactivation on projection failure.
- [x] 2.4 Integrate workflow context binding per step; verify dependency lineage, no inheritance to unrelated steps, fresh resolver authorization, and restore of the same workflow checkpoint.
- [x] 2.5 Audit mailbox host-plan continuation at the actual owner path; prove it only projects lifecycle outcomes and has no handler context safe point, keep mailbox context typed unsupported in manifest/replay, and record a mailbox-owner follow-up without adding a planbridge executor.
- [x] 2.6 Validate resolver ordering and atomic claim races for each enabled owner; source generation/fence changes between validation and claim MUST fail closed before assembly or execution continues.

## 3. Run/Stream, Replay, and Capability Evidence

- [x] 3.1 Add applicable owner parity coverage and fake resolver/model/tool integration for each enabled projection path; compare normalized context, budget, handoff, restore, terminal and rejection evidence. Unsupported forms remain rejected; only owners exposing equivalent Run/Stream APIs require both modes.
- [x] 3.2 Add reference-only versioned replay fixtures for identity/graph/node/continuation drift, foreign/stale/expired references, capability expansion, authorization changes, privacy leakage, unsupported owners, and duplicate restore; verify replay is read-only.
- [x] 3.3 Add capability source projections with per-form owner namespace, versions, deployment scope, expiry/retention, resolver requirement and concrete evidence references; owner API support alone must not promote a context cell without its own complete evidence.
- [x] 3.4 Add shell and PowerShell composite-context contract gates and wire both to the quality/replay path; assert rejection-before-resolver/claim/activation and no direct-run/re-enqueue fallback.

## 4. Documentation and Offline Example

- [x] 4.1 Update README, roadmap, integration guide, runtime boundaries, diagnostics, contract index, hostprojection/handoff package READMEs with exact per-form supported/unsupported matrix, identity binding, ordering, deployment scope, privacy, and rollback.
- [x] 4.2 Update `examples/host-context-assembly/README.md` before example code with semantic anchor, runtime paths, expected markers, and rollback notes; extend the offline example with at least one evidenced composite success and one stale/cross-node rejection.
- [x] 4.3 Verify every Public Semantic Documentation Commitment path exists and is modified in this feature branch; verify Example Impact Assessment remains `修改示例` in proposal/design/tasks.

## 5. Verification and Delivery

- [x] 5.1 Run focused context, runner, composer, teams, workflow, mailbox, planbridge and replay positive/negative/boundary suites, including fresh-process restore and race cases for every promoted cell.
- [x] 5.2 Run dedicated shell/PowerShell gates, strict OpenSpec validation, documentation/example/roadmap gates, `openspec validate --all`, and `git diff --check`; record exact evidence and environment limits.
- [x] 5.3 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml ./...`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; record all results before archive.

## Example Impact Assessment

修改示例

理由：the offline host-context assembly example must exercise source-bound composite context admission and a rejection boundary without adding live providers, business data, or a separate execution path.

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
