## ADDED Requirements

### Requirement: Embedded host SHALL separate dynamic action-resume admission from source outcome

The embedded host contract MUST support a versioned action-resume command carrying the source Run/session correlation, opaque action token, checkpoint version/digest, decision reference, and bounded idempotency key. The command response MUST report only admission or rejection; source-owned `input_required`, resumed, rejected, duplicate, stale, and terminal outcomes MUST arrive as correlated runtime events and terminal projections.

#### Scenario: Action-resume command is accepted
- **WHEN** a host submits a valid resume command for an active matching `input_required` checkpoint
- **THEN** the host receives an accepted command response and later receives the source-owned resumed or terminal outcome with the same Run correlation

#### Scenario: Invalid action-resume command is rejected before mutation
- **WHEN** token, checkpoint digest, session, Run, authorization, or idempotency validation fails
- **THEN** the command response contains a stable rejection classification and no Runner or business executor state is mutated

#### Scenario: Host disconnect does not synthesize resume or terminal state
- **WHEN** the host disconnects while a dynamic action-resume request or response is pending
- **THEN** the adapter settles transport correlation through its bounded disconnect behavior and leaves source-owned Run/action state unchanged

#### Scenario: Action-resume transcript is replayable
- **WHEN** replay consumes a valid action registration, host admission, source pause, decision, and resume transcript
- **THEN** replay reproduces the normalized correlations and Run/Stream terminal semantics without a live host, provider, tool, or executor
