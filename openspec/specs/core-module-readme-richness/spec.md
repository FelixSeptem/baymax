# core-module-readme-richness Specification

## Purpose
TBD - created by archiving change enforce-status-parity-and-core-module-readme-richness-a25. Update Purpose after archive.
## Requirements
### Requirement: Core modules SHALL provide enriched README documentation with minimum section baseline
The repository MUST maintain enriched README files for core modules with at least these sections:
- 功能域
- 架构设计
- 关键入口
- 边界与依赖
- 配置与默认值
- 可观测性与验证
- 扩展点与常见误用

When a section is not applicable for a module, README MUST explicitly mark it as `N/A` (or equivalent explicit not-applicable marker) instead of omitting it.

Covered module set in initial rollout:
- `a2a/README.md`
- `core/runner/README.md`
- `core/types/README.md`
- `tool/local/README.md`
- `mcp/README.md`
- `model/README.md`
- `context/README.md`
- `orchestration/README.md`
- `runtime/config/README.md`
- `runtime/diagnostics/README.md`
- `runtime/security/README.md`
- `observability/README.md`
- `skill/loader/README.md`

#### Scenario: Contributor opens a core module README
- **WHEN** contributor opens a covered core module README
- **THEN** required section baseline is present and provides direct integration guidance without source-code archaeology

#### Scenario: Section is not applicable for a specific module
- **WHEN** one required section does not apply to module semantics
- **THEN** README explicitly marks that section as not applicable instead of silently omitting it

### Requirement: Core module README entries SHALL remain discoverable from repository README
The repository root README MUST keep discoverable links to covered core module README documents.

#### Scenario: Contributor starts from root README
- **WHEN** contributor reads module documentation index in root README
- **THEN** contributor can navigate to covered core module README files through explicit links

### Requirement: Enriched README checks SHALL be gate-verifiable
Module README richness baseline MUST be verified by repository checks rather than manual-only review.

#### Scenario: Module README loses required section
- **WHEN** a covered module README removes one required section marker
- **THEN** documentation consistency checks fail with explicit module-readme-richness classification

### Requirement: Core Module READMEs SHALL Describe Current State Only
Covered core-module READMEs MUST describe current implementation status and supported pathways, and MUST remove temporary, superseded, or stale milestone narrative from active guidance sections.

When historical context is needed, README MUST link to designated index/archive documentation instead of embedding outdated intermediate-state text.

#### Scenario: Contributor reads module README for onboarding
- **WHEN** contributor opens a covered core-module README
- **THEN** the document reflects current state and does not require filtering obsolete temporary notes

#### Scenario: Historical transition context is required
- **WHEN** module behavior has historical staged evolution
- **THEN** README references canonical archive/index path rather than duplicating temporary timeline narrative

### Requirement: Documentation Paths SHALL Be Canonical and Discoverable
Repository documentation MUST define canonical paths for architecture constraints, roadmap status, and contract index references.

Core READMEs and root README MUST use these canonical paths consistently.

#### Scenario: Contributor follows architecture boundary guidance
- **WHEN** contributor navigates from root or module README to architecture constraints
- **THEN** links resolve to canonical current-state documents without duplicate path variants

#### Scenario: Documentation path drift is introduced
- **WHEN** README references non-canonical or obsolete documentation paths for core governance topics
- **THEN** docs consistency validation MUST fail and require path convergence

### Requirement: Runtime Harness Architecture SHALL Have One Canonical Documentation Entry
Repository MUST maintain one canonical runtime harness architecture document that describes `state surfaces`, `guides/sensors`, `tool mediation`, and `entropy control`, and maps these domains to current contract/gate entrypoints.

Root and module READMEs MUST reference this canonical path instead of duplicating parallel architecture narratives.

#### Scenario: Contributor seeks runtime outer-loop architecture view
- **WHEN** contributor navigates architecture links from root or module README
- **THEN** contributor reaches one canonical runtime harness architecture document with current-state contract/gate mappings

#### Scenario: Parallel architecture narrative drifts from canonical doc
- **WHEN** docs consistency validation detects duplicated or conflicting runtime harness architecture narrative outside canonical path
- **THEN** validation MUST fail and require convergence to the canonical runtime harness architecture document

