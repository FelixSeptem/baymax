## MODIFIED Requirements

### Requirement: Pause reasons SHALL be independently evidenced by form

The manifest SHALL declare `input_required`, `approval_required`, and `budget_exhausted` as independent capability cells for every execution form. Each cell MUST carry an explicit source-owned evidence state: `supported` requires a producer, persisted safe point, compatible cross-process restore, cancel/timeout/retry race evidence, terminal evidence, and Run/Stream parity; `unsupported` requires a stable reason; `unavailable` reflects an owner or store that cannot currently answer. A legacy source-level pause list MUST NOT promote a cell without matching per-reason evidence. A missing, foreign, or stale cell SHALL return `unsupported_capability` or typed `unavailable` before activation.

#### Scenario: One supported cell does not promote sibling reasons

- **WHEN** a workflow owner evidences `input_required` but has no approval safe point
- **THEN** the workflow manifest marks only `input_required` as supported
- **AND** approval admission returns `unsupported_capability` before checkpoint claim or step dispatch

#### Scenario: Sibling form evidence does not promote a cell

- **WHEN** a team plan requests `budget_exhausted` but only workflow evidence is configured
- **THEN** team budget continuation remains unsupported
- **AND** the bridge does not borrow the workflow checkpoint or activate a member

#### Scenario: Mailbox lifecycle projection is not handler continuation

- **WHEN** a mailbox owner exposes lifecycle checkpoint/projection facts without a handler execution safe point
- **THEN** all host-handler pause cells remain unsupported
- **AND** admission fails before publish, claim, enqueue, handler dispatch, or context assembly

#### Scenario: Supported cell preserves native restore semantics

- **WHEN** a form/reason cell has complete source evidence and a compatible owner/store
- **THEN** the owner restores the same native graph/message/run and preserves completed effects, terminal evidence, and causal identity
- **AND** Run and Stream return equivalent source terminal semantics

#### Scenario: Unsupported team pause is rejected

- **WHEN** a team plan requests approval continuation before team source evidence exists
- **THEN** admission rejects before member activation and does not borrow workflow or child capability

#### Scenario: Child budget pause preserves consumed resources

- **WHEN** a child owner declares `budget_exhausted` supported with source safe-point and fresh-owner restore evidence
- **THEN** restore revalidates current source budget facts, preserves consumed resources and the original deadline, and does not replenish or reset them

#### Scenario: Child without owner evidence does not declare budget pause

- **WHEN** a child source has no owner-backed budget safe point and fresh-owner restore evidence
- **THEN** its `budget_exhausted` cell remains unsupported
- **AND** admission rejects before child execution instead of restarting the parent or child

## Documentation Impact Assessment

- 对外语义变化：是；同步 `docs/guides/integration.md`、`orchestration/planbridge/README.md`、`orchestration/mailbox/README.md`、`docs/development-roadmap.md`。

## Example Impact Assessment

- 修改示例：host execution plan example adds a supported-cell and lifecycle-only mailbox rejection case.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 维护 owner：planbridge、mailbox、integration、runtime architecture
- 验证：`pwsh -File scripts/check-docs-consistency.ps1` and the dedicated pause-matrix gate
