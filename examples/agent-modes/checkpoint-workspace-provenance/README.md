# checkpoint-workspace-provenance

P3 documentation-first baseline for the Agent Runtime Protocol checkpoint history and workspace provenance projection.

## Purpose

Real runtime semantic example for checkpoint lineage, replay identity, and bounded workspace integrity provenance.

## Run

`go run ./examples/agent-modes/checkpoint-workspace-provenance/minimal`

Production-ish verification:

`go run ./examples/agent-modes/checkpoint-workspace-provenance/production-ish`

## Prerequisites

- Go 1.26+ with repository dependencies available.
- No hosted service, external network, state store, workspace filesystem, or artifact content service is required.

- Semantic anchor: `agent_runtime_protocol.checkpoint_history_workspace_provenance`
- Runtime path: `core/types,orchestration/snapshot,orchestration/composer,observability/event,tool/diagnosticsreplay`
- Contract: checkpoint history/lineage/branch/replay and bounded workspace change-set/integrity references.
- Restore behavior: strict restore rejects schema, lineage, or integrity drift before source mutation; compatible restore preserves the existing bounded downgrade semantics.
- Rollback: omit the optional provenance projection and continue using the existing `CheckpointRef`/snapshot restore path. No workspace files or snapshot manifests are mutated by the protocol projection.

Executable minimal and production-ish variants now exercise pure protocol projection and validation only; neither writes a snapshot nor mutates a workspace.

## Expected markers

Minimal: `checkpoint_history_projected`, `checkpoint_lineage_validated`, `workspace_provenance_projected`.

Production-ish: the minimal markers plus `checkpoint_replay_idempotent`, `workspace_integrity_drift_classified`, `governance_checkpoint_provenance_replay_bound`.

## Real Runtime Path

- Semantic anchor: `agent_runtime_protocol.checkpoint_history_workspace_provenance`.
- Runtime path evidence: `core/types,orchestration/snapshot,orchestration/composer,observability/event,tool/diagnosticsreplay`.
- Related contracts: `checkpoint-history-and-workspace-provenance`, `agent-runtime-protocol-contract`, `unified-state-and-session-snapshot-contract`, `diagnostics-replay-tooling`.
- Required gates: `check-agent-runtime-protocol-contract.*`, `check-state-snapshot-contract.*`.
- Replay fixture: `agent_runtime_protocol_checkpoint_provenance.v1`.

## Expected Output/Verification

- `verification.semantic.anchor=agent_runtime_protocol.checkpoint_history_workspace_provenance`
- `verification.semantic.expected_markers=...`
- `result.checkpoint_id=checkpoint-derived`

## Failure and recovery

The source snapshot/workspace owners remain authoritative. Expected deterministic classifications include `checkpoint.lineage_missing_parent`, `checkpoint.history_disconnected`, `checkpoint.replay_conflict`, `workspace.provenance_missing`, `workspace.association_mismatch`, and `workspace.integrity_drift`.

## Failure/Rollback Notes

- Strict restore rejects schema, lineage, or integrity drift before source mutation; compatible restore preserves bounded downgrade behavior.
- If markers or replay classifications drift, run the protocol and snapshot gates and inspect the pure projection helpers.
- Rollback by omitting optional provenance projection; the existing checkpoint and snapshot path remains valid.
