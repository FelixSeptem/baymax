## ADDED Requirements

### Requirement: Diagnostics-query benchmark SHALL include sandbox rollout-enriched dataset profile
Diagnostics-query benchmark matrix MUST include sandbox rollout-enriched dataset profile that covers additive fields introduced by rollout governance.

At minimum, dataset profile MUST include records with:
- mixed rollout phases (`observe|canary|baseline|full|frozen`)
- mixed capacity actions (`allow|throttle|deny`)
- mixed health budget states (`within_budget|near_budget|breached`)

#### Scenario: Contributor executes diagnostics-query benchmark with sandbox-enriched profile
- **WHEN** benchmark suite runs with default diagnostics profile set
- **THEN** output includes sandbox rollout-enriched query categories and deterministic metric collection

#### Scenario: CI executes sandbox-enriched diagnostics benchmark
- **WHEN** quality gate runs diagnostics performance regression checks
- **THEN** sandbox rollout-enriched profile is included without external dependency requirements

### Requirement: Diagnostics-query regression gate SHALL enforce thresholds for sandbox-enriched query paths
Regression gate MUST enforce documented relative-threshold policy for sandbox-enriched query paths using the same deterministic baseline comparison semantics as existing diagnostics-query checks.

#### Scenario: Sandbox-enriched query p95 regression exceeds threshold
- **WHEN** one or more sandbox-enriched query paths exceed configured `p95-ns/op` regression threshold
- **THEN** diagnostics-query regression gate fails and blocks validation

#### Scenario: Sandbox-enriched query metrics remain within thresholds
- **WHEN** sandbox-enriched query candidate metrics stay within configured thresholds
- **THEN** diagnostics-query regression gate passes without blocking validation
