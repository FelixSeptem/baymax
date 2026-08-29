# Checkpoint History and Workspace Provenance Design

## Decision

Implement the roadmap P3 direction `extend-agent-runtime-protocol-with-checkpoint-history-and-workspace-provenance` using the protocol-projection approach. The first scope is deliberately minimal: checkpoint lineage/history/branch/replay references plus bounded workspace change-set and integrity references.

## Architecture

`core/types` owns additive DTOs and validators. `orchestration/snapshot` remains the manifest and restore fact source and exposes pure projection helpers. Composer and source runtimes supply recovery context; they retain restore, workspace, sandbox, and policy ownership. Diagnostics and OTel additions remain nullable and flow through `RuntimeRecorder`.

## Lifecycle

Root and derived checkpoint references are validated through parent/history links. Branches require a stable parent and branch identifier. Replay keys are idempotent and conflicting normalized data is rejected. Strict/compatible restore behavior is unchanged; workspace integrity drift is classified and returned to source policy without file mutation. Run and Stream share one normalization path.

## Verification

The implementation will add unit, integration, replay, gate, documentation-first example, and full repository verification coverage. No second state store, workspace filesystem, artifact content service, ACL, hosted control plane, or global queue is permitted.
