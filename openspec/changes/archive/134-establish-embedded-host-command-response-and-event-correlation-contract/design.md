## Context

See `proposal.md` for motivation. Baymax already has canonical protocol references, descriptor/action discovery, Realtime envelopes, durable event-stream binding, terminal recovery, Action Gate and clarification resolvers, readiness/policy/sandbox governance, and RuntimeRecorder-backed diagnostics. The missing connection is operational: `Runner` exposes only synchronous `Run` and `Stream`, protocol actions validate availability without executing source operations, Realtime control events are consumed at entry, and `EventHandler.OnEvent` has no delivery result.

The design must preserve the repository's dependency and ownership rules. `core/runner` remains the Run/Stream owner; Realtime keeps sequence/cursor/dedup state; terminal recovery keeps authoritative outcome convergence; diagnostics writes use `RuntimeRecorder`; `orchestration/composer` remains glue; and no `runtime/*` package may depend on MCP transports.

## Example Impact Assessment

修改示例

Before code or task completion for the example, update `examples/agent-modes/MATRIX.md` and `examples/agent-modes/realtime-interrupt-resume/README.md` with the host protocol semantic anchor, runtime path, expected markers, rollback notes, contracts, replay fixture, and gates. Then extend the existing minimal and production-ish variants; do not create a parallel numbered example.

## Goals / Non-Goals

**Goals:**

- Give an embedded process one versioned contract for starting and controlling Runs, correlating admission responses with asynchronous events, answering existing HITL requests, and recovering observation after disconnect.
- Make active Run control process-local, bounded, race safe, and explicitly owned by the source Engine.
- Keep the host coordinator and JSONL binding replaceable without changing Runner, Realtime, HITL, terminal, or diagnostics semantics.
- Make framing, pending cleanup, backpressure, output purity, terminal races, and Run/Stream parity deterministically testable.

**Non-Goals:**

- No steering/follow-up message queues, arbitrary prompt injection into an active model step, or new compaction behavior.
- No network listener, hosted connection manager, remote Session/Artifact store, RBAC, tenant model, attachment/lease, or control plane.
- No second event ledger, cursor, terminal arbiter, Run persistence store, policy engine, or diagnostics writer.
- No provider/model discovery, credential storage, or provider SDK changes.

## Decisions

### 1. Split the implementation into contract, source control, coordinator, and binding layers

Host envelope DTOs, finite command/action kinds, validation, and normalized outcomes belong in `core/types` because they are transport-neutral cross-module contracts. `core/runner` owns a process-local active-control registry and per-Run controls. A new top-level `host` package coordinates commands, pending correlation, existing resolvers, event subscriptions, and terminal queries through injected interfaces. `host/jsonl` provides the first stdin/stdout binding.

Dependencies point inward: `host/jsonl -> host -> core/types` and injected Runner/control/event interfaces; `core/runner -> core/types`. The host packages do not import `runtime/diagnostics`, MCP transports, provider SDKs, or concrete persistence. `orchestration/composer` may expose optional construction glue but does not own host state.

Alternative considered: place the binding under `runtime/*`. Rejected because runtime is the lower-level configuration/diagnostics foundation and must not depend upward on Runner or orchestration. Alternative considered: place it under `mcp/stdio`. Rejected because this is not MCP and non-MCP consumers must not inherit MCP transport semantics.

### 2. Keep a per-Engine active Run registry, not a global service

Each Engine registers a control record by normalized `run_id` before externally controllable execution begins. The record contains only source-owned cancellation, a bounded Realtime control ingress when enabled, a lifecycle snapshot function, and immutable correlation. Registration of a duplicate active ID fails before model/tool work. The record is closed and removed after the authoritative terminal event/snapshot has been published.

Cancel uses the Run's derived context cancellation function. Interrupt/resume enters the Run-owned bounded control ingress and is processed at defined safe points in both Run and Stream loops using the existing Realtime validator/state. A terminal/control race is resolved once by the Run owner and exposed as `accepted`, `duplicate`, `already_terminal`, or `rejected`; the registry never writes a terminal state.

Retry is not implemented by reopening an active control. If a source advertises retry, the coordinator delegates a new causally related Run/attempt request to that source. A source without such an executor does not advertise retry.

Alternative considered: use `RuntimeRecorder.QueryRuns` as the registry. Rejected because diagnostics are bounded historical observations, not an active lifecycle owner. Alternative considered: a package-global registry. Rejected because it creates cross-Engine collisions, leaks lifetime, and behaves like an implicit service.

### 3. Use a finite command vocabulary and two-phase result model

The first profile includes `run.start`, advertised lifecycle action execution, `realtime.interrupt`, `realtime.resume`, `hitl.respond`, `events.subscribe`, and `run.get`. Every command receives one immediate admission response: `accepted`, `rejected`, or `duplicate`, with a stable reason when not accepted. Execution progress and business terminal outcomes are separate `runtime_event` envelopes.

