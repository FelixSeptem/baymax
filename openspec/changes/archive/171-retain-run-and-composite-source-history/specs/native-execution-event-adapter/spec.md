## ADDED Requirements

### Requirement: History expansion SHALL preserve adapter projection ownership

The real-time native event adapter SHALL NOT own retained history, manufacture
journal identities, interpret callback EventID or TraceID as a durable cursor,
or create a source cursor from an application observation sequence. A caller
that needs replay MUST use the source-owned history reader.

#### Scenario: Callback identity cannot be used for replay
- **WHEN** a caller supplies a callback EventID, TraceID, or host observation
  sequence as a history cursor
- **THEN** the source history contract rejects it as malformed or foreign
- **AND** the adapter does not open a journal or activate execution

#### Scenario: Live projection remains available
- **WHEN** a source emits a supported real-time callback
- **THEN** the adapter projects it using the existing callback contract
- **AND** it does not claim durable replay or alter source history ownership

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 维护文档：`docs/runtime-module-boundaries.md`、`host/eventadapter/README.md`、`docs/guides/integration.md`
- 维护 owner：host-integration 与 runtime-core maintainers
- 验证命令：adapter/history contract gates、`go test ./...`、`openspec validate retain-run-and-composite-source-history --strict`

## Example Impact Assessment

无需示例变更（附理由）。该 delta preserves the existing live callback example boundary; the source-history example is maintained by the native history capability.
