## ADDED Requirements

### Requirement: Hooks Middleware Contract Gate Integration
Quality gate MUST include `check-hooks-middleware-contract.sh/.ps1` to enforce A65 contract suites and MUST fail fast on non-zero native command exits.

#### Scenario: Shell gate fail-fast
- **WHEN** `check-hooks-middleware-contract.sh` returns non-zero
- **THEN** `check-quality-gate.sh` MUST fail the pipeline without soft fallback

#### Scenario: PowerShell gate fail-fast parity
- **WHEN** `check-hooks-middleware-contract.ps1` returns non-zero
- **THEN** `check-quality-gate.ps1` MUST fail with equivalent blocking semantics

### Requirement: A65 Impacted Contract Suites Enforcement
Gate execution MUST enforce impacted contract suites per changed A65 module scope, and MUST reject merges when any required suite is missing or failing.

#### Scenario: Runner scope change requires security suites
- **WHEN** A65 changes touch runner lifecycle or dispatch boundaries
- **THEN** gate MUST require relevant security contract suites before allowing merge

#### Scenario: Skill scope change requires replay and skill suites
- **WHEN** A65 changes touch discovery/preprocess/bundle mapping paths
- **THEN** gate MUST require replay and skill-related suites before allowing merge

#### Scenario: Observability scope change requires export and replay suites
- **WHEN** A65 changes touch diagnostics or recorder mapping paths
- **THEN** gate MUST require observability export and diagnostics replay suites before allowing merge
