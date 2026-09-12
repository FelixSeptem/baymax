## Purpose

定义嵌入式宿主与 Baymax source Runtime 之间可验证、可回放且不拥有业务状态的命令响应、异步事件、反向 HITL 请求和严格 JSONL/stdio 绑定合同。

## ADDED Requirements

### Requirement: Embedded host protocol SHALL use versioned transport-neutral envelopes
The embedded host protocol MUST define distinct versioned envelopes for `command`, `command_response`, `runtime_event`, `host_request`, and `host_response`. Every envelope MUST include a bounded unique message identifier, kind, protocol version, and timestamp; envelopes associated with execution MUST retain `run_id`, `session_id`, request correlation, causation, and source correlation when those facts are known.

Unsupported versions, unknown kinds, missing required correlation, invalid identifiers, and malformed payloads MUST fail before source Runtime mutation with deterministic protocol classifications.

#### Scenario: Command is correlated across response and events
- **WHEN** a host submits a valid command with a request identifier and known Run correlation
- **THEN** the command response echoes that request correlation and later Runtime events retain the same `run_id` and source-owned causation facts

#### Scenario: Malformed envelope fails before mutation
- **WHEN** an envelope omits its required identifier or uses an unsupported protocol version
- **THEN** the adapter returns a deterministic protocol rejection and does not invoke a source Runtime action

### Requirement: Command admission SHALL remain separate from asynchronous execution outcome
The protocol MUST expose a finite first-profile command vocabulary covering Run start, advertised lifecycle actions, Realtime interrupt/resume, HITL response, event subscription, and authoritative Run outcome query. A `command_response` MUST report only whether the command was accepted, rejected, or recognized as a duplicate; acceptance MUST NOT be represented as completion of the requested business operation.

Business progress and terminal outcomes MUST arrive as correlated `runtime_event` envelopes from the existing source-owned event and terminal recovery contracts. A protocol adapter MUST NOT synthesize a successful, failed, canceled, or input-required Run state merely from command admission, transport closure, or delivery failure.

#### Scenario: Accepted Run start completes asynchronously
- **WHEN** a valid Run start command passes admission
- **THEN** the host first receives an accepted command response and later receives source-owned progress and exactly one authoritative terminal projection

#### Scenario: Rejected command does not create a terminal outcome
- **WHEN** readiness, authorization, capability, state, or correlation validation rejects a command
- **THEN** the command response contains the stable rejection classification and no synthetic Run terminal event is emitted

### Requirement: Host commands SHALL preserve source ownership and authorization boundaries
The adapter MUST delegate accepted commands to the source-owned Runner, Run control, Realtime, durable stream, terminal recovery, readiness, policy, and sandbox interfaces. Declared action availability MUST remain distinct from authorization. Unknown or inactive Run targets, unsupported actions, expired cursors, and denied actions MUST be rejected deterministically without using diagnostics history as an active Run registry.

#### Scenario: Authorized cancel reaches the source Run control
- **WHEN** a host submits an authorized cancel command for an active Run that advertises cancel support
- **THEN** the adapter delegates to that Run's source-owned control and reports the source admission result without owning the terminal transition

#### Scenario: Available action remains subject to authorization
- **WHEN** an action is advertised but existing policy or sandbox governance denies the caller's command
- **THEN** the adapter rejects the command before source mutation and preserves the existing denial classification

### Requirement: Host-mediated HITL SHALL adapt existing resolver semantics
The adapter MUST be able to project existing clarification and Action Gate resolver calls as correlated `host_request` envelopes and accept exactly one matching `host_response`. The projection MUST preserve the existing RequestID, bounded request payload, timeout, Action Gate fail-closed behavior, clarification `resumed|canceled_by_user` behavior, timeline events, and Run/Stream equivalence.

Duplicate, late, unknown, or mismatched host responses MUST be rejected deterministically and MUST NOT resume execution twice. Transport disconnect MUST NOT directly assign a Run terminal state; unresolved HITL MUST settle through its existing timeout, cancellation, or fail-closed owner.

#### Scenario: Clarification response resumes exactly once
- **WHEN** a host returns one valid response for a pending clarification RequestID before timeout
- **THEN** the existing clarification resolver resumes the matching Run once and a duplicate response is rejected without additional mutation

#### Scenario: Action Gate transport loss remains fail closed
- **WHEN** the host connection closes while an Action Gate confirmation is pending
- **THEN** the pending bridge is closed deterministically and the existing Action Gate owner denies or times out the tool action without the adapter executing it

### Requirement: Pending request lifecycle SHALL close deterministically
Each host connection MUST maintain only bounded transport-correlation state for its commands and reverse requests. On timeout, output failure, protocol shutdown, or disconnect, all affected pending entries MUST reach a deterministic completed, rejected, timed-out, or disconnected result and MUST be removed exactly once.

Closing a host connection MUST stop that connection's observation and reverse-request delivery but MUST NOT implicitly cancel an unrelated business Run. Reconnection MUST use the existing durable stream cursor and terminal recovery contracts rather than replaying adapter-owned state.

