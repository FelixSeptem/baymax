## ADDED Requirements

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
