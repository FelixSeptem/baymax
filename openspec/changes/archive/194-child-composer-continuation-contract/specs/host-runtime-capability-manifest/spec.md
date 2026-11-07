## MODIFIED Requirements

### Requirement: Manifest SHALL expose independent composite continuation cells

The capability manifest SHALL independently declare checkpoint creation, cross-process restore, each pause reason, cancellation, timeout, causal retry, retention/expiry, evidence reference, and deployment limitation for child/fanout, team, workflow, and mailbox. For child/fanout, a form may advertise checkpoint creation and cross-process restore only from its explicitly configured child source owner. Each child pause reason (`input_required`, `approval_required`, `budget_exhausted`) SHALL be independently evidenced by a persisted safe point, compatible fresh-process restore, source position/retention behavior, terminal evidence, and completed-effect non-replay. Missing, foreign, stale, or incomplete evidence SHALL remain unsupported or unavailable before activation. Package presence, execution wiring, or another form's evidence MUST NOT promote child cells or trigger direct fallback. The declared deployment scope MUST be preserved without broadening it. Resumed plan support SHALL be a separate bridge binding cell.

#### Scenario: Child evidence promotes only its complete pause cells
- **WHEN** a configured child owner reports complete fresh-process evidence for a subset of pause reasons
- **THEN** the manifest promotes only those child cells and preserves the source deployment scope

#### Scenario: Child evidence does not enable team
- **WHEN** child continuation cells are supported but team evidence is absent
- **THEN** the manifest reports child support and team `unsupported_capability` independently

#### Scenario: Child execution wiring alone remains unsupported
- **WHEN** child dispatch works but no complete child continuation evidence is configured
- **THEN** the manifest keeps child restore and pause cells unsupported and the planner rejects continuation before activation

#### Scenario: Child source cursor is unavailable
- **WHEN** the configured child owner cannot validate the checkpoint-bound source cursor
- **THEN** the manifest or admission reports unsupported/unavailable and does not infer support from another form