The coordinator does not promise that an accepted command succeeds. For example, accepted cancel means the source cancellation signal was admitted; the authoritative terminal may still be `completed` if completion won the race. `run.get` and reconnect use terminal recovery and durable stream sources rather than coordinator memory.

Alternative considered: return the complete Run result as the command response. Rejected because long-lived execution would couple request timeout to business lifetime and blur transport failure with Run failure.

### 4. Adapt existing HITL resolvers through reverse requests

The host coordinator provides resolver implementations that convert `ClarificationResolver.Resolve` and `ActionGateResolver.Confirm` calls into `host_request` envelopes keyed by the existing RequestID. A connection-scoped pending table waits for one matching `host_response`, enforces bounded payload and deadline, and atomically removes the entry on response, timeout, cancellation, write failure, or disconnect.

The adapter does not invent HITL outcomes. Clarification timeout/transport failure follows the existing `canceled_by_user` path; Action Gate timeout/transport failure remains fail closed. A disconnect closes pending waits but does not directly cancel other Runs. Duplicate or late responses return a deterministic rejection and cannot invoke a resolver twice.

Alternative considered: add a second asynchronous HITL state machine to the protocol. Rejected because current RequestID, timeout, timeline, denial, cancellation, and Run/Stream rules are already authoritative.

### 5. Isolate event delivery from Runner callbacks with bounded per-connection flow control

The coordinator converts existing source events to protocol envelopes and hands them to a bounded per-connection writer. The callback does not perform raw stdout writes and cannot block indefinitely. The writer serializes complete frames and applies the subscription's existing delivery policy. There is no global queue and no adapter-owned event history.

Negotiation, command responses, reverse HITL requests, responses, and authoritative terminal frames are non-droppable. If they cannot be delivered within the binding deadline, the connection closes, pending requests settle, and observation becomes disconnected; the business Run remains controlled by its source. Low-priority Runtime events may be dropped only through an existing source policy that records the drop via standard events and RuntimeRecorder. For the first profile, this is limited to durable Realtime `delta` events returned with both `delivery_policy=drop_with_record` and `source_outcome_declared=true`; a client request alone is insufficient. Direct Runner callbacks and all non-delta Realtime events remain critical, and absence of a standard EventSink disables dropping so failure stays visible.

Alternative considered: call `json.Encoder.Encode` directly from `EventHandler.OnEvent`. Rejected because a slow or panicking output path would synchronously block the Runner and make transport behavior part of execution semantics.

### 6. Define strict JSONL/stdio framing without a new global configuration domain

`host/jsonl` reads raw bytes until LF and decodes exactly one JSON object per frame. It does not treat Unicode `U+2028` or `U+2029` as delimiters. The default maximum frame is 1 MiB and may be changed through validated constructor options with an implementation-defined hard ceiling; invalid options fail construction before serving. This avoids adding runtime config or hot-update semantics for the first binding.

The connection negotiates one supported protocol profile before mutation commands. Stdout is injected as the exclusive protocol writer; logs use a separately injected writer, defaulting to stderr in the executable adapter. A single writer serializes frames, handles partial writes, waits for explicit backpressure bounds, and classifies write/flush failures. Tests use subprocess pipes to prove stdout purity and exit cleanup.

Alternative considered: `bufio.Scanner` with its default token limit and generic split behavior. Rejected because the limit is implicit and framing edge cases are not contract visible. Alternative considered: CBOR or length-prefix framing. Rejected because JSONL is inspectable and sufficient for the first local binding; Pi's remote CBOR protocol remains experimental and out of scope.

### 7. Reuse existing recovery, authorization, and observability contracts

An event subscription delegates cursor validation, catch-up/live-tail, overlap deduplication, expiry, gaps, disconnect, and backpressure classification to the durable binding and Realtime owners. Terminal queries delegate to terminal recovery. Command execution checks the protocol descriptor, readiness admission, and existing policy/sandbox authorization before source mutation.

Host correlation is emitted as standard bounded event payload fields. The first implementation does not add QueryRuns schema or a host-specific diagnostics store. RuntimeRecorder remains the sole diagnostics writer, and OTel additions, if required, are low-cardinality attributes derived from the same standard events.

Alternative considered: persist connection transcripts as a new event store. Rejected because replay fixtures cover contract verification while production history remains source-owned.

### 8. Treat transcript replay and subprocess conformance as separate gates

A versioned host transcript fixture validates normalized envelopes, correlations, admission decisions, HITL races, disconnect cleanup, cursor recovery, terminal convergence, and Run/Stream parity without a live model. A subprocess conformance harness validates LF framing, Unicode separators, maximum frame size, stdout purity, serialized writes, backpressure, EOF, exit code, and rejection of pending operations.

