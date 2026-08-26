## Context

## Example Impact Assessment

- 新增示例

Baymax is a library-first Go runtime. Its existing contracts already provide the implementation facts needed for a production Agent Runtime: `RunRequest.RunID` and `SessionID`, Runner Run/Stream loops, Workflow/Teams/Scheduler/A2A task execution, realtime interrupt/resume envelopes, segmented state/session snapshots, RuntimeRecorder-backed diagnostics, OTel semantic spans, and evaluation replay.

The facts are intentionally owned by their existing modules. A new public protocol must therefore be a projection layer, not a second state store, scheduler, event bus, or transport service. It must respect fixed dependency rules: `runtime/*` cannot import MCP transports, non-MCP packages cannot import MCP internals, and diagnostics writes remain exclusive to `observability/event.RuntimeRecorder`.

### Source Ownership and One-Way Mapping Inventory

| Source fact | Owner | Canonical projection | Preserved source semantics |
| --- | --- | --- | --- |
| `RunRequest.RunID`, `RunRequest.SessionID`, `RunResult` | `core/types`, `core/runner` | `RunRef`, model/tool `StepRef` | Run/Stream loop, policy, retries, terminal result |
| `workflow.Step` and `StepStatus` | `orchestration/workflow` | workflow `StepRef` | DAG, dependency, retry, timeout, checkpoint/resume |
| `teams.Plan`, `teams.Task`, `TaskStatus` | `orchestration/teams` | team-task `StepRef` | strategy, role, local/remote dispatch, aggregation |
| `scheduler.Task`, `Attempt`, `TaskRecord` | `orchestration/scheduler` | scheduler-attempt `StepRef` | lease, retry, DLQ, parent Run relation |
| `a2a.TaskRecord`, Agent Card | `a2a` | A2A `RunRef`/`StepRef` | peer lifecycle, version/delivery negotiation, correlation |
| `RealtimeEventEnvelope` | `core/types`, `core/runner` | realtime `EventEnvelope`, `input_required` lifecycle | `seq`, dedup key, cursor, interrupt/resume taxonomy |
| `types.Event`, Action Timeline | `core/types`, `observability/event` | source-scoped `EventEnvelope` | standard event dispatch and RuntimeRecorder single-write path |
| `IsolateHandoffArtifact` | `context/assembler` | `ArtifactRef` | deferred content/body and reference-first context policy |
| `snapshot.Manifest` | `orchestration/snapshot` | `CheckpointRef` | schema, segments, digest, strict/compatible import, idempotency |
| OTel semantic spans | `observability/trace` | protocol correlation attributes | existing topology and exporter semantics |

## Goals / Non-Goals

**Goals:**

- Provide embedded consumers one canonical vocabulary for the stable lifecycle objects that span existing Baymax Runtime modules.
- Preserve each source module as the only owner of execution, recovery, persistence, transport, and diagnostics behavior.
- Make lifecycle correlation, state transitions, event ordering, artifact lineage, checkpoint references, error classification, replay, and Run/Stream parity contract-testable.
- Reuse existing A2A, realtime, snapshot, tracing, evaluation, tool, and reason-taxonomy contracts as upstream facts.

**Non-Goals:**

- No hosted REST, SSE, WebSocket, JSON-RPC, gRPC, Redis Stream, session repository, artifact store, control plane, UI, RBAC, tenant model, or remote scheduler.
- No replacement of A2A Task, MCP Tool/Resource/Prompt, realtime event, snapshot manifest, Scheduler task, or module-native event semantics.
- No single mandatory loop container or multi-agent topology; Runner, Workflow, Teams, Scheduler, and A2A remain independently selectable Runtime implementations.
- No conversion of security, configuration, contract, or module-boundary failures into model-visible recoverable data.

## Decisions

### 1. Add a protocol projection package in `core/types`

The canonical DTOs and pure validation/state-transition helpers will live in `core/types`, which already owns cross-module DTOs and is safe for Runner, orchestration, A2A, observability, and host consumers to import. The first surface will include `SessionRef`, `RunRef`, `StepRef`, `EventEnvelope`, `ArtifactRef`, and `CheckpointRef`.

The DTOs will carry correlation fields rather than owning data: `session_id`, `run_id`, `step_id`, `parent_step_id`, `event_id`, `causation_id`, `artifact_id`, and `checkpoint_id`. Values not available from a particular subsystem remain omitted, preserving additive + nullable + default compatibility.

Alternative considered: create `runtime/protocol`. Rejected because that would force orchestration/A2A consumers to depend downward on `runtime/*` and would blur the existing configuration/diagnostics ownership boundary.

### 2. Freeze one small Run state machine and map, do not replace

