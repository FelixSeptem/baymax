## ADDED Requirements

### Requirement: Source-owner history fixtures SHALL remain reference-only

Diagnostics replay tooling SHALL validate source capability declarations,
cursor pages, retention floors, terminal evidence, malformed/foreign/expired/
gapped/unsupported/unavailable outcomes and overlap conflicts from bounded
fixtures. Fixture replay MUST NOT construct a source owner, invoke a handler,
model, tool, Run, Stream, retry, resume, restore, mailbox transition, network,
or diagnostics write.

#### Scenario: Unsafe latest fallback is detected
- **WHEN** a fixture marks a cursor expired or gapped but a projection starts
  from latest
- **THEN** the history contract gate fails with a typed fixture classification
- **AND** no source owner or execution path is activated

#### Scenario: Fixture identity and overlap are deterministic
- **WHEN** the same source page is replayed more than once
- **THEN** identical identities deduplicate and conflicting payloads fail closed
- **AND** the fixture result remains independent of runtime process state

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 维护文档：`docs/mainline-contract-test-index.md`、`docs/runtime-config-diagnostics.md`
- 维护 owner：observability、diagnostics-replay 与 documentation maintainers
- 验证命令：history fixture shell/PowerShell gates、`openspec validate retain-run-and-composite-source-history --strict`

## Example Impact Assessment

无需示例变更（附理由）。Fixtures are validation-only and do not change an agent-mode runtime path or user-facing example.
