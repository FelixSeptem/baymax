## ADDED Requirements

### Requirement: Agent Mode Examples SHALL Be Organized Under a Unified Matrix
The repository MUST provide a unified `examples/agent-modes` entrypoint and a matrix index that maps each mode to its runnable examples, required contracts, and required gates.

#### Scenario: Matrix defines full mapping for each mode
- **WHEN** a maintainer inspects `examples/agent-modes/MATRIX.md`
- **THEN** each mode includes `minimal`, `production-ish`, `contracts`, `gates`, and replay/diagnostics coverage references

### Requirement: PocketFlow and Baymax Mode Families SHALL Be Fully Covered
The example pack MUST cover PocketFlow mode families (`agent/workflow/rag/mapreduce/structured-output/multi-agents`) and Baymax extension families (`skill/mcp/react/hitl/context/sandbox/realtime`).

#### Scenario: Mode family coverage is complete
- **WHEN** pattern coverage validation runs
- **THEN** missing required PocketFlow or Baymax mode families fail validation deterministically

### Requirement: Each Mode SHALL Provide Minimal and Production-ish Variants
Each required mode MUST provide both `minimal` and `production-ish` runnable variants; production-ish variants MUST demonstrate governance-path behavior rather than only happy-path execution.

#### Scenario: Minimal and production-ish both exist
- **WHEN** smoke and coverage gates evaluate a required mode
- **THEN** both variants are present and executable in the expected runtime path

### Requirement: Mainline Flow Examples SHALL Cover Canonical Orchestration Paths
The example pack MUST include canonical mainline flow coverage for mailbox (`sync/async/delayed/reconcile`), task-board (`query/control`), scheduler (`qos/backoff/dead-letter`), and readiness/admission degradation semantics.

#### Scenario: Mainline path coverage is validated
- **WHEN** mainline-focused examples are executed
- **THEN** diagnostics/replay evidence confirms canonical path behavior without introducing alternate state sources

### Requirement: Custom Adapter Examples SHALL Cover Four Adapter Domains and Health Circuit
The example pack MUST include custom adapter onboarding coverage for `mcp/model/tool/memory` domains and health-readiness-circuit governance path.

#### Scenario: Adapter domain coverage is complete
- **WHEN** adapter-focused example coverage is validated
- **THEN** all four adapter domains and health-readiness-circuit path are present and mapped to corresponding conformance/contract gates

### Requirement: Example Outputs SHALL Reuse Existing Contract Semantics
Examples MUST reuse existing contract outputs and MUST NOT define parallel semantics for diagnostics fields, reason taxonomy, decision traces, or replay schemas.

#### Scenario: Example semantics remain contract-compatible
- **WHEN** diagnostics/replay assertions are evaluated for example runs
- **THEN** outputs remain additive-compatible with existing contract parsers and drift classifications

### Requirement: Migration Playbook SHALL Define Example-to-Production Checklist
The example pack MUST provide `examples/agent-modes/PLAYBOOK.md` that defines migration checkpoints for config, permission, observability, capacity, rollback, and required gates.

#### Scenario: Migration playbook covers required production checkpoints
- **WHEN** maintainers review the playbook
- **THEN** all required checkpoint categories are explicitly defined with actionable verification steps

### Requirement: Production-ish Readmes SHALL Declare Prod Delta Checklist
Each `production-ish` example README MUST include a `prod delta` checklist describing differences from `minimal`, risk boundaries, and mandatory gates.

#### Scenario: Production-ish readme includes prod delta
- **WHEN** playbook consistency validation inspects production-ish example readmes
- **THEN** missing `prod delta` sections fail validation with explicit missing-checklist classification

### Requirement: Context-Governed Example Completion SHALL Depend on A69 Production Convergence
The `context-governed-reference-first` mode in A62 MUST reuse A69 context compression production outputs and MUST NOT define independent acceptance semantics before A69 convergence.

#### Scenario: Context-governed validation requires A69 contract outputs
- **WHEN** A62 validates context-governed example behavior
- **THEN** validation references A69 gate and replay outputs for compression/tiering/swap-back governance evidence

#### Scenario: Non-context modes continue without A69 completion dependency
- **WHEN** A62 change only touches non-context mode families
- **THEN** those modes MAY proceed independently without waiting for A69 final completion state

### Requirement: Future Contract Proposals SHALL Declare Example Impact Assessment
For subsequent proposals that change runtime behavior, configuration semantics, diagnostics schema, or contract expectations, proposal artifacts MUST explicitly declare example impact assessment.

Allowed declaration outcomes are:
- `新增示例`
- `修改示例`
- `无需示例变更（附理由）`

#### Scenario: Proposal with behavior or contract change declares example impact
- **WHEN** a maintainer drafts a new proposal that changes behavior/config/contract semantics
- **THEN** proposal artifacts include one of the allowed declaration outcomes and corresponding example scope or rationale

#### Scenario: Proposal omits required example impact declaration
- **WHEN** a proposal changes behavior/config/contract semantics but does not include example impact assessment
- **THEN** proposal is treated as incomplete and MUST be updated before approval

### Requirement: Legacy Example TODO Placeholders SHALL Be Eliminated
Historical examples under `examples/` MUST NOT contain unresolved placeholder markers such as `TODO`, `TBD`, `FIXME`, or equivalent pending tags.

#### Scenario: Legacy examples are scanned for placeholder markers
- **WHEN** legacy example cleanup validation scans existing example files
- **THEN** any unresolved placeholder marker fails validation and blocks merge

#### Scenario: Deferred work is tracked outside inline placeholders
- **WHEN** an unfinished example concern is discovered during A62 migration
- **THEN** it is tracked through `MATRIX.md`, `PLAYBOOK.md`, or `tasks.md` instead of inline TODO-style markers
