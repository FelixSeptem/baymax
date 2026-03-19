## ADDED Requirements

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
