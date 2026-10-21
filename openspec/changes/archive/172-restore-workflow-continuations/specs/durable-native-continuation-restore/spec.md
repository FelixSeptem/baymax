## ADDED Requirements

### Requirement: Workflow continuation SHALL be source-owned and durable

The runtime SHALL permit a workflow source owner to create a durable continuation checkpoint only at a declared workflow pause boundary. The checkpoint SHALL bind graph identity, graph version and digest, original parent/run/session/causation identities, step dependencies, pending and completed steps, consumed per-step budget, pending action, completed effect references, terminal arbitration evidence, pause reason, and predecessor/successor checkpoint identity. A new Engine reopening the configured source store SHALL restore that same native workflow graph and SHALL NOT allocate a replacement parent Run, enqueue a replacement workflow, or use history replay as restore material.

#### Scenario: New Engine restores the original workflow graph
- **WHEN** a workflow checkpoint is durably committed and a new Engine opens the same compatible source store
- **THEN** the Engine restores the original graph and native identities from its source-owned continuation material
- **AND** it does not rerun terminal steps or completed effects

### Requirement: Workflow restore admission SHALL fail closed

Workflow resume admission MUST validate checkpoint digest, schema, expiry, source/run/session/parent binding, graph version and digest, ownership, authority/plan/write-scope/effect namespace, occurrence and causation identity, current budget facts, predecessor/successor relation, and required capability subset before source activation. Foreign, stale, conflicting, expired, corrupt, unavailable, or incompatible checkpoints SHALL return typed failure and SHALL NOT fall back to Run, enqueue, retry, latest state, or side-effecting replay.

#### Scenario: Foreign checkpoint is rejected before graph activation
- **WHEN** a resume request presents a checkpoint bound to a different workflow source or graph identity
- **THEN** admission returns a typed foreign checkpoint result before a step, tool, provider, or scheduler is activated

### Requirement: Workflow checkpoint claims SHALL converge duplicate resumes

The workflow source SHALL atomically claim a valid checkpoint using the full normalized resume identity. Exact duplicate resume requests SHALL return the retained admissible or terminal projection for the same logical continuation. A request with a conflicting mode, pause reason, predecessor, request identity, or binding SHALL fail closed and SHALL NOT consume the checkpoint or start a second logical continuation.

#### Scenario: Duplicate workflow resume returns one projection
- **WHEN** two equivalent resume requests race for the same valid workflow checkpoint
- **THEN** exactly one source continuation is admitted
- **AND** both callers observe the same retained logical projection

### Requirement: Unsupported composite forms SHALL remain explicit

Child/fanout, team parallel/vote, mailbox host-plan continuation, and planbridge resumed execution SHALL remain `unsupported_capability` for durable composite restore until their own source owners declare and evidence equivalent continuation material. Workflow support SHALL NOT be used to infer support for another composite form.

#### Scenario: Non-workflow composite restore is rejected
- **WHEN** a caller requests cross-process restore for child, team, mailbox, or planbridge resumed execution
- **THEN** admission returns `unsupported_capability` before source activation
