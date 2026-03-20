## ADDED Requirements

### Requirement: Negotiation outcomes SHALL be profile-versioned and replay-verifiable
Capability negotiation outcomes and reason taxonomy outputs MUST be replay-verifiable under declared `contract_profile_version`.

Changes to negotiation semantics across profile versions MUST be represented by explicit fixture deltas.

#### Scenario: Negotiation replay under same profile
- **WHEN** replay runs against fixtures for the same `contract_profile_version`
- **THEN** negotiation outcomes and reason taxonomy remain deterministic and match baseline

#### Scenario: Negotiation semantics change for new profile
- **WHEN** maintainer updates negotiation behavior for a new profile version
- **THEN** fixtures and profile metadata are updated in the same change with explicit replay diff
