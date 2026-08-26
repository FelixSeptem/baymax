## Why

The archived Agent Runtime Protocol contract establishes a stable vocabulary for `Session`, `Run`, `Step`, `Event`, `Artifact`, and `Checkpoint`, but its `SessionRef` and `RunRef` surface does not yet tell a host what context, participants, capabilities, or concurrent-run behavior the runtime supports. This leaves clients unable to discover legal protocol actions or interpret two Runs submitted against the same Session without learning Baymax-specific internals.

This is the next bounded gap identified by the Agent Runtime Protocol comparison. It strengthens the external contract while preserving Baymax's library-first architecture: existing adapter capability negotiation, policy precedence, readiness admission, scheduler, snapshot, and source-runtime owners remain authoritative.

## What Changes

- Extend the Agent Runtime Protocol with a versioned `ProtocolDescriptor` that separates runtime discovery metadata from required and optional capability declarations.
- Add bounded Session context metadata for participants, agent identity, context scope, and host-visible capabilities without embedding conversation bodies or provider-specific objects.
- Define protocol-visible Run admission semantics for concurrent Runs targeting the same Session, including explicit `reject`, `serialize`, `branch`, and `optimistic` policy declarations and deterministic admission outcomes.
- Express host actions supported by a runtime, including whether `cancel`, `resume`, `retry`, and capability-specific actions are available under the current profile.
- Reuse adapter capability negotiation's required/optional capability model, fail-fast/default strategy, profile versioning, downgrade reasons, and Run/Stream equivalence rules.
- Preserve existing Run lifecycle, Realtime interrupt/resume, Snapshot restore, policy precedence, readiness admission, and fail-fast security/configuration boundaries.
- Add positive, negative, compatibility, concurrency, replay, and Run/Stream parity coverage plus documentation and example updates.

## Example Impact Assessment

修改示例

The existing `agent-runtime-protocol-projection` example will expose the descriptor/context/admission projection and document unsupported capability and concurrent-Run outcomes.

## Capabilities

### New Capabilities

None. This change extends the existing Agent Runtime Protocol capability rather than creating a parallel protocol vocabulary.

### Modified Capabilities

- `agent-runtime-protocol-contract`: add protocol descriptor and capability negotiation, bounded Session context metadata, host action availability, and explicit same-Session concurrent-Run admission semantics while keeping the existing lifecycle object and state machine compatible.

## Impact

- `core/types`: additive public protocol DTOs, capability/profile validation, Session context and Run admission outcome types.
- Protocol mapping adapters and host-facing projection helpers: expose descriptor/context/admission data without becoming an execution owner.
- Existing adapter capability negotiation: reused as the canonical required/optional capability and fallback vocabulary; no duplicate taxonomy.
- Scheduler/composer or other source runtimes: optional projection hooks only; their queue, branch, conflict, and persistence behavior remains source-owned.
- Diagnostics, OTel, replay fixtures, and contract gates: additive correlation and deterministic admission classifications, written through `RuntimeRecorder` where diagnostics are involved.
- `examples/agent-modes/agent-runtime-protocol-projection`: documentation-first update and executable assertions for supported/unsupported capabilities and concurrent Run outcomes.
- No new provider SDK dependency, MCP transport dependency, hosted service, session repository, artifact store, RBAC model, tenant model, or remote scheduler.
