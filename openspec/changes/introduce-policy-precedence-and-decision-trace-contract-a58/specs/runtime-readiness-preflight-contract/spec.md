## ADDED Requirements

### Requirement: Readiness preflight SHALL expose policy-stack candidate mapping
Runtime readiness preflight MUST expose canonical policy candidate mapping metadata so admission can apply deterministic precedence without reclassification drift.

Candidate mapping MUST remain machine-assertable and deterministic under equivalent inputs.

#### Scenario: Preflight observes multi-domain findings
- **WHEN** preflight detects both sandbox and adapter allowlist findings
- **THEN** output includes canonical policy candidate metadata aligned to policy stages

#### Scenario: Repeated preflight under unchanged inputs
- **WHEN** host calls preflight repeatedly without config or dependency changes
- **THEN** candidate mapping remains semantically equivalent and stable

### Requirement: Preflight primary output SHALL align with policy precedence winner
When readiness contributes to policy candidates, preflight output MUST preserve deterministic alignment with precedence winner semantics.

#### Scenario: Readiness blocked but higher-priority stage denies
- **WHEN** preflight contains `readiness_admission` blocked finding while `action_gate` winner exists
- **THEN** preflight outputs readiness finding without overriding policy winner semantics

#### Scenario: Readiness is highest available blocking stage
- **WHEN** no higher-priority stage contributes blocking candidate
- **THEN** readiness stage may become winner and remains machine-assertable
