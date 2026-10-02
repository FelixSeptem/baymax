## Context

See `proposal.md` for the motivation. The repository already contains authoritative material in `docs/`, package README files, OpenSpec artifacts, examples, and governance scripts, but readers do not have a stable path through that material. The design must improve discoverability without copying contract text into a second source of truth.

The repository is governed by strict module boundaries, OpenSpec-first behavior changes, documentation consistency checks, Example Impact Assessment, and document-first rules for `examples/agent-modes`. This change is documentation and governance only; it must not introduce runtime dependencies or alter package behavior.

## Goals / Non-Goals

**Goals:**

- Define a Chinese-first documentation information architecture serving new users, integrators, and contributors.
- Make README a concise landing page with explicit reader paths and links to canonical detail pages.
- Document architecture, runtime lifecycle, key components, integration patterns, configuration/diagnostics, testing/replay/gates, troubleshooting, best practices, and contribution/OpenSpec workflow.
- Establish a repeatable Documentation Impact Assessment for every new OpenSpec proposal.
- Add deterministic drift checks that compare proposal declarations, changed surfaces, required docs, README/navigation links, roadmap status, and example impact declarations.
- Preserve existing authoritative specs, package ownership, module boundaries, and historical links during migration.

**Non-Goals:**

- No Go code, runtime behavior, provider contract, configuration key, diagnostic schema, or public API change.
- No full English translation or documentation website/toolchain in the first phase.
- No duplication of complete OpenSpec requirements, package internals, or generated API reference in README.
- No changes to `examples/agent-modes` runtime behavior or expected markers.

## Decisions

### 1. Layered information architecture

Create three reader-facing layers while retaining existing specialist documents:

- `README.md`: project story, support/boundary summary, five-minute start, minimal navigation, and common entry points.
- `docs/architecture/`: architecture overview, lifecycle/data-event flow, module/dependency boundaries, ownership map, and design principles.
- `docs/components/`: responsibility-oriented pages for runner/context/provider/tool/orchestration/runtime/observability/adapter/MCP/extension/skills.
- `docs/guides/`: integration, configuration/diagnostics, tests/replay/gates, troubleshooting, best practices, contribution, and OpenSpec workflow.
- `docs/README.md` or equivalent index: audience-based navigation and canonical source map.

Existing detailed documents remain authoritative where they already own a contract. New overview pages link to them and state ownership rather than copying their requirement text.

Alternative: put all detail in README. Rejected because it creates an unbounded landing page and makes drift harder to review. Alternative: add a documentation site now. Deferred because it introduces publishing/toolchain scope without solving the repository's immediate information architecture problem.

### 2. Uniform component page contract

Each component page uses the same semantic sections: purpose, owned facts/state, inputs and outputs, dependencies and forbidden dependencies, lifecycle/data flow, failure and rollback behavior, observability/diagnostics ownership, verification commands, and links to source/specs/examples. This makes architecture review and maintenance comparable across modules.

### 3. Documentation Impact Assessment is mandatory for new proposals

Add a required section and checklist to the OpenSpec proposal/design/tasks workflow. The author must classify impact for architecture, components, configuration, contract/API, diagnostics, examples, CLI/integration, best practices, and roadmap. Allowed outcomes are `新增文档`, `修改文档`, or `无需文档变更（附理由）` per affected area, with required paths and verification evidence when changes are needed.

The assessment is part of the proposal contract even for docs-only changes. A proposal that changes behavior but omits a valid assessment fails validation before implementation.

### 4. Drift gate compares declared and observed surfaces

Add repository-local checks, with shell and PowerShell entrypoints, that:

- parse the Documentation Impact Assessment;
- inspect changed paths and OpenSpec artifacts;
- require documentation tasks when behavior/config/contract/diagnostic/example surfaces change;
- reject stale or missing README/docs links and duplicate canonical sections;
- verify roadmap/OpenSpec/archive status consistency;
- verify Example Impact Assessment remains present and valid;
- produce deterministic reason codes and do not mutate files or call networks.

The gate is review/governance evidence, not a runtime service. It may report `insufficient-evidence` for ambiguous ownership and require explicit human classification rather than guessing.

### 5. Migration and deprecation policy

Move or split only when the old path can be preserved with a redirect/link stub or an explicit migration note. New pages must use semantic names and avoid proposal numbers. The documentation index records canonical ownership and deprecated aliases. Broken links, orphaned pages, and stale roadmap status are blocking issues.

### 6. Example impact remains explicit

This proposal changes documentation structure and governance only. Its Example Impact Assessment is `无需示例变更（附理由）` because no `examples/agent-modes` configuration, runtime path, expected marker, or rollback note changes. Any future proposal that changes those fields must use the existing document-first example workflow.

## Risks / Trade-offs

- [Duplicate explanations drift from specs] → Keep contract detail in existing owner documents; overview pages link to canonical sources and the source map is checked.
- [Documentation tree becomes too broad] → Start with high-value architecture/components/guides indexes and migrate existing pages incrementally; defer a docs website.
- [Impact assessment becomes checkbox theater] → Require changed-surface evidence, affected paths, verification commands, and stable failure reasons; reject empty declarations for behavior changes.
- [Legacy links break during migration] → Preserve link stubs or migration notes and run link/reference checks before merge.
- [Docs gate adds maintenance cost] → Keep checks deterministic, offline, bounded, and focused on ownership/status/link drift; do not parse arbitrary prose as runtime truth.

## Migration Plan

1. Inventory current README, docs, package README, examples, OpenSpec, and governance scripts; define canonical ownership and a migration table.
2. Add the docs index and architecture/component/guide skeletons with content baselines, links, and terminology rules.
3. Rewrite README around reader paths and a concise quickstart; move detailed material only when a canonical destination exists.
4. Add proposal Documentation Impact Assessment instructions, templates, validation, and shell/PowerShell drift gates; wire them into docs/quality checks.
5. Migrate or link existing high-value pages, run link/status/example-impact checks, and record unresolved documentation gaps explicitly.
6. Roll back by reverting the new docs tree and governance gate while preserving existing specialist documents and runtime behavior. No data or configuration migration is required.

## Open Questions

None for the first phase. The language, audience split, layered structure, and mandatory proposal assessment were confirmed before drafting.

## Example Impact Assessment

无需示例变更（附理由）：本 change 只新增技术文档信息架构、README 导航和 OpenSpec 文档影响治理，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或 rollback notes。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 新增文档 | `docs/architecture/*` | maintainer | docs consistency |
| components | 新增文档 | `docs/components/*` | package owners | source/test link review |
| configuration | 修改文档 | `docs/guides/configuration-and-diagnostics.md` | runtime/config | config tests |
| contract/API | 无需文档变更（附理由） | — | maintainer | runtime/API unchanged |
| diagnostics | 修改文档 | `docs/guides/configuration-and-diagnostics.md` | observability | replay gate |
| examples | 无需文档变更（附理由） | — | example owners | no agent-mode behavior change |
| CLI/integration | 新增文档 | `docs/guides/integration.md` | integration owners | link check |
| best practices | 新增文档 | `docs/guides/best-practices.md` | maintainer | docs consistency |
| roadmap | 修改文档 | `docs/development-roadmap.md` | release owner | roadmap status gate |

表格是治理 contract 的最小格式；后续提案不得只写“文档已同步”而省略受影响路径和证据。
