## ADDED Requirements

### Requirement: Readiness findings SHALL align with composite replay fixtures
Runtime readiness preflight findings MUST remain alignable with A47 composite replay fixtures through canonical fields and stable finding-code taxonomy.

Readiness fixture assertions MUST cover:
- strict/non-strict classification path,
- primary finding code stability,
- degraded-to-blocked escalation semantics.

#### Scenario: Strict escalation is validated through composite fixture
- **WHEN** composite fixture models degraded finding under strict readiness policy
- **THEN** replay assertion confirms blocked classification with canonical readiness code mapping

#### Scenario: Readiness taxonomy drifts from canonical mapping
- **WHEN** composite fixture detects non-canonical readiness finding code
- **THEN** replay fixture validation fails and blocks gate
