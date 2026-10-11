# native-execution-event-adapter Specification

## Purpose
Provide an embeddable, transport-neutral contract for consuming real source execution events with bounded host correlation and explicit observation capabilities. Preserve existing lifecycle, ordering, identity and diagnostic owners while making projection and delivery failures visible to the embedding application.

## Requirements

### Requirement: Native execution projection SHALL consume source events without synthesizing execution outcomes

An opt-in adapter SHALL project actual Run/Stream callbacks and explicitly bound source timeline events using the existing canonical protocol mapping. It MUST preserve source identity, native event type and schema version, iteration, time, Run/Step correlation, available trace/span and source parent/causation references, and compatible payload facts. Missing optional source correlation MUST remain absent unless supplied by a validated source binding. Invalid source or conflicting Run/Session binding MUST be rejected before sink delivery without mutating source state. Final results MUST NOT be used to manufacture additional parent, progress, resolution or terminal events.

#### Scenario: Real model and tool events reach the host
- **WHEN** a real runner with fake model and tool invokes Run or Stream with the opt-in handler
- **THEN** the sink receives mapped native model/tool/lifecycle events with original source correlation and receives no final-result-derived synthetic events

#### Scenario: Child source facts remain distinct
- **WHEN** parent and child sources emit interleaved events with existing parent-child references
- **THEN** each event retains its source Run and supplied causal references without the adapter imposing a global sequence or replacing the child Run with the parent Run

#### Scenario: Source binding mismatch is rejected
- **WHEN** an event contradicts the expected source Run or Session binding
- **THEN** projection reports a deterministic correlation failure, delivers no conflicting event and leaves source execution unchanged

### Requirement: Observation capabilities SHALL distinguish callback references from durable source identity

The adapter SHALL expose a versioned capability profile distinguishing callback ordering, identity provenance, deduplication, gap verification, durable history and cursor availability. Direct callbacks MUST declare mapping-reference identity, preserve their existing mapped EventID, expose no source sequence or durable cursor, and MUST NOT deduplicate solely by EventID, TraceID, timestamp or payload hash. Every valid direct callback MUST be delivered while the observer is healthy. Callback order guarantees MUST be limited to a serial source stream; concurrent independent sources MUST NOT claim a global order. Unsupported required durable capabilities MUST fail explicitly before observation starts, without silently starting at latest.

#### Scenario: Shared trace identity does not discard distinct events
- **WHEN** two distinct native callbacks share a TraceID and therefore the existing mapper returns the same EventID
- **THEN** both callbacks are delivered with their native types preserved and the profile makes clear that the ID is a mapping reference rather than a deduplication key

#### Scenario: Repeated direct callback is not inferred to be a duplicate
- **WHEN** the same valid direct callback is received twice by a healthy observer
- **THEN** both callbacks are delivered without adapter-owned logical deduplication

#### Scenario: Runner cursor request fails explicitly
- **WHEN** a consumer requires after-cursor observation from a direct runner callback profile
- **THEN** admission reports unsupported cursor/history capability before execution and does not invent a cursor or silently downgrade

### Requirement: Realtime projection SHALL reuse source-owned binding semantics

A separate projection entry point SHALL accept only an existing valid source-normalized Realtime binding and preserve its identity, sequence, subscription correlation, cursor outcome, retention and delivery policy facts. It MUST reuse existing overlap, gap and compatibility validation. It MUST NOT broaden the binding to other sources, retain a history store, perform subscription control, add cross-batch deduplication state or treat cursor replay as execution resume.

#### Scenario: Normalized overlap retains source identity
- **WHEN** a valid source-normalized Realtime catch-up/live projection is adapted
- **THEN** its source-owned EventID, sequence, subscription boundary and normalized overlap are retained without a new adapter sequence or deduplication cache

#### Scenario: Gap or expired cursor is not disguised as live
- **WHEN** the existing binding classifies a sequence gap or expired cursor
- **THEN** projection preserves or reports that source classification rather than silently substituting a complete live stream

### Requirement: Host correlation SHALL be bounded and isolated from source facts

Host additions SHALL be restricted to reference-only task, schedule occurrence, plan version/digest, projection digest, knowledge generation and citation references. They MUST occupy a separate namespace and MUST NOT override source identity, parent/causation, lifecycle, decision, authorization or terminal fields. The default profile SHALL limit each scalar reference to 1024 UTF-8 bytes, citation lists to 16 items, combined host-reference JSON to 8 KiB and projected event JSON to 64 KiB. Explicit overrides MUST be finite positive constructor values; this profile introduces no runtime configuration keys.

Unknown reference keys, invalid types or UTF-8, serialization failure, conflicting source references and exceeded limits MUST fail deterministically without source mutation or silent field truncation. Successfully delivered events MUST own a detached snapshot of nested payload and reference values. Sensitive event bodies MUST NOT be copied to new diagnostics or replay fixtures.

#### Scenario: Host references enrich an event without replacing its owner
- **WHEN** a host supplies valid task, plan and citation references for a mapped tool event
- **THEN** the output contains those references separately and preserves the source Run/Step, lifecycle and action facts

#### Scenario: Exact bound succeeds and over-limit input fails
- **WHEN** a valid reference or event is exactly at its effective byte/item bound and another exceeds it
- **THEN** the first is accepted and the second reports a stable limit failure without partial sink delivery or source mutation

#### Scenario: Mutation after callback cannot change the delivered snapshot
- **WHEN** a source changes a nested payload or the host changes its original citation list after the callback returns
- **THEN** the previously delivered projection retains the original values

