## MODIFIED Requirements

### Requirement: Plan bridge continuation capability SHALL be form and owner scoped

The host execution plan bridge MUST validate the requested execution form, pause reason, owner namespace, capability version, and deployment scope before source activation. It SHALL admit only when the manifest cell is supported by matching source-owned evidence. Missing, unsupported, unavailable, foreign, or version-mismatched cells MUST return typed errors before Run, Stream, publish, enqueue, retry, or checkpoint claim. The bridge MUST preserve the owner's native restore and terminal semantics and MUST NOT fall back to a fresh parent Run or another form's owner.

#### Scenario: Unsupported approval is rejected before effects

- **WHEN** a plan requests approval continuation for a form whose owner only evidences input pause
- **THEN** compile/admission returns `unsupported_capability` before any effect or source claim

#### Scenario: Run and Stream share the matrix decision

- **WHEN** equivalent plans request a supported or unsupported form/reason through Run and Stream
- **THEN** both return the same admission/terminal decision
- **AND** Stream only changes event delivery

#### Scenario: Mailbox lifecycle does not enable handler restore

- **WHEN** a mailbox plan has lifecycle continuation facts but no handler safe point
- **THEN** the bridge rejects handler continuation before publish or handler dispatch
- **AND** it does not synthesize context restore or direct execution

#### Scenario: Sibling owner evidence does not promote workflow

- **WHEN** a workflow plan is bound only to child or team continuation evidence
- **THEN** workflow continuation remains unsupported and activation is rejected before step dispatch

#### Scenario: Mailbox lifecycle support does not imply handler restore

- **WHEN** a mailbox owner exposes lifecycle admission/projection but not a handler execution safe point
- **THEN** the bridge returns `unsupported_capability` before publish, claim, or handler dispatch

## Documentation Impact Assessment

- 对外语义变化：是；同步 `orchestration/planbridge/README.md`、`docs/guides/integration.md`、`examples/host-execution-plan/README.md`、`docs/development-roadmap.md`。

## Example Impact Assessment

- 修改示例：host execution plan example adds matrix admission and rejection output.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 维护 owner：planbridge、integration、examples、runtime architecture
- 验证：planbridge contract tests, dedicated gate, docs consistency and roadmap gates
