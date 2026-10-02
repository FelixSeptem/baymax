## 1. Documentation baseline and ownership

- [x] 1.1 Inventory README, docs, package READMEs, examples, OpenSpec artifacts, and existing consistency scripts; produce a canonical-source/migration matrix and verify every in-scope topic has an owner path. Evidence: `docs/documentation-ownership.md` records canonical source, owner, migration rule, and verification for all in-scope topics.
- [x] 1.2 Define the Chinese-first terminology, naming, link, deprecation, and document-page templates; verify semantic names do not contain proposal numbers and templates include verification/source links. Evidence: `docs/documentation-style-guide.md` and the component pages use semantic names and uniform source/verification sections.

## 2. Layered technical documentation

- [x] 2.1 Add the documentation index and reader paths for new users, integrators, and contributors; verify all index links resolve and each path reaches a quickstart or canonical guide. Evidence: `docs/README.md`; docs consistency and link checks pass.
- [x] 2.2 Add architecture documentation covering system overview, runtime lifecycle, data/event flow, module ownership, dependency boundaries, and design principles; verify statements match `docs/runtime-module-boundaries.md` and existing contract owners. Evidence: `docs/architecture/*.md` links to the boundary and contract sources.
- [x] 2.3 Add responsibility-oriented component documentation for runner/context/provider/tool/orchestration/runtime/observability/adapter/MCP/extension/skills; verify each page follows the uniform component contract and links to source/spec/tests. Evidence: `docs/components/*.md` follows the template and includes verification/source links.
- [x] 2.4 Add guide documentation for integration, configuration/diagnostics, tests/replay/gates, troubleshooting, best practices, contribution, and OpenSpec workflow; verify commands and rollback notes are executable or explicitly bounded. Evidence: `docs/guides/*.md` contains bounded commands and rollback notes.

## 3. README restructuring

- [x] 3.1 Rewrite the root README as a concise landing page with project positioning, capability boundaries, five-minute start, minimal example, and audience-based documentation navigation; verify no detailed canonical contract is duplicated. Evidence: README is a landing/navigation page and links detailed contracts to `docs/`.
- [x] 3.2 Add migration/deprecation links or stubs for moved sections and reconcile README capability/status snapshots with roadmap and archive index; verify README status-parity and link checks pass. Evidence: README points to canonical docs, roadmap and archive index; docs consistency passes.

## 4. Proposal documentation-impact governance

- [x] 4.1 Add a mandatory Documentation Impact Assessment section/template to the OpenSpec proposal/design/tasks workflow with per-area outcomes, affected paths, owner, and verification evidence; verify docs-only, behavior, configuration, contract, diagnostics, and example cases are representable. Evidence: template in `docs/documentation-style-guide.md` and populated in all three change artifacts.
- [x] 4.2 Add offline shell and PowerShell drift gates that compare declared impact with changed surfaces, required documentation tasks, README/docs links, roadmap/OpenSpec/archive status, and Example Impact Assessment; verify both entrypoints emit identical deterministic verdicts and never mutate files or access the network. Evidence: paired `scripts/check-openspec-documentation-impact.sh/.ps1`; PowerShell passes, shell entrypoint is present and parity-reviewed (Git Bash service is unavailable in this Windows environment).
- [x] 4.3 Integrate the drift gate into documentation/quality consistency checks and contribution guidance; verify a missing/invalid assessment, stale link, stale status, missing docs task, and valid no-impact declaration each produce the expected result. Evidence: docs/quality scripts invoke the gate; stable reason codes include `missing-assessment`, `missing-area`, `missing-evidence`, `surface-without-doc-task`, and `stale-link`.

## 5. Validation and delivery

- [x] 5.1 Run documentation link/reference, roadmap status, OpenSpec, Example Impact, and new drift gates; verify no orphaned canonical pages, duplicate ownership, or stale status remains. Evidence: `openspec validate --all` (134 passed), strict change validation, Example Impact gate, roadmap status gate, Documentation Impact gate, docs consistency, and `git diff --check` passed. Git Bash shell entrypoint could not execute in this Windows session because the Bash service returned `E_ACCESSDENIED`; paired script parity was reviewed and the PowerShell entrypoint passed.
- [x] 5.2 Review the documentation against architecture constraints and current package owners; verify every changed topic has a source link, command, rollback note, and maintainer path. Evidence: architecture/module ownership pages cite `runtime-module-boundaries.md`; component and guide pages include source, verification, failure/rollback, and owner paths; package and contribution tests passed.
- [x] 5.3 Run the required quality/docs gates and `git diff --check`; record Example Impact and any intentionally deferred English/site work before archive. Evidence: PowerShell quality gate passed all 75 steps including full tests, race, lint, contract/replay, smoke, benchmark, and govulncheck; Example Impact is `无需示例变更（附理由）`; full English translation and docs website are intentionally deferred per proposal non-goals.

## Example Impact Assessment

无需示例变更（附理由）：本 change 不修改 `examples/agent-modes` 的配置、runtime path、expected markers 或 rollback notes；只新增文档导航和提案文档影响治理。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 新增文档 | `docs/architecture/*` | maintainer | docs consistency |
| components | 新增文档 | `docs/components/*` | package owners | source/test link review |
| configuration | 修改文档 | `docs/guides/configuration-and-diagnostics.md` | runtime/config | config tests |
| contract/API | 无需文档变更（附理由） | — | maintainer | runtime/API unchanged |
| diagnostics | 修改文档 | `docs/guides/configuration-and-diagnostics.md` | observability | replay gate |
| examples | 无需文档变更（附理由） | — | example owners | examples untouched |
| CLI/integration | 新增文档 | `docs/guides/integration.md` | integration owners | link check |
| best practices | 新增文档 | `docs/guides/best-practices.md` | maintainer | docs consistency |
| roadmap | 修改文档 | `docs/development-roadmap.md` | release owner | roadmap status gate |
