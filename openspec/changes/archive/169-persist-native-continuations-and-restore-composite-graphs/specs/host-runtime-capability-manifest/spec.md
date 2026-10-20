## ADDED Requirements

### Requirement: Manifest SHALL expose checkpoint and pause support by form

Capability entries SHALL separately declare checkpoint creation, cross-process restore, `input_required`, `approval_required`, and `budget_exhausted` for direct, child/fanout, team, workflow, and mailbox forms. Each supported cell MUST reference source evidence, contract version, build/module revision, public API version, and retention/expiry limitation; absent evidence is unsupported.

#### Scenario: Direct durable cell is source-evidenced
- **WHEN** a durable direct store and restore path are configured
- **THEN** the manifest marks only the verified pause cells supported and identifies their evidence

#### Scenario: Composite cells remain unsupported
- **WHEN** a child/team/workflow/mailbox owner has no graph restore evidence
- **THEN** each corresponding cell is explicitly `unsupported_capability` with a stable limitation

### Requirement: Admission SHALL reject unsupported restore before side effects

Planner and host admission MUST validate the required form, pause reason, contract version, and current source availability before enqueue, publish, Run, Stream, retry, or restore. Manifest values MUST NOT be inferred from package presence, DTO compilation, terminal-only behavior, or historical event replay.

#### Scenario: Required unsupported cell fails closed
- **WHEN** a plan requires `approval_required` on a form whose manifest marks it unsupported
- **THEN** admission returns typed `unsupported_capability` before source activation and does not substitute direct execution
