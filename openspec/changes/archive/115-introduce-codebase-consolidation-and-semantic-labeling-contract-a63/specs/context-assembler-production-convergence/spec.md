## ADDED Requirements

### Requirement: Context Assembler Production Convergence SHALL Use Semantic Stage Labels in Active Artifacts
Active production-convergence implementation and verification artifacts MUST describe Context Assembler stages with semantic labels, not CA-number stage labels.

Historical CA numbering MAY be referenced only through canonical mapping index for traceability.

#### Scenario: Convergence tests are updated
- **WHEN** contributor adds or modifies tests for context assembler convergence behavior
- **THEN** test names and descriptions MUST use semantic stage labels and avoid direct `ca1|ca2|ca3|ca4` wording

#### Scenario: Historical CA naming is needed during incident analysis
- **WHEN** maintainer needs to correlate historical CA-number references
- **THEN** correlation MUST be resolved via canonical mapping index and not by restoring CA-number vocabulary in active artifacts

### Requirement: Context Convergence Migration SHALL Preserve Existing Runtime Semantics
Context naming convergence MUST preserve deterministic threshold, fallback, and Run/Stream equivalence behavior defined by production convergence contracts.

#### Scenario: Stage labels are renamed to semantic vocabulary
- **WHEN** naming migration updates context assembler labels
- **THEN** existing threshold strategy, token-count fallback order, and Run/Stream equivalence semantics MUST remain unchanged

#### Scenario: Legacy alias path is exercised during migration
- **WHEN** compatibility alias is consumed by existing tests or scripts
- **THEN** canonical behavior outcomes MUST remain equivalent to semantic-label path
