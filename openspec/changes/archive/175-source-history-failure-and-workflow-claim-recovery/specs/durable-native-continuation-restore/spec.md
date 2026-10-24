## ADDED Requirements

### Requirement: Reopened workflow stores SHALL classify incomplete running claims

A workflow continuation store reopened after process loss MUST classify every resume claim without a committed result, including claims persisted as `claimed` or `running`, as `recovery_required`. Exact duplicate requests MUST return that stable typed recovery outcome. The store MUST NOT restart uncertain step dispatch automatically.

#### Scenario: Process exits after running claim
- **WHEN** a new workflow store opens a persisted `running` resume claim without a committed result
- **THEN** it changes the observable claim state to `recovery_required`
- **AND** duplicate resume returns a typed recovery-required error without dispatching any step

#### Scenario: Process exits after terminal result commit
- **WHEN** a new workflow store opens a claim with a committed terminal result
- **THEN** an exact duplicate resume returns the retained result projection
- **AND** no completed step or effect is dispatched again

### Requirement: Workflow FileStore SHALL enforce its advertised single-writer scope

A workflow FileStore that advertises same-host single-writer sequential reopen MUST acquire a process-wide owner fence before reading or mutating continuation records. A concurrent or stale writer MUST fail with a typed ownership/unavailable outcome before resume admission. Releasing the store MUST release its fence so a successor process can reopen it.

#### Scenario: Concurrent workflow store open is rejected
- **WHEN** a second process opens the same workflow continuation store while an owner holds its fence
- **THEN** the second open fails with a typed ownership/unavailable outcome
- **AND** it does not read a stale snapshot or claim a resume

#### Scenario: Successor reopens after owner release
- **WHEN** the current store owner closes cleanly and a successor opens the same path
- **THEN** the successor reads committed continuation records and applies incomplete-claim recovery classification
- **AND** no distributed or multi-writer capability is implied
