## ADDED Requirements

### Requirement: Manifest expansion SHALL keep history evidence source-specific

The versioned host runtime capability manifest SHALL expose durable history and
after-cursor replay independently for runner, isolated child, team, workflow,
and mailbox forms. Each value SHALL include the source namespace, retention and
cursor boundary, availability, unsupported reason when false, and a reference
to fixture or contract evidence. Planner and admission MUST NOT infer support
from a form type, import, DTO, callback adapter, or mailbox capability.

#### Scenario: Unsupported composite history is rejected before read
- **WHEN** a plan asks for history for a child, team, or workflow source whose
  owner has not supplied retained evidence
- **THEN** the manifest reports unsupported with a stable reason
- **AND** admission does not create a cursor, subscribe to callbacks, or run
  the source

#### Scenario: Available source evidence is independently reported
- **WHEN** one source form has a retained journal and another source form does
  not
- **THEN** the manifest reports their history cells independently
- **AND** support is not borrowed across forms

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 维护文档：`docs/development-roadmap.md`、`docs/guides/integration.md`、`orchestration/planbridge/README.md`、`README.md`
- 维护 owner：host-integration、runtime-core 与 documentation maintainers
- 验证命令：manifest contract gate、`openspec validate retain-run-and-composite-source-history --strict`、`go test ./...`

## Example Impact Assessment

无需示例变更（附理由）。本 delta 只细化 manifest 的 source-specific evidence；history behavior 的 executable example is covered by the native event history example path.
