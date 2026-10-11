## 1. Governance Contract

- [x] 1.1 Update `AGENTS.md`, documentation style guidance, README and roadmap to make the public semantic documentation commitment mandatory; verify all paths and wording are consistent.
- [x] 1.2 Add the proposal-governance spec delta with scenarios for affirmative sync, missing commitment, unjustified no-impact, and cross-artifact drift; verify `openspec validate --strict`.

## 2. Gate Implementation

- [x] 2.1 Extend the PowerShell documentation impact gate to parse and validate the five commitment fields in proposal/design/tasks, emit stable failure codes, and verify a positive declaration has maintenance paths, owner and evidence.
- [x] 2.2 Mirror the same commitment parser and failure taxonomy in the shell gate; verify PowerShell and shell accept/reject equivalent fixtures.
- [x] 2.3 Add or update deterministic gate fixtures/tests covering missing, invalid, contradictory and valid commitments, including a declared document absent from the change diff; verify both gate entrypoints and `check-docs-consistency.*`.

## 3. Verification and Delivery

- [x] 3.1 Run OpenSpec validation, documentation impact/example/roadmap gates, docs consistency and `git diff --check`; record results and any platform limitations.
- [x] 3.2 Run affected governance tests and required repository quality gates; update verification evidence and keep task checkboxes tied to code, test, docs and gate evidence.
- [x] 3.3 Archive only through `scripts/openspec-archive-seq.ps1` after all tasks and evidence are complete; merge and push the feature branch only after merged-result verification.

## Example Impact Assessment

无需示例变更（附理由）：本 change 不修改 `examples/agent-modes` 的运行语义，只增加 proposal 治理承诺和门禁 fixture。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`AGENTS.md`、`README.md`、`docs/documentation-style-guide.md`、`docs/development-roadmap.md`、`docs/guides/openspec-workflow.md`、`openspec/changes/archive/161-enforce-openspec-public-semantics-documentation-contract/specs/proposal-governance-automation-contract/spec.md`
- 维护 owner：governance maintainers
- 验证证据：双平台 documentation impact gate、docs consistency、OpenSpec strict validation

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | maintainer | no runtime architecture changes |
| components | 无需文档变更（附理由） | — | package owners | no component changes |
| configuration | 无需文档变更（附理由） | — | runtime/config | no config changes |
| contract/API | 修改文档 | `AGENTS.md`, governance spec | governance maintainers | OpenSpec strict validation |
| diagnostics | 无需文档变更（附理由） | — | observability | no diagnostics changes |
| examples | 无需文档变更（附理由） | — | example owners | Example Impact gate |
| CLI/integration | 修改文档 | `README.md`, `docs/development-roadmap.md` | release owners | docs consistency gate |
| best practices | 修改文档 | `docs/documentation-style-guide.md` | maintainer | documentation impact gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | release owner | roadmap status gate |
