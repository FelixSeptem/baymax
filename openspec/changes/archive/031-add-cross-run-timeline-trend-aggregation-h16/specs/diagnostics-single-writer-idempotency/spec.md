## ADDED Requirements

### Requirement: Cross-run trend aggregation SHALL remain idempotent under replay and duplicate submissions
Cross-run trend aggregates MUST be computed from idempotent run diagnostics records so replayed or duplicated timeline submissions do not increase logical counts or distort latency distributions.

#### Scenario: Duplicate timeline replay for an already-recorded run
- **WHEN** timeline events for a previously recorded run are replayed or duplicated
- **THEN** cross-run trend counts and latency aggregates remain unchanged

#### Scenario: Concurrent duplicate submissions for same run
- **WHEN** multiple goroutines submit duplicate diagnostics for the same run concurrently
- **THEN** cross-run trend aggregates reflect one logical run sample only
