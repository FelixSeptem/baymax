## ADDED Requirements

### Requirement: Owner append failures SHALL be observable

When an execution owner is configured with a native event history writer, append failures MUST NOT be silently discarded while capability state continues to claim complete retained source history. The owner MUST expose a typed health/availability outcome and preserve whether source facts may be incomplete. This requirement does not make event history and external consumer projection one transaction.

#### Scenario: Journal write failure is surfaced
- **WHEN** an explicitly configured runner, child, team, or workflow journal cannot persist an appended source event
- **THEN** the owner records or returns a typed history unavailable/degraded outcome associated with the source
- **AND** it does not claim that the failed event is replayable

#### Scenario: No history writer is configured
- **WHEN** the owner runs without an explicitly injected native history writer
- **THEN** its history capability remains unsupported
- **AND** it does not create an implicit journal or derive history from callback identifiers

### Requirement: History replay SHALL NOT repair uncertain execution

Source history replay MUST return only facts whose source append committed. If a source transition may have occurred without a committed fact, the owner MUST report recovery-required/unavailable and MUST NOT use replay, parent Run, child Run, retry, or resume to synthesize the missing fact.

#### Scenario: Append failure leaves a source gap
- **WHEN** a source journal append fails and a later read observes unavailable or incomplete history
- **THEN** the reader reports a typed unavailable or gap outcome
- **AND** it does not silently return latest or invent a terminal event