#### Scenario: Disconnect rejects pending transport requests
- **WHEN** a host disconnects with command responses or reverse requests still pending
- **THEN** the adapter closes each pending entry once with a disconnected classification and retains no connection-owned pending entry afterward

#### Scenario: Business Run survives observer disconnect
- **WHEN** a host disconnects after a Run command was accepted and no explicit cancel policy applies
- **THEN** source execution remains authoritative and a later subscriber recovers events and terminal outcome through the existing cursor and recovery contracts

### Requirement: JSONL stdio binding SHALL preserve framing and output purity
The first binding MUST encode exactly one JSON envelope per LF-terminated frame over stdin/stdout. Only byte `0x0A` terminates a frame; valid JSON string content containing Unicode `U+2028` or `U+2029` MUST NOT split a frame. The binding MUST validate a configurable bounded frame size, reject oversized or malformed frames fail-fast, and perform explicit protocol-version negotiation before accepting mutation commands.

Stdout MUST contain protocol frames only. Logs and non-protocol diagnostics MUST use an injected non-stdout writer, with stderr as the command default. Partial writes, interleaved frames, and write failures MUST be detected and classified.

#### Scenario: Unicode line separators remain inside one frame
- **WHEN** a valid JSON string contains `U+2028` or `U+2029` but no LF byte
- **THEN** the binding parses it as one frame and preserves the string content

#### Scenario: Oversized frame is rejected without source mutation
- **WHEN** an input frame exceeds the effective maximum before its LF delimiter
- **THEN** the binding returns a deterministic frame-too-large failure, discards no subsequent bytes as a valid mutation command, and invokes no source action for that frame

#### Scenario: Accidental stdout log is detected
- **WHEN** non-protocol output is written to the protocol stdout channel
- **THEN** subprocess conformance fails output-purity validation rather than treating the line as an asynchronous Runtime event

### Requirement: Delivery backpressure SHALL be explicit and bounded
Command responses, host requests, host responses, protocol negotiation, and authoritative terminal events MUST NOT be silently dropped. Runtime event delivery MUST declare and reuse a compatible source-owned delivery policy; an adapter MUST NOT obtain backpressure by implicitly blocking the Runner's callback indefinitely or by creating an unbounded or global event queue.

When a critical frame cannot be delivered, the binding MUST close or reject the affected transport operation deterministically, settle pending correlation, and leave the source business terminal owner unchanged. Low-priority event dropping is allowed only when the existing source policy permits it and records the drop through the existing observability path. In the first profile, an event is eligible only when a source-returned durable Realtime projection declares both `delivery_policy=drop_with_record` and `source_outcome_declared=true`, the source event type is `delta`, and a standard EventSink is available to record the outcome. A client-requested policy alone MUST NOT authorize dropping. Direct Runner callback events and Realtime `request|interrupt|resume|ack|error|complete` events remain non-droppable.

#### Scenario: Slow writer cannot silently lose a terminal event
- **WHEN** output remains backpressured beyond the binding's bounded delivery deadline while a terminal event is pending
- **THEN** the binding reports a delivery failure and closes the affected observation path without synthesizing a different Run terminal outcome

#### Scenario: Low-priority drop follows source policy
- **WHEN** a source explicitly allows recorded low-priority drops and the connection queue reaches its bound
- **THEN** only eligible events are dropped, the drop is recorded through the standard event path, and control and terminal frames remain non-droppable

#### Scenario: Requested drop policy lacks source declaration or event sink
- **WHEN** a client requests `drop_with_record` but the source does not declare the outcome or no standard EventSink is available
- **THEN** the adapter treats every projected frame as non-droppable and applies the bounded critical-delivery failure behavior

### Requirement: Host contract SHALL be replayable and Run/Stream equivalent
The Runtime MUST provide versioned fixtures and subprocess conformance for valid correlation, admission/event separation, duplicate and late commands, HITL response races, disconnect cleanup, cursor recovery, terminal conflict, LF framing, frame bounds, stdout purity, write backpressure, and unsupported versions. Equivalent Run and Stream requests MUST normalize to equivalent command admission, lifecycle, HITL, failure, and terminal outcomes after permitted event-order normalization.

The contract gate MUST preserve shell/PowerShell parity and MUST fail if the implementation introduces a hosted listener, remote Session/Artifact store, adapter-owned terminal state machine, diagnostics writer, global event queue, or implicit steering/follow-up semantics.

#### Scenario: Canonical host transcript replays successfully
- **WHEN** replay consumes a valid command/response/event/HITL transcript and its expected source-owned outcomes
- **THEN** normalization succeeds deterministically without network access or a live model

#### Scenario: Run and Stream host outcomes diverge
- **WHEN** equivalent Run and Stream host transcripts produce different admission, HITL, failure, or terminal semantics outside permitted event ordering
- **THEN** replay classifies parity drift and the contract gate fails
