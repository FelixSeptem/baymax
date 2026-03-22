## ADDED Requirements

### Requirement: Quality gate SHALL block legacy direct invoke API reintroduction
The shared multi-agent quality gate and default quality gate MUST include canonical-only checks that block:
- re-exposing legacy direct invoke public APIs for sync/async orchestration paths,
- reintroducing cross-module usage that bypasses mailbox canonical entrypoints.

Canonical-only checks MUST be treated as blocking validation in both shell and PowerShell quality workflows.

#### Scenario: Change reintroduces direct invoke public API surface
- **WHEN** validation detects legacy direct invoke APIs are reintroduced as supported public entrypoints
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Change keeps mailbox canonical entrypoints only
- **WHEN** validation confirms sync/async/delayed orchestration calls route through mailbox canonical entrypoints
- **THEN** canonical-only checks pass without introducing additional failures
