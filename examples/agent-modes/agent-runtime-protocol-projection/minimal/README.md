# agent-runtime-protocol-projection (minimal)

## Purpose
Real runtime semantic example for the Agent Runtime Protocol projection contract,
including capability discovery, bounded Session context, and source-owned
concurrent-Run admission projection.

## Run
`go run ./examples/agent-modes/agent-runtime-protocol-projection/minimal`

## Prerequisites
- Go 1.26+ with repository dependencies available.
- No hosted service, external network, session server, or artifact store.

## Real Runtime Path
- Semantic anchor: `agent_runtime_protocol.capability_context_admission`.
- Classification: `agent_runtime_protocol.projection`.
- Runtime path evidence: `core/types,core/runner,orchestration/scheduler,observability/event,orchestration/snapshot`.
- Related contract: `agent-runtime-protocol-contract`.
- Required gate: `check-agent-runtime-protocol-contract.*`.
- Replay fixture: `agent_runtime_protocol.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.anchor=agent_runtime_protocol.capability_context_admission`
- `verification.semantic.expected_markers=protocol_run_mapped,protocol_event_mapped,protocol_checkpoint_mapped,protocol_descriptor_validated,protocol_context_validated,protocol_admission_projected`
- one line per marker: `verification.semantic.marker.<token>=ok`

## Failure/Rollback Notes
- If mapping or descriptor/context/admission markers are missing, run the protocol gate and inspect `core/types/protocol.go` source projections.
- If replay fails, inspect `agent_runtime_protocol.v1` fixture normalization and drift classification.
- For rollback, revert the example directory together; the core protocol contract remains independently testable.
