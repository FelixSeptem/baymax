# agent-runtime-protocol-projection (production-ish)

## Purpose
Production-ish runtime semantic example for protocol projection with capability
negotiation, bounded context, concurrent-Run admission, source lineage, and
rejected-transition evidence.

## Variant Delta (vs minimal)
- Reuses the same real Run/Event/Checkpoint projection path, adds terminal-outcome projection, and adds a terminal-state transition rejection branch.
- Emits `protocol_invalid_transition_rejected` only when `completed -> working` is rejected by the canonical lifecycle validator.
- Keeps the same library-first boundary; the difference is behavior and evidence, not marker-only parameterization.

## Run
`go run ./examples/agent-modes/agent-runtime-protocol-projection/production-ish`

## Prerequisites
- Go 1.26+ with repository dependencies available.
- No hosted control plane or external persistence service.

## Real Runtime Path
- Semantic anchor: `agent_runtime_protocol.capability_context_admission_terminal_outcome`
- Classification: `agent_runtime_protocol.projection`
- Runtime path evidence: `core/types,core/runner,orchestration/scheduler,observability/event,observability/trace,orchestration/snapshot`
- Related contract: `agent-runtime-protocol-contract`
- Required gates: `check-agent-runtime-protocol-contract.*`, `check-terminal-outcome-contract.*`
- Replay fixture: `agent_runtime_protocol.v1`

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.anchor=agent_runtime_protocol.capability_context_admission_terminal_outcome`
- `verification.semantic.expected_markers=protocol_run_mapped,protocol_event_mapped,protocol_checkpoint_mapped,protocol_descriptor_validated,protocol_context_validated,protocol_admission_projected,terminal_outcome_projected,protocol_invalid_transition_rejected`
- one line per marker: `verification.semantic.marker.<token>=ok`

## Failure/Rollback Notes
- The production-ish variant must differ through a real invalid-transition branch, not marker-only output.
- Run the protocol gate to classify mapping, lineage, and control-plane drift.
- Terminal outcome fields remain additive and can be ignored for rollback to the legacy projection.
- Revert this directory as one unit if the example contract changes.
