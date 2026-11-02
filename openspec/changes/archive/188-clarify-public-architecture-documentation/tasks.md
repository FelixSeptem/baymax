## 1. Establish the source baseline

- [x] 1.1 Capture `openspec list --json`, the archive index, the actual `docs/architecture` and `docs/components` file lists, and all current stale-status matches in the change verification notes.
- [x] 1.2 Reconcile `docs/documentation-ownership.md` with actual files, including `runtime-lifecycle.md`, `data-event-flow.md`, and `module-ownership.md`; remove or repair non-existent aliases.
- [x] 1.3 Add a focused documentation consistency fixture that fails for a missing canonical path or an active-change claim that is absent from `openspec list --json`.

## 2. Build the external-developer reading path

- [x] 2.1 Rewrite the root README's opening and reader routing so it states the design thesis, capability boundary, and next page without duplicating historical release snapshots.
- [x] 2.2 Rewrite `docs/README.md` as the task-oriented index for new users, integrators, and contributors, with the architecture-to-example path shown explicitly.
- [x] 2.3 Update `docs/architecture/overview.md` and `docs/runtime-harness-architecture.md` so their diagrams and prose agree on the same module layers and host boundary.
- [x] 2.4 Make `docs/architecture/design-principles.md`, `runtime-lifecycle.md`, `data-event-flow.md`, and `module-ownership.md` mutually linkable and explicit about source ownership, Run/Stream parity, fail-closed admission, and host-owned effects.
- [x] 2.5 Add the canonical `docs/architecture/execution-model.md` page with the cross-component noun/owner matrix, lifecycle diagram, identity/recovery boundaries, and host business-effect boundary.

## 3. Normalize component and guide explanations

- [x] 3.1 Audit every `docs/components/*.md` page against the shared page contract; add missing purpose, owned facts/state, lifecycle, failure/rollback, observability, verification, and source links without copying field-level specs.
- [x] 3.2 Remove stale proposal/change labels from `core/runner/README.md`, `core/types/README.md`, `orchestration/README.md`, `docs/guides/integration.md`, and `docs/runtime-config-diagnostics.md`; replace them with current support boundaries and archive links where history is useful.
- [x] 3.3 Refactor `docs/guides/integration.md` into a capability/boundary matrix followed by task-specific paths; keep source history, continuation restore, mailbox, and plan bridge distinctions explicit.
- [x] 3.4 Align `docs/guides/best-practices.md`, `configuration-and-diagnostics.md`, `testing-replay-gates.md`, and `troubleshooting.md` with the same terminology and failure ownership model.
- [x] 3.5 Re-audit component pages against current source-owner contracts; correct the orchestration plan bridge's mailbox capability/status wording and replace stale active-change links with the actual archive source.
- [x] 3.6 Add the external-developer getting-started guide and the real-Provider quickstart README; document toolchain prerequisites, offline/online paths, host ownership, verification and rollback without changing example runtime code.

## 4. Add drift and reader-path checks

- [x] 4.1 Extend the existing documentation consistency script(s) with canonical-path, stale-status, and required-reader-path assertions, preserving stable failure categories and PowerShell/shell parity.
- [x] 4.2 Add a contract-index/documentation fixture proving every reader-path link resolves and every source-map verification command references an existing command or test target.
- [x] 4.3 Document the check ownership and rollback procedure in the documentation style/maintenance guide.
- [x] 4.4 Add the execution-model path to README/docs indexes, documentation ownership, and the navigation fixture; verify the canonical link and source-map command.
- [x] 4.5 Add a public-Markdown relative-link fixture covering package, example, guide, archive, and canonical paths; skip only external URLs, fragment-only links, and internal `docs/superpowers` drafts.
- [x] 4.6 Extend public Markdown link/status coverage to package and example READMEs and the current host-resume/mailbox/runtime canonical pages; classify archived 150/164/165/166 material as historical rather than active.
- [x] 4.7 Expose `docs/api-reference-d1.md` and the memory, native event adapter, context handoff/projection, and integration owner READMEs through the public README/index/source map; add navigation assertions for those canonical paths.

## 5. Validate and hand off

- [x] 5.1 Run `openspec validate --all`, the OpenSpec documentation-impact and roadmap-status checks, and the focused documentation consistency tests.
- [x] 5.2 Run `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml` to verify that the documentation-only change does not alter the Go build or behavior. The second full Go and race runs passed; independent lint reported `0 issues` with only a Windows cache warning. The aggregate quality-gate timeout is recorded in `verification-notes.md`.
- [x] 5.3 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`; record any unavailable commands explicitly. The PowerShell docs check passed. With `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=3600`, the full quality gate completed all 86 steps and passed Go, race, lint, contract, replay, and documentation stages; it exited non-zero only because strict `govulncheck ./...` found `GO-2026-6809` in `crypto/x509@go1.26.9` (fixed in `go1.26.10`). The Bash docs check remains unavailable because the environment returned `E_ACCESSDENIED`.
- [x] 5.4 Review the final diff for stale active-change wording, broken links, duplicated canonical contracts, and accidental generated artifacts before commit. Validation caches remain untracked and are explicitly excluded.
- [x] 5.5 Re-audit public owner READMEs for archived host-context status remnants, run the expanded navigation/stale-status suite and all documentation/OpenSpec gates, and record the evidence in `verification-notes.md`.

## Example Impact Assessment

无需示例变更（附理由）：the work changes documentation navigation and governance checks only; no example runtime path, semantic anchor, expected marker, or rollback note changes.

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
| components | 修改文档 | `docs/components/*.md`, package READMEs | component owners | component checklist |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md`, configuration guide | runtime/config owners | config tests |
| contract/API | 修改文档 | integration guide and contract index | integration owners | contract index test |
| diagnostics | 修改文档 | observability/replay guides | observability owners | replay references |
| examples | 修改文档 | `examples/quickstart/README.md` and navigation links; no runtime example code | example owners | link audit and example smoke references |
| CLI/integration | 修改文档 | integration/contribution guides | integration maintainers | docs consistency |
| best practices | 修改文档 | best-practices/troubleshooting | documentation maintainers | docs consistency |
| roadmap | 修改文档 | roadmap status if needed | release owner | roadmap status gate |
