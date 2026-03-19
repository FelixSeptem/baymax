# release-status-parity-governance Specification

## Purpose
TBD - created by archiving change enforce-status-parity-and-core-module-readme-richness-a25. Update Purpose after archive.
## Requirements
### Requirement: Repository SHALL maintain release status parity between OpenSpec and contributor-facing progress docs
When reporting project progress, repository documentation MUST keep status parity with OpenSpec authority sources:
- active changes from `openspec list --json`,
- archived changes from `openspec/changes/archive/INDEX.md`.

At minimum, roadmap and README milestone snapshot entries MUST not conflict with active/archived status semantics.

#### Scenario: Change is archived in OpenSpec index
- **WHEN** a change appears in `openspec/changes/archive/INDEX.md`
- **THEN** roadmap and README progress summaries MUST NOT describe that change as active or in-progress

#### Scenario: Change is active in OpenSpec list
- **WHEN** a change appears as active in `openspec list --json`
- **THEN** roadmap and README progress summaries MUST NOT describe that change as archived or stable-frozen

### Requirement: Status parity checks SHALL fail fast on semantic drift
If release/progress docs contain semantic conflict with OpenSpec authority status, repository validation MUST fail fast with deterministic non-zero exit status.

#### Scenario: Roadmap marks archived change as in-progress
- **WHEN** status parity check detects conflict between roadmap text and OpenSpec archive status
- **THEN** validation exits non-zero and blocks completion

#### Scenario: README snapshot omits current active change status updates
- **WHEN** status parity check detects README milestone snapshot does not reflect active change posture
- **THEN** validation exits non-zero and reports status-parity classification

