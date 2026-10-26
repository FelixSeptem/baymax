## ADDED Requirements

### Requirement: Manifest SHALL declare composite context assembly and restore independently

The versioned capability manifest SHALL declare host context projection assembly and correlated handoff/restore independently for runner/direct, child/fanout, team, workflow, and mailbox forms. Each supported cell MUST include source owner/namespace, supported native-node scope, projection and continuation contract/API versions, build/module revision, deployment profile, resolver reauthorization requirement, retention/expiry policy, and a concrete fixture or contract-test evidence reference. Missing or incompatible evidence MUST remain unsupported or unavailable with a stable reason. A supported context cell MUST NOT imply execution resume, source history replay, or business authorization/commit.

#### Scenario: Manifest enables only an evidenced source form
- **WHEN** a planner reads a manifest with valid context binding evidence for team but not mailbox
- **THEN** it may admit the declared team context path while mailbox context assembly remains unsupported/unavailable

#### Scenario: Context manifest does not imply execution restore
- **WHEN** a form supports context projection but lacks matching continuation restore evidence
- **THEN** the manifest reports context assembly separately and rejects cross-process context restore before checkpoint claim or source activation

#### Scenario: Lifecycle-only mailbox evidence cannot promote context cells
- **WHEN** mailbox evidence proves delivery, lease, retry, DLQ, or terminal lifecycle projection but does not expose a context-aware handler safe point
- **THEN** `host_context_assembly` and `host_context_handoff_restore` remain unsupported for `mailbox_async`
- **AND** the manifest MUST NOT infer context support from mailbox continuation, history, or planbridge package presence

#### Scenario: Deployment scope remains bounded
- **WHEN** the only configured context/continuation evidence is a same-host single-writer FileStore
- **THEN** the manifest declares sequential reopen scope and does not advertise multi-writer or cross-machine projection restore
