## MODIFIED Requirements

### Requirement: Replay tooling SHALL validate checkpoint and workspace provenance fixtures

Diagnostics replay tooling MUST support a versioned `agent_runtime_protocol_checkpoint_provenance.v1` fixture containing checkpoint history, session-history root/leaf references when available, lineage, branch/replay identity, restore source, workspace integrity references, and expected Run/Stream-normalized outcomes. Malformed or unsupported fixtures MUST fail fast with deterministic validation reasons. Replay MUST remain offline, read-only, and MUST NOT execute tools/providers or mutate snapshot, workspace, session, or checkpoint state.

#### Scenario: Canonical provenance fixture replays successfully
- **WHEN** tooling receives a valid provenance fixture whose normalized history, checkpoint, workspace, and Run/Stream output matches expectations
- **THEN** replay succeeds deterministically without live runtime connectivity or side effects

#### Scenario: Malformed provenance fixture fails fast
- **WHEN** tooling receives missing required fixture fields, broken history continuity, or an unsupported fixture version
- **THEN** replay exits with deterministic schema or lineage validation classification and no partial success

#### Scenario: Replay side-effect attempt is blocked
- **WHEN** replay processing attempts to invoke a provider/tool or mutate source state
- **THEN** replay fails with `session.replay_side_effect` and leaves all source state unchanged

### Requirement: Provenance replay drift SHALL use canonical classifications

Replay MUST classify at minimum `checkpoint_lineage_drift`, `checkpoint_schema_drift`, `checkpoint_branch_drift`, `checkpoint_replay_drift`, `workspace_provenance_drift`, `workspace_integrity_drift`, `run_stream_provenance_parity_drift`, `session.history_gap`, `session.checkpoint_association_mismatch`, and `run_stream_history_replay_parity_drift`.

#### Scenario: Replay detects workspace integrity drift
- **WHEN** actual normalized before/after integrity or drift decision differs from the fixture
- **THEN** replay fails with `workspace_integrity_drift`

#### Scenario: Replay detects lineage or branch drift
- **WHEN** parent, history order, branch conflict, session association, or replay identity differs from canonical expectation
- **THEN** replay fails with the corresponding deterministic checkpoint or session-history drift classification

#### Scenario: Historical fixtures remain compatible
- **WHEN** provenance fixtures run alongside archived `agent_runtime_protocol.v1` and snapshot fixtures without session-history extensions
- **THEN** all fixture generations remain parseable and deterministic
