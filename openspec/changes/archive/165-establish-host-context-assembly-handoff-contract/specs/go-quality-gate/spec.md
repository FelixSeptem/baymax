## ADDED Requirements

### Requirement: Quality gate SHALL enforce host context assembly handoff conformance

The standard quality gate SHALL invoke equivalent shell and PowerShell host-context-assembly-handoff contract gates when this contract's code, fixtures, replay, documentation, or example scope changes. The gates MUST execute the focused unit/integration/replay/privacy/Run-Stream suites, verify deliberate drift rejection, and fail if the change introduces source-body persistence, a Provider SDK dependency in `context/*`, a second budget ledger, a new execution/terminal/recovery state machine, direct diagnostic writing, or source side effects before descriptor rejection.

The gate MUST also verify the required public semantic documentation commitment, example impact declaration, active-roadmap status, and existence of the documented offline example baseline. A skipped or unavailable required check MUST report its reason and MUST NOT be represented as a passing conformance result.

#### Scenario: Contract drift blocks both platform gates
- **WHEN** a fixture or deliberate test mutation introduces projection digest drift, capability expansion, unauthorized restore, privacy leakage, Run/Stream divergence, or side effect before rejection
- **THEN** both platform gate implementations fail with the corresponding stable contract classification

#### Scenario: Ownership violation blocks quality admission
- **WHEN** the change introduces an unapproved context source store, budget write-back, provider SDK dependency, direct RuntimeRecorder write, or parallel continuation/terminal owner
- **THEN** the quality gate fails before the change can be accepted as conformant
