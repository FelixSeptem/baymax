## ADDED Requirements

### Requirement: Realtime Interrupt/Resume SHALL Preserve Run and Stream Semantic Equivalence
For equivalent input, effective configuration, and dependency state, Run and Stream MUST preserve semantically equivalent interrupt/resume outcomes, including interrupt acceptance, resume restoration class, and terminal classification.

#### Scenario: Equivalent interrupt accepted in Run and Stream
- **WHEN** equivalent requests in Run and Stream hit identical interrupt trigger
- **THEN** both paths MUST expose semantically equivalent interrupt acceptance and freeze boundary semantics

#### Scenario: Equivalent resume success in Run and Stream
- **WHEN** equivalent requests in Run and Stream resume from semantically equivalent cursor
- **THEN** both paths MUST expose semantically equivalent resume outcome and terminal class

#### Scenario: Equivalent invalid-resume rejection in Run and Stream
- **WHEN** equivalent requests in Run and Stream resume from invalid cursor
- **THEN** both paths MUST expose semantically equivalent rejection classification

### Requirement: Realtime Interrupt/Resume MUST NOT Introduce Parallel Loop Semantics
Realtime interrupt/resume integration MUST remain within canonical loop boundaries and MUST NOT introduce parallel loop family or alternate termination taxonomy.

#### Scenario: Realtime enabled ReAct run remains under canonical taxonomy
- **WHEN** realtime protocol and interrupt/resume are enabled for ReAct path
- **THEN** runtime MUST continue using canonical A56 termination taxonomy without alternate loop family

