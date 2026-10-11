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

### Requirement: Public Semantic Changes SHALL Carry a Documentation Synchronization Commitment
Every active OpenSpec `proposal.md`, `design.md`, and `tasks.md` MUST explicitly accept the repository rule that public semantic changes are documented in the same change. A public semantic change includes changes to public API or contract, runtime behavior, event ordering/identity/terminal semantics, error feedback, configuration defaults or validation, diagnostics fields, CLI/integration behavior, or example semantics.

The commitment MUST declare whether public semantics change. When the declaration is `是`, the artifact MUST identify maintained documentation paths, an owner, and verification evidence in the same change. Each declared documentation path MUST exist and appear in the current change's Git diff against its base branch, including uncommitted and untracked files. When the declaration is `否`, the artifact MUST provide a concrete reason and MUST NOT use additive scope, small code size, or test coverage alone as the reason.

#### Scenario: Public semantic change declares synchronized documentation
- **WHEN** proposal, design, and tasks declare `对外语义变化：是`
- **THEN** each artifact identifies maintained documentation paths, owner, and verification evidence, and the documentation impact gate passes

#### Scenario: Declared documentation exists but was not updated
- **WHEN** an artifact declares `对外语义变化：是` and lists an existing documentation path absent from the current change's Git diff
- **THEN** validation fails with `public-semantic-doc-sync-missing`

#### Scenario: Public semantic change omits synchronization commitment
- **WHEN** any required artifact omits the public semantic documentation commitment
- **THEN** validation fails with `missing-public-semantic-documentation-commitment` before implementation or archive

#### Scenario: Public semantic change declares no impact without a reason
- **WHEN** an artifact declares `对外语义变化：否` without a concrete reason and verification evidence
- **THEN** validation fails with `invalid-public-semantic-documentation-commitment`

#### Scenario: Commitment differs across proposal artifacts
- **WHEN** proposal, design, and tasks disagree on the public semantic change or synchronization state
- **THEN** validation fails with `public-semantic-documentation-drift`
