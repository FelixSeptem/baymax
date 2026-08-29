## ADDED Requirements

### Requirement: Replay tooling SHALL validate checkpoint and workspace provenance fixtures

Diagnostics replay tooling MUST support a versioned `agent_runtime_protocol_checkpoint_provenance.v1` fixture containing checkpoint history, lineage, branch/replay identity, restore source, workspace integrity references, and expected Run/Stream-normalized outcomes. Malformed or unsupported fixtures MUST fail fast with deterministic validation reasons.

#### Scenario: Canonical provenance fixture replays successfully
- **WHEN** tooling receives a valid provenance fixture whose normalized output matches expectations
- **THEN** replay succeeds deterministically without live runtime connectivity

#### Scenario: Malformed provenance fixture fails fast
- **WHEN** tooling receives missing required fixture fields or an unsupported fixture version
- **THEN** replay exits with deterministic schema validation classification and no partial success

### Requirement: Provenance replay drift SHALL use canonical classifications

Replay MUST classify at minimum `checkpoint_lineage_drift`, `checkpoint_schema_drift`, `checkpoint_branch_drift`, `checkpoint_replay_drift`, `workspace_provenance_drift`, `workspace_integrity_drift`, and `run_stream_provenance_parity_drift`.

#### Scenario: Replay detects workspace integrity drift
- **WHEN** actual normalized before/after integrity or drift decision differs from the fixture
- **THEN** replay fails with `workspace_integrity_drift`

#### Scenario: Replay detects lineage or branch drift
- **WHEN** parent, history order, branch conflict, or replay identity differs from canonical expectation
- **THEN** replay fails with the corresponding deterministic checkpoint drift classification

#### Scenario: Historical fixtures remain compatible
- **WHEN** provenance fixtures run alongside archived `agent_runtime_protocol.v1` and snapshot fixtures
- **THEN** all fixture generations remain parseable and deterministic
