## Why

Baymax already contains substantial architecture, component, integration, and governance material, but an external developer still has to reconstruct the project's design from several competing entry points. The root README mixes current capability facts with historical snapshots, several pages retain stale "proposal in progress" wording, and the ownership map names paths that do not exist. This makes it difficult to understand the request lifecycle, state ownership, supported versus unsupported recovery boundaries, and the intended contribution path without reading source code or archive history first.

The change is needed now because the runtime contract surface has grown significantly while the documentation navigation and status language have not converged. A coherent public reading path is a prerequisite for evaluating the existing library-first and contract-first design, and it will reduce future drift by making each topic's canonical source and verification command explicit.

## What Changes

- Establish one external-developer reading path from the root README through architecture, integration, examples, and verification guidance.
- Add an explicit getting-started guide that states toolchain prerequisites, an offline quickstart, a real Provider path, host responsibilities, and rollback/verification expectations.
- Reframe the architecture documentation around the project's design ideas: library-first embedding, contract-first behavior, source ownership, Run/Stream parity, fail-closed admission, host-owned business effects, and offline evidence.
- Add a canonical cross-component execution model that explains Run, Session, Task, Attempt, Event, Checkpoint, Terminal Outcome, and Business Effect without duplicating field-level contracts.
- Correct the documentation ownership matrix so every canonical path exists and every alias points to the correct source.
- Normalize architecture and component pages around purpose, owned facts/state, lifecycle/data flow, dependencies, failure/rollback, observability, verification, and source links.
- Remove stale "proposal in progress" and "in progress change" wording where `openspec list --json` and the archive prove that the work is complete.
- Replace history-heavy integration prose with a scannable capability/boundary matrix, retaining links to archived proposals for historical detail.
- Extend documentation consistency checks to catch broken canonical paths, stale active-change wording, and missing primary-path links.
- Audit every public Markdown relative link and classify canonical reference versus historical/planning material so archive, package, example, and guide links cannot silently drift.
- Expose the API reference and important source-owner package READMEs through the public index and ownership map, including memory, native event observation, context handoff/projection, and integration verification.
- Remove archived host-context status remnants from package READMEs and extend the stale-status fixture to catch those specific historical phrases without rejecting legitimate runtime `pending` state.
- Keep runtime behavior, public APIs, configuration semantics, diagnostics schemas, examples' execution paths, and OpenSpec capability requirements unchanged.

## Capabilities

### New Capabilities

None. This is a documentation and governance change; `.openspec.yaml` sets `skip_specs: true` because no runtime or contract requirement changes.

### Modified Capabilities

None. Existing specs remain the behavior source of truth and are not modified.

## Impact

- Documentation only: `README.md`, `docs/README.md`, `docs/architecture/*`, `docs/components/*`, `docs/guides/*`, `docs/documentation-ownership.md`, `docs/api-reference-d1.md`, owner package READMEs, selected example READMEs, and stale-status canonical pages.
- Governance tooling: documentation consistency checks and their focused fixtures, without changing runtime quality thresholds.
- No Go packages, exported APIs, configuration keys, diagnostics fields, event identity, terminal semantics, or example runtime paths are changed.
- The change is reversible by reverting the documentation and check updates; no persisted data or deployment state is affected.

## Example Impact Assessment

无需示例变更（附理由）：the change updates navigation and explanatory text only. Existing examples keep the same runtime path, semantic anchors, expected markers, and rollback behavior; links to them may be reorganized but no example code or mode contract changes.

## Public Semantic Documentation Commitment

- 对外语义变化：否
- 文档同步状态：无需同步（附理由）：本 change only clarifies and reindexes behavior already defined by code, tests, and existing contracts; it does not add or alter a public API, runtime behavior, configuration default, diagnostic field, event ordering, terminal state, or integration protocol.
- 维护文档：—
- 维护 owner：documentation maintainers
- 验证证据：`openspec validate --all`；`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/architecture/overview.md`, `docs/architecture/design-principles.md`, `docs/architecture/runtime-lifecycle.md`, `docs/architecture/data-event-flow.md`, `docs/architecture/module-ownership.md` | documentation maintainers + runtime owners | `pwsh -File scripts/check-docs-consistency.ps1` |
| components | 修改文档 | `docs/components/*.md`, `memory/README.md`, `host/eventadapter/README.md`, `context/handoff/README.md`, `context/hostprojection/README.md`, `integration/README.md` | component owners | page checklist and docs consistency gate |
| configuration | 修改文档 | `docs/guides/configuration-and-diagnostics.md`, `docs/runtime-config-diagnostics.md` | runtime/config owners | `go test ./runtime/config ./runtime/diagnostics -count=1` |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md`, `docs/api-reference-d1.md` | integration and contract owners | link check plus existing contract index test |
| diagnostics | 修改文档 | `docs/components/observability.md`, `docs/guides/testing-replay-gates.md` | observability owners | replay/index references and docs consistency gate |
| examples | 修改文档 | `examples/quickstart/README.md` and navigation links; no runtime example code | example owners | link audit and existing example smoke references |
| CLI/integration | 修改文档 | `docs/guides/integration.md`, `docs/guides/getting-started.md`, `docs/guides/contribution.md` | integration maintainers | docs consistency gate |
| best practices | 修改文档 | `docs/guides/best-practices.md`, `docs/guides/troubleshooting.md` | documentation maintainers | focused documentation checks |
| roadmap | 修改文档 | `docs/development-roadmap.md` only if status wording needs correction | release owner | `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` |
