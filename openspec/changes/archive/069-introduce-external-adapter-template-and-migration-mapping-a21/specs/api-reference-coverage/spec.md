## ADDED Requirements

### Requirement: API reference SHALL expose external adapter template entry points
Repository API reference materials MUST include discoverable entry points for MCP, Model, and Tool adapter templates.

Entry links MUST be reachable from README and docs index navigation.

#### Scenario: New contributor starts from README
- **WHEN** contributor opens repository README
- **THEN** contributor can navigate to external adapter templates through explicit documentation links

#### Scenario: Contributor opens API reference index
- **WHEN** contributor inspects docs API reference navigation
- **THEN** adapter template sections for MCP/Model/Tool are present and discoverable

### Requirement: API reference SHALL include adapter migration mapping index
API reference docs MUST include a dedicated migration mapping index covering capability domains and code-snippet mapping entries.

#### Scenario: Contributor updates adapter-facing API docs
- **WHEN** maintainer changes adapter-related integration guidance
- **THEN** migration mapping index is updated in the same change or tracked with explicit follow-up

#### Scenario: Integrator searches for migration guidance
- **WHEN** integrator navigates API reference materials
- **THEN** integrator finds domain-based and snippet-based migration mappings without scanning unrelated docs

