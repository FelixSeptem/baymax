## Why

Baymax already implements most production Agent Runtime mechanics across `core/runner`, orchestration, A2A, MCP, snapshots, realtime recovery, diagnostics, tracing, and evaluation. Those mechanics are exposed as subsystem-specific DTOs and events, however, so an embedding host, frontend, audit system, or peer runtime cannot rely on one stable lifecycle vocabulary across implementations.

The protocol-object analysis identifies a focused opportunity: define a library-first protocol projection for stable task-lifecycle objects while preserving the existing Runtime implementations and their module boundaries. This is timely because the supporting contracts are now stable and no OpenSpec change is active.

## What Changes

- Add a canonical Agent Runtime Protocol object model for embedded consumers: `session`, `run`, `step`, `event`, `artifact`, and `checkpoint` references and their required correlation fields.
- Freeze a minimal public Run lifecycle state machine, including `submitted`, `working`, `input_required`, `completed`, `failed`, and `canceled`, plus valid cancel, retry, and resume transitions.
- Define deterministic mappings from existing Runner, Workflow, Teams, Scheduler, A2A, Realtime, Snapshot, and RuntimeRecorder semantics into the canonical protocol objects without replacing module-native contracts.
- Define canonical event-envelope mapping, event ordering/idempotency rules, and Run/Stream semantic-equivalence expectations by reusing the existing realtime and timeline semantics.
- Define reference-only Artifact lineage (`id`, `type`, `locator`, `digest`, producing run/step) and Checkpoint references; this change does not introduce an artifact store or hosted session service.
- Define the Error-as-Data boundary: recoverable tool or business failures may be represented as protocol step/event outcomes, while security, configuration, protocol, and module-boundary violations remain fail-fast.
- Add protocol replay fixtures, contract tests, a dedicated gate, documentation, and an agent-mode example that demonstrates a real protocol projection path.

## Example Impact Assessment

- 新增示例

## Capabilities

### New Capabilities

- `agent-runtime-protocol-contract`: Defines the library-first canonical task-lifecycle object model, state machine, correlation, event, artifact, checkpoint, and error-boundary contract.

### Modified Capabilities

- `realtime-event-protocol-and-interrupt-resume-contract`: Makes the existing realtime envelope and resume cursor a canonical protocol-event and Run lifecycle mapping, without changing realtime transport semantics.
- `unified-state-and-session-snapshot-contract`: Adds canonical checkpoint-reference and protocol correlation requirements while retaining existing segmented snapshot ownership.
- `runtime-otel-tracing-and-agent-eval-interoperability-contract`: Adds canonical protocol correlation fields and artifact/step lineage to the existing trace and evaluation interoperability surface.
- `a2a-minimal-interoperability`: Requires A2A Task lifecycle and correlation fields to map deterministically to the canonical protocol Run/Step/Event model.
- `go-quality-gate`: Adds the Agent Runtime Protocol contract gate with shell/PowerShell parity and replay coverage.

## Impact

- Public library DTOs under `core/types`, plus translation adapters in the owning runtime/orchestration modules.
- Existing contracts in `core/runner`, `orchestration/*`, `a2a`, `observability/event`, `observability/trace`, `runtime/diagnostics`, and `tool/diagnosticsreplay`.
- Documentation: `README.md`, `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, and `docs/mainline-contract-test-index.md`.
- New contract and replay coverage, a quality-gate script pair, and a new `examples/agent-modes` protocol-projection example.
- Non-goals: no hosted control plane, REST/SSE gateway, platform UI/RBAC, multi-tenant service, generic artifact storage, or replacement of A2A/MCP/Realtime/Snapshot module contracts.