#### Scenario: Snapshot cannot invoke unbounded custom serialization
- **WHEN** payload values implement custom JSON or text serialization, other than standard time.Time
- **THEN** projection rejects them before invoking that serialization and reports a bounded serialization classification; empty Realtime binding metadata remains subject to UTF-8 and byte limits

### Requirement: Sink delivery SHALL expose bounded observation failure independently of execution

The adapter SHALL use a context-aware error-returning sink, a finite delivery deadline and fixed-size observation status. The default deadline SHALL be one second, with finite positive constructor overrides. Sink implementations MUST cooperate with context cancellation and MUST NOT reenter the same observer. The adapter MUST NOT create an unbounded queue, per-event detached worker, automatic delivery retry or indefinite serialization wait. Concurrent or reentrant input to one serial observer MUST fail deterministically instead of being silently reordered.

Because the native callback has no error return, projection failure, sink error, timeout, concurrency rejection and delivery after observer stop MUST be visible through bounded status containing the first failure reason/reference and delivery counts. A sink error or timeout SHALL mean unconfirmed delivery, not proof that a business write did not occur. After the first failure the observer MUST stop further sink delivery and retain its failure status. Observer context, close, disconnect and failure MUST NOT implicitly cancel, resume, retry or assign terminal state to the source Run. Diagnostics, when emitted through existing standard facts, MUST still use RuntimeRecorder only.

#### Scenario: Slow cooperative sink reaches its deadline
- **WHEN** a context-aware sink does not acknowledge an event before the effective deadline
- **THEN** the observer reports delivery timeout, stops further sink delivery, retains no unbounded worker or queue and leaves the source Run outcome under its owner

#### Scenario: Failure after application write remains unconfirmed
- **WHEN** a sink performs an application write and then returns an error
- **THEN** status records unconfirmed delivery and the adapter neither retries the event nor claims the write was rolled back

#### Scenario: Canceled Run can still report its source terminal
- **WHEN** a source Run context is canceled while the independent observer context remains active
- **THEN** the source canceled event can be delivered under the observer deadline without inventing or replacing its terminal outcome

#### Scenario: Observer close is idempotent and does not cancel execution
- **WHEN** the consumer closes an observer twice and the source subsequently emits another event
- **THEN** no further sink delivery occurs, stopped observation is visible and the adapter invokes no source lifecycle control

### Requirement: Lifecycle projection SHALL preserve pause resolution and authoritative terminal distinctions

The adapter MUST retain source lifecycle and terminal facts without owning a terminal state machine. An event named run.finished with state input_required MUST remain a non-terminal pause. Dynamic action resolution MUST remain a decision-received fact; confirm MUST NOT imply canceled, and deny/timeout MUST preserve only the source-owned canceled outcome. The adapter MUST NOT emit extra resolution or terminal events for duplicate equivalent resume. Missing or undelivered terminal evidence MUST be reported as incomplete observation rather than synthesized completion. Application submission remains subject to the host business commit gate.

#### Scenario: Dynamic action pause is not completion
- **WHEN** the source emits run.finished with input_required after producing a PendingAction
- **THEN** the projection retains input_required and the consumer contract does not classify the Run as completed or canceled

#### Scenario: Confirmation retains one resolution and source completion
- **WHEN** equivalent Run and Stream executions accept a confirm decision and later complete
- **THEN** each healthy observation contains exactly the source resolution followed by source completion, with no synthetic canceled event

#### Scenario: Denial timeout and duplicate resume retain source cardinality
- **WHEN** the source accepts deny or timeout, or receives an equivalent duplicate confirmation
- **THEN** the projection preserves the one resolution and canceled terminal for deny/timeout and emits no additional resolution/terminal for the duplicate

#### Scenario: Source completion and observation failure remain separate
- **WHEN** execution completes but its terminal event cannot be delivered
- **THEN** observer status reports incomplete delivery and does not change completed to failed or assert successful terminal receipt

### Requirement: Adapter conformance SHALL be offline replayable and Run Stream equivalent

The change SHALL provide versioned native_execution_event.v1 fixtures, a real-runner fake-model/tool integration suite, a host-neutral example and shell/PowerShell gates with equivalent semantics. Equivalent Run/Stream inputs SHALL preserve source lifecycle, correlation, resolution cardinality, failure classes, capability boundaries and observation outcomes after normalization of explicitly allowed generated identifiers, timestamps and model-delta fragmentation. Normalization MUST NOT hide ownership, terminal, identity-loss or delivery-failure drift.

Offline replay MUST invoke no model, tool, network, application sink or business write. Gates MUST reject introduction of a hosted event service, global ordering/dedup queue, adapter terminal state machine, direct diagnostics writer or new source execution owner.

#### Scenario: Permitted streaming differences do not hide lifecycle drift
- **WHEN** equivalent Run and Stream transcripts differ only in model-delta fragmentation and generated IDs/times
- **THEN** replay accepts them as equivalent, while a changed source terminal or resolution count produces parity drift

#### Scenario: Offline fixtures expose integration boundary failures
- **WHEN** replay consumes an unsupported cursor, correlation conflict, reference overflow, identity-loss, pause-as-terminal or observation-failure fixture
- **THEN** it returns the expected deterministic classification without runtime or application side effects

#### Scenario: Delta normalization retains semantic and business drift
- **WHEN** transcripts differ in an error, clarification, metadata or unknown model delta, delta source correlation, or nested business value that resembles an identity/time field
- **THEN** parity rejects that difference rather than removing it as fragmentation; only schema-owned generated fields and known fragment-only callbacks attached to an observed model step can be normalized