The public states are `submitted`, `working`, `input_required`, `completed`, `failed`, and `canceled`. `resume` is valid only from `input_required`; `retry` creates or activates a new Run attempt under a defined causation relationship rather than mutating a terminal Run; `cancel` is idempotent and terminal.

Runner, A2A, Scheduler, Workflow, Teams, and realtime remain owners of their richer internal states. Mapping adapters normalize those states into the public state machine. The protocol does not require all sources to expose every state, and does not make a Session a lock or define global concurrent-Run policy.

Alternative considered: make Scheduler Task states the shared canonical state machine. Rejected because a Scheduler Task is a subagent dispatch primitive, not the public boundary for every single-agent Run.

### 3. Reuse existing event paths through a canonical envelope

Protocol event mapping accepts existing `types.Event`, timeline events, realtime envelopes, and A2A lifecycle events. It emits a normalized envelope with correlation identifiers, canonical event kind, timestamp, sequence when available, payload, and stable source metadata.

Realtime retains authority over `event_id`, ordered `seq`, deduplication, resume cursor, and its `request|delta|interrupt|resume|ack|error|complete` taxonomy. Other sources map to protocol event kinds without claiming realtime ordering guarantees. RuntimeRecorder remains the only diagnostics writer; adapters only produce standard events for existing dispatch/recording paths.

Alternative considered: replace all existing event DTOs with a new event bus. Rejected because it would create parallel event semantics and risk breaking Run/Stream, A2A, and diagnostics replay contracts.

### 4. Model Artifact and Checkpoint as references only

`ArtifactRef` contains an ID, type, locator, optional digest, producing Run ID, and producing Step ID. `CheckpointRef` contains an ID, schema version, source component, optional Run/Session IDs, and immutable integrity reference. `orchestration/snapshot.Manifest` remains the source of snapshot structure, segment ownership, digest, `strict|compatible` restore, and import idempotency.

Existing isolate-handoff artifacts and snapshot manifests will be adapted into references. No first-phase payload persistence, ACL model, garbage collection, content lookup API, or artifact/file transport is introduced.

Alternative considered: add a generic content-addressed artifact store. Rejected as a platform data service outside the repository's library-first scope.

### 5. Partition failures by recovery authority

Recoverable tool and business failures can appear as failed Step outcomes or error Events only if their current owner already permits recovery. Security denials, configuration validation failures, protocol validation errors, compatibility conflicts, and boundary violations remain errors returned by their owning module and retain current fail-fast behavior.

This decision takes the useful portion of Error-as-Data without weakening Baymax's security and configuration contracts.

### 6. Build contract-first before adapters and examples

Implementation order is: protocol DTO/state-machine tests; mapping tests per source; observability/replay tests; adapters; docs; then the agent-mode example after its README baseline is complete. A new `agent_runtime_protocol.v1` fixture and dedicated shell/PowerShell gate will assert valid mappings plus invalid transitions, missing correlations, ordering/dedup drift, lineage drift, and control-plane absence.

## Risks / Trade-offs

- [A broad DTO grows into a parallel Runtime] -> Keep the initial object model reference-only; each added field must name its source owner and have an additive contract/replay test.
- [State normalization loses subsystem detail] -> Preserve `source` and module-native status/reason metadata in mapped payloads; require no reverse mapping.
- [Realtime ordering semantics are incorrectly generalized] -> Only expose monotonic sequence guarantees when the source is realtime; other event sources have source-scoped ordering semantics.
- [Artifact work expands into storage] -> Reject content persistence, file serving, and retention controls in this change; expose only locators/digests/lineage.
- [Error-as-Data bypasses a fail-fast guard] -> Maintain an explicit failure-class allowlist and use negative tests for security/configuration/protocol failures.
- [New docs or examples drift from the contract] -> Add contract-index entries, replay fixtures, a dedicated gate, docs consistency coverage, and an agent-mode example with doc-first enforcement.

## Migration Plan

1. Add additive DTOs and mapping APIs without modifying existing public request/result types or stored snapshot schemas.
2. Map existing module outputs in opt-in adapters and run old/new behavior side by side in contract tests.
3. Add RuntimeRecorder/OTel/diagnostics correlation fields as additive + nullable + default values and retain source-specific fields.
4. Publish the mapping documentation and real example only after contract fixtures and gates pass.
5. Roll back by disabling/removing only protocol adapter wiring; existing Runner, A2A, Realtime, Snapshot, and diagnostics contracts remain independently functional because their source-of-truth behavior is unchanged.

## Open Questions

- Determine whether the first public mapping API should return values directly, write standard `types.Event` values, or offer both with one canonical implementation.
- Determine the minimal canonical event-kind taxonomy after inventorying the existing Runner, timeline, Scheduler, Workflow, Teams, and A2A event types.
- Determine whether a retry relationship requires an explicit `attempt` reference in phase one or can initially be represented through `causation_id` plus source metadata.
