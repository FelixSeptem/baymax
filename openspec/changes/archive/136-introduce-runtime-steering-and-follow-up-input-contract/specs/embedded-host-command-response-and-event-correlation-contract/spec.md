## MODIFIED Requirements

### Requirement: Command admission SHALL remain separate from asynchronous execution outcome
The protocol MUST expose a finite first-profile command vocabulary covering Run start, advertised lifecycle actions, Realtime interrupt/resume, HITL response, steering/follow-up input submission, event subscription, and authoritative Run outcome query. A `command_response` MUST report only whether the command was accepted, rejected, or recognized as a duplicate; acceptance MUST NOT be represented as completion of the requested business operation. Input command acceptance MUST identify the source-owned input kind and admission correlation without claiming that steering was applied or that follow-up execution has started.

Business progress, input-application facts, and terminal outcomes MUST arrive as correlated `runtime_event` envelopes from the existing source-owned event and terminal recovery contracts. A protocol adapter MUST NOT synthesize a successful, failed, canceled, input-required, steering-applied, or follow-up-started state merely from command admission, transport closure, or delivery failure.

#### Scenario: Accepted steering command completes at a later boundary
- **WHEN** a valid steering command passes admission while a Run is active
- **THEN** the host first receives an accepted command response and later receives a source-owned input-application or not-applied fact at the eligible decision boundary

#### Scenario: Accepted follow-up command does not reopen a terminal Run
- **WHEN** a valid follow-up command is accepted after the current Run reaches its idle or terminal boundary
- **THEN** the host receives a correlated source-owned follow-up admission/execution projection and the original terminal Run is not rewritten

#### Scenario: Rejected input command does not create a synthetic outcome
- **WHEN** input validation, authorization, backpressure, stale state, or terminal arbitration rejects a steering or follow-up command
- **THEN** the command response contains the stable rejection classification and no synthetic input-applied or Run terminal event is emitted

#### Scenario: Accepted Run start completes asynchronously
- **WHEN** a valid Run start command passes admission
- **THEN** the host first receives an accepted command response and later receives source-owned progress and exactly one authoritative terminal projection

#### Scenario: Rejected command does not create a terminal outcome
- **WHEN** readiness, authorization, capability, state, or correlation validation rejects a command
- **THEN** the command response contains the stable rejection classification and no synthetic Run terminal event is emitted

### Requirement: Host commands SHALL preserve source ownership and authorization boundaries
The adapter MUST delegate accepted lifecycle, Realtime, and steering/follow-up commands to source-owned Runner, Run control, input admission, durable stream, terminal recovery, readiness, policy, and sandbox interfaces. Declared action or input capability availability MUST remain distinct from authorization. Unknown or inactive Run targets, unsupported input kinds, expired cursors, stale input identities, full source-owned input boundaries, and denied actions MUST be rejected deterministically without using diagnostics history or host state as an active Run or input registry.

#### Scenario: Authorized steering reaches source input admission
- **WHEN** a host submits an authorized steering command for an active Run that advertises steering support
- **THEN** the adapter delegates to the source-owned input admission and reports its outcome without applying the input or owning the terminal transition

#### Scenario: Host does not own a global follow-up queue
- **WHEN** a follow-up command is accepted for a Run that is not yet idle
- **THEN** the source-owned bounded input boundary retains the declared pending outcome and the host adapter does not create a global queue or mutate Session history

#### Scenario: Available input capability remains subject to authorization
- **WHEN** a descriptor advertises steering or follow-up but existing policy, readiness, or sandbox governance denies the caller
- **THEN** the adapter rejects the command before source mutation and preserves the existing denial classification

#### Scenario: Authorized cancel reaches the source Run control
- **WHEN** a host submits an authorized cancel command for an active Run that advertises cancel support
- **THEN** the adapter delegates to that Run's source-owned control and reports the source admission result without owning the terminal transition

#### Scenario: Available action remains subject to authorization
- **WHEN** an action is advertised but existing policy or sandbox governance denies the caller's command
- **THEN** the adapter rejects the command before source mutation and preserves the existing denial classification

### Requirement: Host contract SHALL be replayable and Run/Stream equivalent
The Runtime MUST provide versioned fixtures and subprocess conformance for valid correlation, admission/event separation, duplicate and late commands, steering/follow-up admission and apply-boundary races, HITL response races, disconnect cleanup, cursor recovery, terminal conflict, LF framing, frame bounds, stdout purity, write backpressure, and unsupported versions. Equivalent Run and Stream requests MUST normalize to equivalent command admission, input kind/order/apply outcomes, lifecycle, HITL, failure, and terminal outcomes after permitted event-order normalization.

The contract gate MUST preserve shell/PowerShell parity and MUST fail if the implementation introduces a hosted listener, remote Session/Artifact store, adapter-owned terminal state machine, diagnostics writer, global input/event queue, direct Provider prompt injection, or host-owned steering/follow-up semantics.

#### Scenario: Canonical input transcript replays successfully
- **WHEN** replay consumes a valid steering/follow-up command, source admission, apply-boundary, and terminal transcript
- **THEN** normalization succeeds deterministically without network access, a live model, or a live tool

#### Scenario: Run and Stream input outcomes diverge
- **WHEN** equivalent Run and Stream host transcripts produce different input admission, apply boundary, failure, or terminal semantics outside permitted event ordering
- **THEN** replay classifies parity drift and the contract gate fails

#### Scenario: Canonical host transcript replays successfully
- **WHEN** replay consumes a valid command/response/event/HITL transcript and its expected source-owned outcomes
- **THEN** normalization succeeds deterministically without network access or a live model

#### Scenario: Run and Stream host outcomes diverge
- **WHEN** equivalent Run and Stream host transcripts produce different admission, HITL, failure, or terminal semantics outside permitted event ordering
- **THEN** replay classifies parity drift and the contract gate fails

#### Scenario: Host-owned input semantics are introduced
- **WHEN** implementation introduces an unbounded host/global input queue or applies steering by injecting into an in-flight Provider call
- **THEN** the contract gate fails the source-ownership and safe-point assertions
