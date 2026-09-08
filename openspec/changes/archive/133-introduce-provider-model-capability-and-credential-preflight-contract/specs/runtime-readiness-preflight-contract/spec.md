## ADDED Requirements

### Requirement: Readiness projects provider-model admission findings
The runtime readiness preflight SHALL consume the provider-model catalog admission projection for the selected provider and model. It MUST expose canonical findings for unknown models, required capability gaps, credential evidence, declared fallback selection, and catalog reload rollback while retaining normalized provider/model identity and catalog version.

#### Scenario: Unknown model is reported as a blocking readiness finding
- **WHEN** readiness evaluates a provider/model identity absent from the active catalog
- **THEN** it returns `blocked` with the canonical unknown-model finding and does not dispatch provider work

#### Scenario: Optional fallback is reported as degraded readiness
- **WHEN** readiness evaluates an optional capability gap resolved by an independently admitted declared fallback
- **THEN** it returns `degraded`, preserves the original optional-gap finding, and exposes the selected fallback identity

### Requirement: Existing readiness aggregation remains authoritative
Provider-model findings SHALL use the existing `ready|degraded|blocked` statuses, strict-mode escalation, primary-reason arbitration, and canonical finding schema. The change MUST NOT introduce a parallel readiness or terminal-state machine, duplicate primary-reason field, or provider-specific status vocabulary.

#### Scenario: Strict mode escalates unverified credential evidence
- **WHEN** an otherwise admissible model has `unverified` credential evidence and strict readiness policy is enabled
- **THEN** readiness applies the existing strict-mode escalation and selects the canonical primary credential reason

### Requirement: Run and Stream readiness semantics remain equivalent
For equivalent catalog, credential evidence, request capability, and runtime-policy inputs, Run and Stream SHALL invoke the same readiness projection and produce semantically equivalent status, selected identity, fallback identity, and ordered canonical findings.

#### Scenario: Run and Stream return equivalent preflight outcomes
- **WHEN** Run and Stream are evaluated with equivalent provider-model admission inputs
- **THEN** their readiness outcomes are equivalent and neither path introduces a separate decision or termination meaning
