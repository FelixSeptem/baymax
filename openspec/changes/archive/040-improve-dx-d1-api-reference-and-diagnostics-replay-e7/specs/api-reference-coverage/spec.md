## ADDED Requirements

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
