## MODIFIED Requirements

### Requirement: Team and mailbox continuation capability SHALL be owner-evidenced per pause reason

The public continuation capability document SHALL expose bounded, detached evidence for each canonical pause reason. Each entry MUST declare whether it is supported and, respectively, a valid evidence reference or a stable unsupported reason. Validation MUST reject missing, duplicate, foreign, package-only, or unbounded evidence. Capability manifest promotion MUST use only the configured source owner's entry for the requested execution form and deployment scope.

#### Scenario: Partial source evidence is representable

- **WHEN** a source supports input pause but not approval or budget pause
- **THEN** validation succeeds with two explicit unsupported entries
- **AND** only the input cell is promoted in the manifest

#### Scenario: Package presence cannot promote capability

- **WHEN** a source returns a pause reason name without source evidence
- **THEN** the corresponding cell remains unsupported or unavailable
- **AND** admission does not activate the source

#### Scenario: Lifecycle-only mailbox evidence is bounded

- **WHEN** a mailbox source reports lifecycle continuation without handler safe-point evidence
- **THEN** the manifest preserves lifecycle facts but leaves handler pause cells and host context restore unsupported
- **AND** each unsupported handler pause cell uses the stable reason `mailbox_handler_safe_point_unsupported`

#### Scenario: Team source promotes only team cells

- **WHEN** a valid team continuation capability source is configured
- **THEN** the manifest promotes only supported team checkpoint/restore/pause cells with its evidence and deployment scope
- **AND** child, workflow, mailbox and resumed-plan cells remain unchanged unless their own owners are configured

#### Scenario: Mailbox source promotes only mailbox cells

- **WHEN** a valid mailbox host-plan continuation capability source is configured
- **THEN** the manifest promotes only supported mailbox checkpoint/restore/pause, causal-retry and mailbox-continuation cells with its evidence and deployment scope
- **AND** it does not imply resumed planbridge execution

#### Scenario: Missing owner evidence remains rejectable

- **WHEN** a planner asks for a team or mailbox continuation cell without a valid owner capability source, version, namespace or evidence reference
- **THEN** the manifest reports `unsupported` or `unavailable` with a bounded reason
- **AND** admission rejects before member activation, publish, consume or resume

## Documentation Impact Assessment

- 对外语义变化：是；同步 `docs/guides/integration.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`。

## Example Impact Assessment

- 修改示例：manifest fixture and host-plan example show partial and rejected cells.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 维护 owner：runtime types、planbridge、diagnostics、contracts
- 验证：OpenSpec/documentation-impact/example-impact/roadmap gates and matrix fixture tests
