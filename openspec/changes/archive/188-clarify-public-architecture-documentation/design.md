## Design Summary

This change treats documentation as a navigable public product with explicit facts sources, rather than as a collection of release notes. The root README answers what Baymax is and is not. `docs/README.md` answers where to go next. The getting-started guide proves the toolchain and first offline/Provider paths before the reader enters architecture and integration detail. Architecture pages explain why the system is shaped this way. Component pages explain who owns facts and state. Guides explain how an integrator performs a task. OpenSpec and archive pages remain the historical and governance record, not the primary onboarding path.

## Reader Model and Information Architecture

The primary path is:

```text
README.md
  -> docs/README.md
  -> docs/guides/getting-started.md
  -> docs/architecture/overview.md
  -> docs/architecture/design-principles.md
  -> docs/architecture/runtime-lifecycle.md
  -> docs/architecture/data-event-flow.md
  -> docs/architecture/module-ownership.md
  -> docs/guides/integration.md
  -> examples/<scenario>/README.md
  -> docs/guides/testing-replay-gates.md
```

The path is task-oriented. Each page must link forward to the next reader decision and backward to the canonical source map. Architecture pages may summarize contracts, but field-level requirements remain in code, tests, and `openspec/specs`.

The path now includes `docs/architecture/execution-model.md` between the
architecture overview and design principles. This page is the canonical
semantic glossary for cross-component nouns and identity boundaries. It uses a
single lifecycle diagram and compact owner matrix, then links to source-owned
contracts. It must not define a new API or imply that a terminal-only form,
callback projection, metadata snapshot, or reused RunID is durable recovery.

The public index also classifies specialized guides, canonical reference pages,
and historical/planning studies. A repository-relative link audit covers the
root README and all public Markdown below `docs/` (excluding internal
`docs/superpowers` drafts). External URLs and fragment-only links are ignored;
missing package, example, archive, guide, or canonical paths are reported as a
stable documentation-navigation issue.

The public index also exposes the API reference and the source-owner package
READMEs that an integrator needs to understand state ownership: memory,
native event observation, context handoff/projection, and cross-module
integration verification. These remain package-level facts sources rather than
duplicated component contracts.

## Canonical Source Rules

`docs/documentation-ownership.md` remains the source map for topic ownership. Every row must point to an existing file or directory, name an owner, and provide an executable verification command. Aliases must either be real pages or be removed; they must not silently duplicate a canonical contract. Historical state belongs in `openspec/changes/archive/INDEX.md` and archived change artifacts.

The consistency check will resolve repository-relative links and source-map paths. It will also compare active-change language with `openspec list --json`, so a page cannot claim a change is in progress when the change is archived or absent. The public component pages must use current source-owner facts; archive links must target the actual archive directory rather than an old active-change path.

The stale-status fixture includes archived host-context phrases that previously
survived the generic marker list (`Active host-context assembly boundary`,
`active host-context bridge`, and the archived task-6.3 pending wording). It
does not reject ordinary runtime words such as `pending action` or `pending
state`; only the known historical claims are classified as documentation drift.

## Architecture Narrative

The architecture pages will explain these invariants in concrete terms:

- **Library-first**: the host owns credentials, network, persistence, business authorization, effects, and global routing; Baymax supplies embeddable execution primitives.
- **Contract-first**: `core/types` defines cross-module DTOs and error/event vocabulary; behavior changes require tests, replay evidence, and gates.
- **Source ownership**: the runner, workflow, scheduler, mailbox, and other owners retain their own lifecycle facts; adapters project or correlate facts and do not invent durable history or terminal state.
- **Run/Stream parity**: streaming changes delivery shape, not decision, cancellation, timeout, or terminal semantics.
- **Fail-closed boundaries**: unsupported, unavailable, stale, foreign, expired, or malformed inputs are rejected before side effects; no implicit retry or fallback to a new run is documented as recovery.
- **Evidence over assertion**: diagnostics, replay fixtures, contract tests, and gates prove behavior; README text does not promote an unsupported capability.

The lifecycle page will show the common path (`admit -> prepare -> model/tool steps -> decide -> finalize/abort`) and the distinct ownership of pause, retry, timeout, cancellation, checkpoint, history replay, and business commit. The data-event-flow page will distinguish source facts, projected events, diagnostic writes, and host-owned effects. The module-ownership page will make allowed and forbidden dependency directions visible.

## Component Page Contract

Each component page keeps its existing local details but follows the same order:

1. Purpose and non-goals.
2. Owned facts/state and lifecycle.
3. Inputs, outputs, and typed failures.
4. Allowed and forbidden dependencies.
5. Run/Stream or sync/async parity rules.
6. Failure, rollback, and unsupported boundaries.
7. Observability and recorder ownership.
8. Verification commands and canonical source/spec/test/example links.

Pages will link to each other instead of repeating complete contract tables. Where a package README currently carries a stale proposal snapshot, it will be reduced to current behavior plus links to the canonical architecture or archive record.

## Integration Guide Shape

The integration guide will open with a decision matrix for direct Run/Stream, tools, MCP, provider adapters, orchestration, native events/history, checkpoints, and host bridges. Each row will identify the owner, whether it is durable, whether it is read-only, the default boundary, and the next guide/example. Long historical explanations will become short boundary notes with links to the relevant archive proposal.

The guide will explicitly separate:

- execution from observation;
- source history replay from continuation restore;
- source terminal outcome from host business commit;
- admission/readiness from execution;
- retry/lease ownership from application idempotency.

This prevents external integrators from inferring capabilities from package names or DTOs.

## Validation and Rollback

Validation is documentation-focused:

- repository-relative link and canonical-path checks;
- active/archive status consistency checks;
- required reader-path markers;
- API-reference and source-owner package navigation checks;
- archived host-context status fixture checks;
- `openspec validate --all` with `skip_specs: true`;
- existing Go package, contract-index, replay, and quality gates to prove no runtime behavior changed.

The change rolls back by reverting the documentation and focused check commits. No runtime store, configuration, generated artifact, or persisted state is touched.

## Example Impact Assessment

无需示例变更（附理由）：examples are only re-indexed and linked from the reader path. No example implementation, configuration, semantic anchor, expected marker, or rollback note changes.

## Public Semantic Documentation Commitment

- 对外语义变化：否
- 文档同步状态：无需同步（附理由）：本 change only clarifies and reindexes behavior already defined by code, tests, and existing contracts; it does not add or alter a public API, runtime behavior, configuration default, diagnostic field, event ordering, terminal state, or integration protocol.
- 维护文档：—
- 维护 owner：documentation maintainers
- 验证证据：`openspec validate --all`；`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/architecture/*.md`, `docs/runtime-harness-architecture.md` | documentation + runtime owners | docs consistency |
| components | 修改文档 | `docs/components/*.md`, package READMEs with stale status | component owners | component page checklist |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md`, configuration guide | runtime/config owners | config tests + docs consistency |
| contract/API | 修改文档 | integration and contract index navigation | integration owners | contract index test |
| diagnostics | 修改文档 | observability and replay guide | observability owners | replay references |
| examples | 修改文档 | `examples/quickstart/README.md` and navigation links; no runtime example code | example owners | link audit and example smoke references |
| CLI/integration | 修改文档 | integration/contribution guides | integration maintainers | docs consistency |
| best practices | 修改文档 | best-practices/troubleshooting | documentation maintainers | focused docs checks |
| roadmap | 修改文档 | roadmap status wording if required | release owner | roadmap status gate |
