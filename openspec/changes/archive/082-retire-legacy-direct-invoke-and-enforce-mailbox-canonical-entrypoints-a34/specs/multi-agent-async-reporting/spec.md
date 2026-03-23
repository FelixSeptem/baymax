## MODIFIED Requirements

### Requirement: Legacy direct report-sink API SHALL be deprecated
Legacy direct async report-sink contract from pre-mailbox path MUST NOT remain a supported public canonical contract surface.

Mailbox result delivery MUST be the only supported canonical async invoke/reporting path for mainline multi-agent orchestration.

#### Scenario: Maintainer validates async contract entrypoint
- **WHEN** maintainer reviews async reporting mainline contract
- **THEN** mailbox result delivery is canonical and legacy direct async invoke/report-sink path is excluded from supported public entrypoints

#### Scenario: Legacy direct async entrypoint is referenced as canonical path
- **WHEN** repository change reintroduces direct async invoke/report-sink API as canonical public usage
- **THEN** contract validation treats this as regression and blocks completion
