# proposal-governance-automation-contract Specification

## Purpose
TBD - created by archiving change introduce-governance-automation-and-consistency-gate-contract-a70. Update Purpose after archive.
## Requirements
### Requirement: Governance Status Source of Truth SHALL Be Deterministic
Repository governance status evaluation MUST use deterministic sources:
- active changes from `openspec list --json`,
- archived changes from `openspec/changes/archive/INDEX.md`,
- roadmap projection from `docs/development-roadmap.md`.

#### Scenario: Roadmap status matches OpenSpec sources
- **WHEN** governance consistency check evaluates current repository state
- **THEN** active/archive/candidate status mapping in roadmap is consistent with OpenSpec sources

#### Scenario: Roadmap status drifts from OpenSpec sources
- **WHEN** roadmap marks a proposal status that conflicts with OpenSpec active or archive state
- **THEN** governance check fails with deterministic `roadmap-status-drift` classification

### Requirement: Proposal Example Impact Declaration SHALL Be Mandatory
For subsequent proposals that introduce behavior, configuration, diagnostics, or contract-facing changes, proposal artifacts MUST include an explicit `example impact assessment` declaration.

Allowed declaration values are:
- `新增示例`
- `修改示例`
- `无需示例变更（附理由）`

#### Scenario: Proposal contains valid example impact declaration
- **WHEN** governance check evaluates proposal artifacts with one allowed declaration value
- **THEN** validation passes for example-impact declaration requirement

#### Scenario: Proposal omits required example impact declaration
- **WHEN** governance check evaluates proposal artifacts without required declaration
- **THEN** validation fails with deterministic `missing-example-impact-declaration` classification

#### Scenario: Proposal uses unsupported declaration value
- **WHEN** declaration value is not one of the allowed values
- **THEN** validation fails with deterministic `invalid-example-impact-value` classification

### Requirement: Governance Checks SHALL Emit Auditable Failure Taxonomy
Governance automation checks MUST emit stable machine-readable reason codes and human-readable remediation hints.

#### Scenario: Multiple governance violations are present
- **WHEN** governance checks detect multiple violations in one run
- **THEN** output includes all violations in deterministic order with stable reason codes

