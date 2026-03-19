## ADDED Requirements

### Requirement: Governance documentation SHALL avoid implicit stable-release claims during pre-1 phase
When repository release stage remains `0.x`, contributor-facing governance documents MUST NOT imply `1.0`, `stable`, or `prod-ready` commitments unless a dedicated release-governance change explicitly upgrades the stage.

This requirement applies to roadmap and versioning policy as source-of-truth entries.

#### Scenario: Maintainer edits roadmap milestones
- **WHEN** roadmap milestone text is updated while repository is still in `0.x`
- **THEN** roadmap language remains compatible with pre-1 compatibility posture and does not imply stable-release guarantees

#### Scenario: Maintainer proposes explicit stage upgrade
- **WHEN** maintainers decide to move from `0.x` to explicit stable-release commitment
- **THEN** governance change is required before source-of-truth docs can claim `1.0/prod-ready` posture

### Requirement: Pre-1 governance SHALL define long-term direction as non-execution scope
Governance documentation in pre-1 phase MUST keep long-term platformization directions as non-execution scope and MUST not place them in near-term active milestones.

#### Scenario: Roadmap includes long-term platformization themes
- **WHEN** maintainer documents platformization themes such as control plane or multi-tenant governance
- **THEN** those themes are marked as long-term directions and excluded from near-term execution scope
