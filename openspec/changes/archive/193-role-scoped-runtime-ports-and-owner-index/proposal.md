## Why

Roadmap E's owner-traceability trigger is met: the mainline index describes
archive 192's model-surface and resolver evidence together but does not identify
their distinct owners or a complete owner-to-public-documentation chain.  The
documentation source map assigns both to broad context/runner owners and omits
`model/conformance`.  The source-adapter lifecycle is another independently
owned contract shape.  This makes owner and evidence review dependent on manual
cross-document interpretation.

No evidence establishes that two different callers can cross owner boundaries
through the same broad runtime interface.  Therefore this increment addresses
the evidenced index gap only, not port narrowing.

## What Changes

- Add a versioned, static index of enrolled public semantics, each with exactly
  one owner, a main spec, tests, replay evidence or justified non-applicability,
  dedicated gate, public documentation and rollback/non-goal boundary.
- Add strict offline validation for malformed, ambiguous, missing, conflicting
  or unwired evidence and connect paired dedicated gates to normal quality.
- Enroll resolver material admission, provider-neutral model-surface evidence
  and source-adapter lifecycle as three separate semantics with three owners.
- Align the documentation source map, navigation, contract index, architecture
  boundary and roadmap status without changing runtime behavior.

## Capabilities

### New Capabilities

- `owner-evidence-index`: versioned, read-only semantic-owner evidence and
  blocking repository governance validation.

### Modified Capabilities

None.

## Impact

Owners are `tool/contributioncheck` and documentation maintainers for the
index, with `context/hostprojection`, `model/conformance`, and
`host/sourceadapter` retaining their own semantics. Deployment is repository
development/CI only. There is no runtime registry, new source owner, config,
diagnostic writer, provider dependency, execution, authorization, recovery, or
effect control. Rollback removes the static index and its checks and links;
existing runtime owners and stored facts remain untouched. The principal risk
is false confidence from stale or ambiguous references, mitigated by strict
structure, owner-map alignment, file existence, and gate-wiring checks.

## Example Impact Assessment

无需示例变更（附理由）：此变更仅为既有公开语义建立离线 owner/证据索引；
不改变 `examples/agent-modes` 的配置、runtime path、semantic anchor、
expected markers 或 rollback notes，也不增加运行时使用方式。

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