The dedicated shell and PowerShell gate asserts semantic parity and scans for forbidden hosted listeners, external stores, global queues, parallel terminal ownership, and steering/follow-up semantics. Existing Agent Runtime Protocol, Realtime, durable binding, terminal recovery, Action Gate, docs consistency, and quality gates remain mandatory.

## Risks / Trade-offs

- [Active registry leaks controls after panic or early return] -> Register through one scoped lifecycle guard, recover/close at the Engine boundary, remove exactly once, and test panic, cancellation, and terminal races.
- [Control ingress changes model/tool loop timing] -> Process only at documented safe points, use a bounded ingress, and assert semantic rather than nanosecond event-order parity across Run/Stream.
- [Accepted cancel is mistaken for canceled terminal] -> Keep admission and Runtime event envelopes distinct and test completion-wins/cancel-wins races.
- [Connection backpressure blocks execution] -> Use bounded per-connection delivery, explicit deadlines and policy outcomes; never write stdout in the Runner callback.
- [HITL disconnect weakens fail-closed behavior] -> Delegate final outcome to existing resolver deadlines and require Action Gate transport loss to deny execution.
- [Host package grows into a remote control plane] -> Gate out listeners, tenancy, remote persistence, connection management services, attachment/lease, and authorization ownership.
- [JSONL payload or logs corrupt stdout] -> Single protocol writer, separate log writer, frame bounds, subprocess purity tests, and fail-fast malformed-frame handling.

## Migration Plan

1. Add contract DTOs, validators, fixtures, and negative tests without wiring them into Engine execution.
2. Add the per-Engine active control registry and safe-point Realtime ingress behind opt-in construction; existing Run/Stream callers retain current behavior when no host control is requested.
3. Add the transport-neutral coordinator, resolver adapters, event delivery, and source authorization hooks using only existing owners.
4. Add `host/jsonl`, transcript replay, subprocess conformance, and dedicated shell/PowerShell gates.
5. Complete the required example documentation baseline, then extend minimal and production-ish example behavior and expected markers.
6. Update module boundaries, runtime/config diagnostics documentation where ownership needs clarification, the contract-test index, Roadmap active status, README entry points, CONTRIBUTING/PR evidence only where their governed lists change.
7. Roll back by removing or disabling host coordinator construction and active-control registration. Existing direct Run/Stream, entry-time Realtime events, resolver callbacks, durable subscriptions, terminal recovery, and diagnostics remain valid because all new surfaces are additive and opt-in.

## Open Questions

None. The first profile, ownership boundaries, frame policy, disconnect behavior, HITL outcomes, and deferred steering/follow-up scope are fixed by this design; implementation discoveries that would change them require an OpenSpec update before code changes.

## Implementation review and residual risks

Verification completed on 2026-09-12:

- `go test ./... -count=1 -timeout 20m` passed.
- `go test -race ./... -count=1 -timeout 30m` passed with the installed MinGW compiler made visible through an isolated CGO wrapper; the wrapper is not part of the product tree.
- `golangci-lint run --config .golangci.yml`, OpenSpec validation, docs consistency, examples smoke/stability, and the A64 harnessability scorecard passed.
- The runner helper extraction reduced `core/runner/runner.go` to `5840` lines against the `5871` baseline, so the unchanged Go file-line-budget policy now passes.
- The full PowerShell quality gate passed repository hygiene, docs consistency, host-specific contracts, ordinary tests, CGO-enabled race tests, lint, A64 semantic stability, and A64 performance regression in 902.27s (62 steps) with `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=1800`. The diagnostics-query benchmark used its existing sampling controls (`BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME=500ms`, `BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT=9`) to reduce transient host scheduling noise; no baseline or threshold was changed.
- The earlier default-sampling attempt showed high variance in the two RuntimeRecorder `ns/op` medians, while the stable-sampling rerun passed both RuntimeRecorder benchmarks and the remaining diagnostics-query and multi-agent performance checks. This is retained as execution evidence rather than treated as a product regression or a reason to recalibrate the governed baseline.

Residual risks are intentionally kept visible:

1. Task 1.3 now has a reproducible pre-implementation RED matrix in `red-evidence.md`: the actual pre-change `HEAD` is isolated, only the contract tests/fixture are injected, and all six focused suites fail with missing implementation symbols before any host implementation source is present. The matrix explicitly maps the listed scenarios to the focused test areas and transcript cases.
2. Direct Runner callbacks remain non-droppable because `EventHandler` carries no source delivery policy. Recorded dropping is limited to source-declared durable Realtime `delta` projections with `delivery_policy=drop_with_record`, `source_outcome_declared=true`, and a standard EventSink; without that sink the host fails closed and treats the projection as critical.
3. The first profile excludes steering/follow-up input semantics and remote gateway/persistence concerns by design.

Archive readiness: **ready for archive review** after the final user/reviewer confirmation. All implementation, test, documentation, gate, and pre-implementation RED evidence tasks are complete. No archive or destructive cleanup is authorized automatically by this review.
