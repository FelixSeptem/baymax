# checkpoint-workspace-provenance (production-ish)

Production-ish runtime semantic example for checkpoint history and workspace provenance, including replay idempotency and integrity drift classification.

## Variant Delta (vs minimal)

- Reuses the same pure checkpoint lineage and workspace provenance projection path.
- Adds a real replay-idempotency assertion and an integrity-drift rejection branch.
- Keeps snapshot, workspace, and policy ownership outside the protocol projection.

## Run

`go run ./examples/agent-modes/checkpoint-workspace-provenance/production-ish`

## Prerequisites

- Go 1.26+ with repository dependencies available.
- No hosted service, external persistence, workspace filesystem, or artifact content service.

## Real Runtime Path

- Semantic anchor: `agent_runtime_protocol.checkpoint_history_workspace_provenance`.
- Classification: `agent_runtime_protocol.checkpoint_workspace_provenance`.
- Runtime path evidence: `core/types,orchestration/snapshot,orchestration/composer,observability/event,observability/trace,tool/diagnosticsreplay`.
- Related contracts: `checkpoint-history-and-workspace-provenance`, `agent-runtime-protocol-contract`, `unified-state-and-session-snapshot-contract`, `diagnostics-replay-tooling`.
- Required gates: `check-agent-runtime-protocol-contract.*`, `check-state-snapshot-contract.*`.
- Replay fixture: `agent_runtime_protocol_checkpoint_provenance.v1`.

## Expected Output/Verification

- `verification.semantic.anchor=agent_runtime_protocol.checkpoint_history_workspace_provenance`
- `verification.semantic.expected_markers=checkpoint_history_projected,checkpoint_lineage_validated,workspace_provenance_projected,checkpoint_replay_idempotent,workspace_integrity_drift_classified,governance_checkpoint_provenance_replay_bound`
- `result.checkpoint_id=checkpoint-derived`

## Failure/Rollback Notes

- Drift is classified before source mutation; the example does not roll back workspace files.
- Run the protocol and snapshot gates when lineage, replay, or integrity markers diverge.
- Revert this directory as one unit; existing snapshot and checkpoint references remain valid.
