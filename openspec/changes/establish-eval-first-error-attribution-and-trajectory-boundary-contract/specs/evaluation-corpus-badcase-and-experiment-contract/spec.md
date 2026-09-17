## ADDED Requirements

### Requirement: Badcase and experiment records SHALL expose additive first-error attribution association

Badcase and experiment records MUST be able to associate a versioned first-error attribution by bounded reference and stable corpus item, Badcase, run, step, or experiment correlation. The association MUST remain additive, nullable, and reference-only; historical payloads without attribution MUST retain existing defaults, and attribution drift MUST NOT change existing reproduction status, metric/rubric comparison, experiment aggregation, or continuity comparison semantics.

#### Scenario: Badcase references a valid attribution
- **WHEN** a replayable Badcase has a valid first-error attribution for the same corpus item and run correlation
- **THEN** the normalized Badcase exposes the attribution reference while preserving its existing reproduction status and digest behavior

#### Scenario: Historical Badcase omits attribution
- **WHEN** a historical Badcase or experiment payload contains no first-error attribution fields
- **THEN** normalization and comparison retain existing behavior using documented nullable defaults

#### Scenario: Attribution correlation conflicts
- **WHEN** a referenced attribution identifies a different corpus item, Badcase, run, or first-error step than the owning evaluation record
- **THEN** evaluation reports deterministic attribution-correlation drift and does not rewrite either record

### Requirement: Evidence-linked feedback SHALL remain review-only

Feedback recommendations MAY reference a validated first-error attribution and bounded evidence references, but MUST preserve explicit reviewer identity, decision context, and `pending|approved|rejected` status. Attribution-linked feedback MUST NOT automatically modify prompt, Skill, tool, policy, memory, runtime configuration, code, tests, gates, repository state, or execution decisions.

#### Scenario: Reviewer approves an evidence-linked recommendation
- **WHEN** a reviewer approves a recommendation with valid Badcase or experiment correlation, first-error attribution, bounded evidence, and decision context
- **THEN** evaluation emits an auditable review-only recommendation without applying it to runtime or repository behavior

#### Scenario: Recommendation lacks validated evidence context
- **WHEN** a recommendation cites a missing, conflicting, body-bearing, or unvalidated attribution or evidence reference
- **THEN** validation leaves the recommendation non-actionable and reports deterministic approval or evidence classification

#### Scenario: Approved recommendation is observed by a later run
- **WHEN** a later run can read an approved evidence-linked recommendation
- **THEN** prompt, Skill, tool, policy, memory, runtime configuration, code, tests, gates, and execution decisions remain unchanged unless a separate reviewed change explicitly implements it
