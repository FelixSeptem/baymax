## ADDED Requirements

### Requirement: Core Module READMEs SHALL Describe Current State Only
Covered core-module READMEs MUST describe current implementation status and supported pathways, and MUST remove temporary, superseded, or stale milestone narrative from active guidance sections.

When historical context is needed, README MUST link to designated index/archive documentation instead of embedding outdated intermediate-state text.

#### Scenario: Contributor reads module README for onboarding
- **WHEN** contributor opens a covered core-module README
- **THEN** the document reflects current state and does not require filtering obsolete temporary notes

#### Scenario: Historical transition context is required
- **WHEN** module behavior has historical staged evolution
- **THEN** README references canonical archive/index path rather than duplicating temporary timeline narrative

### Requirement: Documentation Paths SHALL Be Canonical and Discoverable
Repository documentation MUST define canonical paths for architecture constraints, roadmap status, and contract index references.

Core READMEs and root README MUST use these canonical paths consistently.

#### Scenario: Contributor follows architecture boundary guidance
- **WHEN** contributor navigates from root or module README to architecture constraints
- **THEN** links resolve to canonical current-state documents without duplicate path variants

#### Scenario: Documentation path drift is introduced
- **WHEN** README references non-canonical or obsolete documentation paths for core governance topics
- **THEN** docs consistency validation MUST fail and require path convergence
