## Context

See `proposal.md` for the traceability reproduction and approved scope. The
repository already has Go contribution checks, source-map documentation,
paired quality scripts and an OpenSpec status gate. These provide the validation
and status boundaries without adding a service or changing a runtime port.

## Goals / Non-Goals

**Goals:** Provide a versioned, machine-readable source for unique semantic
owners and their verifiable evidence, validate it offline, and align public
navigation and documentation ownership.

**Non-Goals:** No Run/Stream, terminal, recovery, effect, diagnostics,
configuration or provider dependency changes; no global registry, service
locator, execution control plane or premature port narrowing.

## Decisions

1. Store `owner-evidence-index.v1` in `docs/owner-evidence-index.v1.json` and
   link it from a human-readable guide. This avoids competing copies of the
   owner-to-evidence rows. An unstructured Markdown-only table was rejected
   because strict shape and reference checks would be fragile.
2. Split resolver material admission (`context/hostprojection`) from model
   surface (`model/conformance`) and enroll source lifecycle
   (`host/sourceadapter`) separately. Multiple owners on one entry are invalid;
   coordination remains with the relevant consumers, not a synthetic owner.
3. Validate repository-relative paths, required fields, exact owner/source-map
   association, replay mode and quality gate wiring in `tool/contributioncheck`.
   A replay exemption requires a concrete reason. Tests cover invalid examples
   and the checked-in artifact, and paired shell/PowerShell scripts execute the
   same focused contract tests in the standard quality paths.
4. Delegate active/archive status consistency to the existing OpenSpec roadmap
   gate, which remains in the normal documentation/quality path. Do not invent a
   second status source in the index.

## Risks / Trade-offs

- Existing file paths do not alone prove a test's semantic strength -> list
  concrete tests, keep dedicated contract gates, and review evidence on change.
- Markdown source-map syntax may drift -> parse exact row cells and fail closed
  for missing or contradictory semantic/owner associations.
- A gate string could be present without executable invocation -> test the
  actual blocking shell and PowerShell invocation shapes, not arbitrary text.
- Index scope can be mistaken for complete repository coverage -> explicitly
  describe the three enrolled semantics and do not claim universal coverage.

## Migration Plan

Add the static index and quality checks alongside current contract docs; no
runtime migration is required. Roll back by removing the index, validator,
dedicated gate and navigation references, leaving existing runtime semantics and
source facts untouched.

## Example Impact Assessment

无需示例变更（附理由）：此变更只增加离线 evidence index，不改变
`examples/agent-modes` 的配置、runtime path、semantic anchor、expected
markers 或 rollback notes，也不引入运行时示例路径。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/README.md`、`docs/owner-evidence-index.md`、`docs/documentation-ownership.md`、`docs/mainline-contract-test-index.md`、`docs/runtime-module-boundaries.md`、`docs/development-roadmap.md`
- 维护 owner：`tool/contributioncheck` 和 documentation maintainers
- 验证证据：`pwsh -File scripts/check-owner-evidence-index-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | boundary review |
| components | 修改文档 | `docs/documentation-ownership.md` | documentation maintainers | owner index gate |
| configuration | 无需文档变更（附理由） | no new runtime keys | runtime maintainers | offline validation |
| contract/API | 修改文档 | `docs/owner-evidence-index.md` | contributioncheck maintainers | owner index gate |
| diagnostics | 无需文档变更（附理由） | no diagnostic writer or fields | runtime maintainers | boundary review |
| examples | 无需文档变更（附理由） | no runtime example changes | example maintainers | example impact assessment |
| CLI/integration | 修改文档 | `README.md`, `docs/README.md` | documentation maintainers | docs consistency |
| best practices | 修改文档 | `docs/mainline-contract-test-index.md` | documentation maintainers | owner index gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status gate |
