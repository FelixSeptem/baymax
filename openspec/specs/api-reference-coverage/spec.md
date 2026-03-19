# api-reference-coverage Specification

## Purpose
TBD - created by archiving change improve-dx-d1-api-reference-and-diagnostics-replay-e7. Update Purpose after archive.
## Requirements
### Requirement: Repository SHALL provide D1 API reference coverage for core runtime surfaces
The repository MUST maintain D1-level API reference and minimal usage examples for `core/*`, `runtime/*`, `context/*`, and `skill/*` packages that are part of external integration paths.

Documentation MUST be Chinese-first and MUST accept English examples and descriptions.

#### Scenario: New integrator checks core package usage
- **WHEN** a contributor reads API reference entry docs
- **THEN** the contributor can locate at least one minimal usage example for target `core/runtime/context/skill` package without relying on source-code archaeology

#### Scenario: Maintainer updates externally visible API
- **WHEN** maintainer changes exported behavior in covered package set
- **THEN** corresponding API reference docs are updated in the same change or explicitly marked with tracked follow-up

### Requirement: API reference entry points SHALL remain discoverable from README/docs index
The repository MUST expose a stable entry path from `README.md` to D1 API reference materials, including package-scope navigation hints for `core/runtime/context/skill`.

#### Scenario: Contributor starts from repository homepage
- **WHEN** contributor opens `README.md`
- **THEN** contributor can navigate to D1 API reference materials through explicit documentation links

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

