## MODIFIED Requirements

### Requirement: Embedded host SHALL separate dynamic action-resume admission from source outcome

The embedded host contract MUST support a versioned action-resume command carrying the source Run/session correlation, opaque action token, checkpoint version/digest, decision reference, and bounded idempotency key. The command response MUST report only admission or rejection. For every accepted confirm, deny, or timeout decision, the source Runtime MUST emit exactly one correlated `run.dynamic_action.resolved` event after recording the decision. The source Runtime MUST emit a canceled terminal projection only for deny or timeout; confirm MUST produce the subsequent source-owned resumed or terminal outcome without a synthetic canceled event. Duplicate equivalent action-resume submissions MUST be recognized before event emission and MUST NOT duplicate the resolution event.

#### Scenario: Action-resume command is accepted for confirmation
- **WHEN** a host submits a valid resume command for an active matching `input_required` checkpoint with a confirm decision
- **THEN** the host receives an accepted command response and later receives exactly one source-owned `run.dynamic_action.resolved` event followed by the resumed or terminal outcome with the same Run correlation

#### Scenario: Action-resume command is accepted
- **WHEN** a host submits a valid resume command for an active matching `input_required` checkpoint
- **THEN** the host receives an accepted command response and later receives the source-owned resumed or terminal outcome with the same Run correlation

#### Scenario: Action-resume command is accepted for denial
- **WHEN** a host submits a valid resume command with a deny decision
- **THEN** the host receives an accepted command response, one correlated resolution event, and one source-owned canceled terminal projection; the business executor is not invoked

#### Scenario: Action-resume command is accepted for timeout
- **WHEN** a valid timeout decision is admitted for an active checkpoint
- **THEN** the host receives one correlated resolution event and one canceled terminal projection without action execution

#### Scenario: Duplicate equivalent action-resume command is admitted idempotently
- **WHEN** the same action-resume identity and normalized decision are submitted more than once
- **THEN** the duplicate response returns the prior source result and no additional resolution or terminal event is emitted

#### Scenario: Invalid action-resume command is rejected before mutation
- **WHEN** token, checkpoint digest, session, Run, authorization, or idempotency validation fails
- **THEN** the command response contains a stable rejection classification and no Runner or business executor state is mutated

#### Scenario: Host disconnect does not synthesize resume or terminal state
- **WHEN** the host disconnects while a dynamic action-resume request or response is pending
- **THEN** the adapter settles transport correlation through its bounded disconnect behavior and leaves source-owned Run/action state unchanged

#### Scenario: Action-resume transcript is replayable
- **WHEN** replay consumes a valid action registration, host admission, source pause, decision, resolution, and resume transcript
- **THEN** replay reproduces the normalized correlations and Run/Stream terminal semantics without a live host, provider, tool, or executor
