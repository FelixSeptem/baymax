## ADDED Requirements

### Requirement: Primary-reason arbitration SHALL align with policy-stack winner semantics
Cross-domain primary-reason arbitration MUST preserve alignment between arbitration output and policy precedence winner fields.

When policy-stack winner exists, arbitration explainability output MUST expose consistent stage/source semantics and MUST NOT remap winner source to a conflicting taxonomy.

#### Scenario: Arbitration receives policy winner from higher-precedence stage
- **WHEN** policy evaluator marks `security_s2` as winner and readiness also reports blocked
- **THEN** primary-reason arbitration output preserves winner-stage/source alignment without conflicting remap

#### Scenario: Arbitration output is replayed with equivalent winner input
- **WHEN** equivalent arbitration events with identical policy winner are replayed
- **THEN** primary reason and policy winner alignment remains deterministic and idempotent
