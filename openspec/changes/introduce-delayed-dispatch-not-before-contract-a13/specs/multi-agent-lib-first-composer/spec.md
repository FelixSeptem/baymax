## ADDED Requirements

### Requirement: Composer SHALL expose delayed child dispatch contract
Composer child dispatch contract MUST allow passing delayed dispatch intent (`not_before`) to scheduler-managed child tasks.

#### Scenario: Host dispatches child task with delayed execution
- **WHEN** host submits composer child dispatch request with future `not_before`
- **THEN** composer enqueues child task with delayed semantics and no premature claim occurs

### Requirement: Composer delayed child execution SHALL preserve Run/Stream semantic equivalence
For equivalent delayed child requests, composer-managed Run and Stream paths MUST preserve semantic equivalence of terminal category and additive counters.

#### Scenario: Equivalent delayed child workflow via Run and Stream
- **WHEN** equivalent delayed child dispatch is exercised through Run and Stream
- **THEN** terminal category and delayed-related additive summaries remain semantically equivalent
