# checkpoint-workspace-provenance (minimal)

## Purpose

Minimal runtime example for checkpoint lineage and bounded workspace provenance.

## Run

`go run ./examples/agent-modes/checkpoint-workspace-provenance/minimal`

## Prerequisites

- Go 1.26+ with repository dependencies available.

## Real Runtime Path

- Semantic anchor: `agent_runtime_protocol.checkpoint_history_workspace_provenance`.
- Runtime path evidence: `core/types,orchestration/snapshot`.
- Required gate: `check-agent-runtime-protocol-contract.*`.
- Replay fixture: `agent_runtime_protocol_checkpoint_provenance.v1`.

## Expected Output/Verification

- `verification.semantic.anchor=agent_runtime_protocol.checkpoint_history_workspace_provenance`
- Markers: `checkpoint_history_projected,checkpoint_lineage_validated,workspace_provenance_projected`.

## Failure/Rollback Notes

- Provenance projection is reference-only and does not mutate snapshots or workspace files.
- Roll back by omitting optional provenance fields; legacy checkpoint references remain valid.
