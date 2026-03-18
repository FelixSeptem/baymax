## ADDED Requirements

### Requirement: Composer-managed timeline SHALL preserve canonical multi-agent namespaces
Composer-managed timeline events MUST preserve canonical reason namespaces (`team.*`, `workflow.*`, `a2a.*`, `scheduler.*`, `subagent.*`) and MUST NOT emit non-canonical multi-agent reasons.

#### Scenario: Composer executes mixed orchestration flow
- **WHEN** composer executes a flow involving workflow, teams, A2A, and scheduler
- **THEN** emitted timeline reasons stay within canonical namespaces and pass shared contract gate checks

### Requirement: Composer-managed scheduler paths SHALL carry required correlation fields
For scheduler-managed timeline events under composer execution, events MUST include `task_id` and `attempt_id` where scheduler correlation is required by contract.

#### Scenario: Composer path emits scheduler claim and terminal events
- **WHEN** composer emits scheduler claim/requeue/join-related timeline events
- **THEN** each required scheduler event includes `task_id` and `attempt_id` correlation fields
