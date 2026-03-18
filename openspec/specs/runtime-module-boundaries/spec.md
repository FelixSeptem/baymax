# runtime-module-boundaries Specification

## Purpose
TBD - created by archiving change refactor-runtime-responsibility-boundaries-and-enrich-docs. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL define explicit module boundaries
The system MUST define explicit boundaries between global runtime platform capabilities and MCP-specific capabilities, with documented ownership and allowed dependency directions.

#### Scenario: Developer evaluates module responsibility
- **WHEN** developer checks runtime architecture documentation
- **THEN** documentation clearly identifies which package owns global config lifecycle versus MCP policy semantics

### Requirement: Runtime modules SHALL enforce one-way dependency direction
Global runtime packages MUST NOT depend on MCP transport packages, and MCP packages MUST consume runtime configuration and diagnostics APIs via stable interfaces. MCP shared reliability internals MUST be restricted to `mcp/internal/*` and MUST NOT be imported by non-MCP packages.

#### Scenario: Build-time dependency check
- **WHEN** static dependency checks run in CI
- **THEN** no import cycle or reverse dependency from global runtime package to MCP package is allowed, and no non-MCP package imports `mcp/internal/*`

### Requirement: Runtime refactor SHALL provide migration compatibility guidance
The system MUST provide migration mapping for package moves, deprecation notes, and replacement API examples.

#### Scenario: User upgrades from previous structure
- **WHEN** user follows migration guide from old MCP 单体 runtime 包结构
- **THEN** user can locate equivalent function-scoped package APIs and complete migration without behavior ambiguity

### Requirement: Runtime diagnostics API SHALL be owned by global runtime layer
The system MUST place diagnostics API ownership in the global runtime layer, while MCP layer keeps only MCP-specific diagnostic field semantics.

#### Scenario: Multi-subsystem diagnostics access
- **WHEN** runner, tool, skill, MCP, or observability components request runtime diagnostics
- **THEN** they use a shared diagnostics API surface without importing MCP transport packages

### Requirement: Documentation SHALL include architecture and ownership reference
The repository MUST include a maintained architecture page with module responsibilities, ownership hints, extension constraints, and internal/shared versus transport-specific MCP layering guidance.

#### Scenario: Contributor plans new runtime feature
- **WHEN** contributor proposes a new runtime capability
- **THEN** contributor can determine target module and boundary constraints from documentation without ad hoc clarification

### Requirement: MCP architecture docs SHALL describe shared-core layering
Documentation MUST explicitly describe MCP shared-core and transport-specific layering, including examples of which logic belongs in shared internal modules versus `mcp/http` or `mcp/stdio` packages.

#### Scenario: Contributor modifies MCP retry behavior
- **WHEN** contributor changes retry behavior for MCP clients
- **THEN** docs indicate that semantic retry logic belongs to shared internal MCP core and transport-specific code only provides protocol hooks

### Requirement: Boundary reviews SHALL verify context-model responsibility split
Boundary governance reviews MUST verify that `context/*` packages orchestrate policy only, while provider SDK protocol actions remain in `model/*` packages and are consumed via interfaces.

#### Scenario: Token counting path is reviewed
- **WHEN** reviewer inspects context assembly token-count flow
- **THEN** context layer invokes model-facing interfaces and does not import provider SDK packages directly

### Requirement: Boundary reviews SHALL validate dependency and semantic direction together
Boundary checks MUST include both import-direction validation and semantic responsibility validation for cross-module orchestration paths.

For R4 multi-agent domains, boundary governance MUST include a blocking shared-contract consistency gate. Changes touching Teams/Workflow/A2A specs or implementation MUST pass this gate before merge.

Minimum gate checks for this milestone:
- unified status mapping compliance (including `a2a.submitted -> pending`),
- reason code namespace compliance (`team.*|workflow.*|a2a.*`),
- canonical `peer_id` naming compliance.

#### Scenario: Multi-agent change violates reason namespace
- **WHEN** a change introduces reason code outside approved multi-agent namespaces
- **THEN** shared-contract gate fails and the change is blocked from merge

#### Scenario: Multi-agent change uses non-canonical A2A peer field
- **WHEN** a change emits remote peer field as non-`peer_id` key
- **THEN** shared-contract gate fails and the change is blocked from merge

### Requirement: Boundary governance outcomes SHALL be reflected in architecture docs
When module responsibility corrections are made, architecture and boundary documentation MUST be updated in the same change to preserve a single source of truth.

For R4 multi-agent scope, `docs/multi-agent-identifier-model.md` MUST be treated as the shared contract source for identifier, status mapping, and reason namespace conventions.

#### Scenario: Multi-agent contract is updated
- **WHEN** teams/workflow/a2a shared identifier or reason conventions change
- **THEN** architecture and shared contract docs are updated in the same change set

### Requirement: Teams orchestration SHALL preserve runner-core boundary stability
Teams orchestration logic MUST be implemented outside `core/runner` and consumed via explicit interfaces so the runner main state machine remains focused on single-run loop semantics.

#### Scenario: Contributor introduces team orchestration logic
- **WHEN** a change implements Teams collaboration behavior
- **THEN** the implementation resides in the designated orchestration module and does not add cross-agent state transitions directly inside `core/runner`

### Requirement: Boundary checks SHALL cover Teams ownership rules
Boundary governance checks MUST verify both import direction and semantic ownership for Teams modules, including event emission and diagnostics write-path constraints.

Teams change implementation MUST pass shared multi-agent contract gate before merge, including:
- status mapping consistency (with unified semantic layer),
- reason namespace consistency (`team.*|workflow.*|a2a.*`),
- canonical peer-field naming consistency (`peer_id` for A2A-related references).

#### Scenario: Teams module emits diagnostics
- **WHEN** Teams implementation adds observability output
- **THEN** output flows through `observability/event.RuntimeRecorder` without introducing direct diagnostics store writes

### Requirement: Workflow engine SHALL remain decoupled from runner core state machine
Workflow orchestration MUST be implemented in a dedicated module and MUST consume runner/tool/mcp/skill capabilities through interfaces, without embedding workflow dependency-graph logic directly in `core/runner`.

#### Scenario: Contributor adds workflow orchestration support
- **WHEN** contributor implements workflow planning and scheduling behavior
- **THEN** dependency-graph and scheduling logic are placed in the workflow module, and `core/runner` retains single-run loop responsibility

### Requirement: Boundary governance SHALL verify workflow ownership and diagnostics write path
Boundary checks MUST verify workflow ownership and ensure workflow observability writes continue to use `observability/event.RuntimeRecorder` as the single diagnostics write entry.

Workflow change implementation MUST pass shared multi-agent contract gate before merge, including:
- status mapping consistency (with unified semantic layer),
- reason namespace consistency (`team.*|workflow.*|a2a.*`),
- canonical peer-field naming consistency (`peer_id` for A2A-related references).

#### Scenario: Workflow observability is added
- **WHEN** workflow implementation emits run and step observability data
- **THEN** data is recorded through the single-writer event path and not by direct diagnostics store mutation

### Requirement: Boundary governance SHALL enforce A2A and MCP responsibility separation
Architecture boundaries MUST enforce that A2A modules own inter-agent collaboration semantics while MCP modules own tool-integration semantics, with no semantic overlap in responsibility.

#### Scenario: Contributor adds cross-agent request feature
- **WHEN** a feature implements peer-agent task lifecycle operations
- **THEN** implementation is placed in A2A module scope and not in MCP transport packages

### Requirement: A2A modules SHALL consume runtime observability through the shared single-writer path
A2A modules MUST emit events and diagnostics through the same `observability/event.RuntimeRecorder` single-writer path used by existing runtime components.

A2A change implementation MUST pass shared multi-agent contract gate before merge, including:
- status mapping consistency (including `submitted -> pending` mapping at semantic layer),
- reason namespace consistency (`team.*|workflow.*|a2a.*`),
- canonical peer-field naming consistency (`peer_id`).

#### Scenario: A2A module adds diagnostic output
- **WHEN** A2A module records lifecycle outcomes
- **THEN** diagnostics flow through the shared event recorder path and do not introduce direct diagnostics store writes

### Requirement: Boundary governance SHALL keep A2A delivery/version negotiation outside MCP scope
A2A delivery-mode negotiation and Agent Card version negotiation MUST be implemented in A2A module scope and MUST NOT be implemented in MCP transport packages.

#### Scenario: Contributor adds delivery negotiation logic
- **WHEN** a change introduces A2A delivery-mode negotiation
- **THEN** implementation resides in `a2a/*` scope and not in `mcp/http` or `mcp/stdio`

#### Scenario: Contributor adds version negotiation logic
- **WHEN** a change introduces Agent Card version compatibility checks
- **THEN** implementation resides in A2A module scope and not MCP transport scope

### Requirement: A2A delivery/version observability SHALL use shared single-writer path
A2A delivery/version observability writes MUST flow through `observability/event.RuntimeRecorder` and MUST NOT bypass runtime diagnostics single-writer path.

#### Scenario: A2A delivery fallback emits diagnostics
- **WHEN** runtime records fallback behavior for A2A delivery
- **THEN** records are ingested through shared runtime recorder path without direct diagnostics store writes

#### Scenario: A2A version mismatch emits diagnostics
- **WHEN** runtime records version mismatch outcome
- **THEN** records are ingested through shared runtime recorder path with replay-idempotent semantics

### Requirement: Shared contract gate SHALL validate A2A delivery/version naming consistency
Changes touching A2A delivery/version semantics MUST pass shared multi-agent contract gate for naming consistency.

Minimum checks for this milestone:
- reason namespace consistency (`a2a.*`),
- canonical peer-field naming consistency (`peer_id`),
- delivery/version field naming consistency (snake_case additive fields).

#### Scenario: Non-canonical peer field is introduced
- **WHEN** a change emits A2A peer field using key other than `peer_id`
- **THEN** shared contract gate fails and blocks merge

#### Scenario: Non-namespaced A2A reason is introduced
- **WHEN** a change emits A2A timeline reason without `a2a.*` namespace
- **THEN** shared contract gate fails and blocks merge

### Requirement: Boundary governance SHALL enforce orchestration-to-A2A integration ownership
Composed orchestration features MUST be implemented in orchestration and A2A modules through explicit interfaces, and MUST NOT move peer-collaboration semantics into MCP transport packages.

#### Scenario: Contributor adds workflow remote-step integration
- **WHEN** a change adds workflow remote execution via A2A
- **THEN** implementation remains in orchestration/A2A ownership scope and does not introduce MCP responsibility overlap

### Requirement: Boundary governance SHALL preserve single-writer diagnostics path in composed flows
Composed orchestration observability MUST continue to use `observability/event.RuntimeRecorder` as the only diagnostics write path.

#### Scenario: Composed feature emits new diagnostics fields
- **WHEN** workflow/teams/A2A integration emits additive diagnostics output
- **THEN** writes flow through single-writer recorder and do not directly mutate diagnostics store

### Requirement: Shared contract gate SHALL cover composed orchestration consistency checks
Shared multi-agent contract validation MUST include composed orchestration checks for identifier mapping, reason namespace compliance, and canonical `peer_id` naming.

#### Scenario: Composed change introduces non-canonical field naming
- **WHEN** a composed orchestration change emits non-`peer_id` peer field or non-namespaced reason
- **THEN** shared-contract gate fails and blocks merge

### Requirement: Scheduler module SHALL preserve boundary ownership separation
Distributed scheduler implementation MUST reside in dedicated scheduler/orchestration ownership scope and MUST NOT move peer-collaboration semantics into MCP transport packages.

#### Scenario: Contributor adds lease-claim logic
- **WHEN** a change introduces lease claim and takeover behavior
- **THEN** implementation is placed in scheduler/orchestration scope and not in `mcp/http` or `mcp/stdio`

### Requirement: Scheduler observability SHALL use shared single-writer diagnostics path
Scheduler and subagent observability output MUST enter diagnostics through `observability/event.RuntimeRecorder` single-writer path only.

#### Scenario: Scheduler emits queue and lease metrics
- **WHEN** scheduler emits queue/lease lifecycle events
- **THEN** diagnostics ingestion occurs through RuntimeRecorder and not via direct diagnostics store writes

### Requirement: Shared contract governance SHALL include scheduler/subagent consistency checks
Shared multi-agent contract gate MUST validate scheduler/subagent identifier mapping and reason namespace compliance in addition to existing team/workflow/a2a checks.

#### Scenario: Change introduces non-namespaced scheduler reason
- **WHEN** scheduler/subagent change emits reason outside approved namespace conventions
- **THEN** shared-contract gate fails and blocks merge

### Requirement: Shared contract gate SHALL include scheduler/subagent closure checks
Runtime boundary governance MUST extend shared-contract gate checks to include scheduler/subagent namespace and correlation requirements.

Minimum additional checks:
- scheduler/subagent reason namespace compliance,
- attempt-level correlation field presence,
- single-writer diagnostics path compliance.

#### Scenario: Non-canonical scheduler reason is introduced
- **WHEN** a change emits scheduler/subagent reason outside canonical taxonomy
- **THEN** shared-contract gate fails and blocks merge

### Requirement: Composer module SHALL remain orchestration-layer glue only
The `orchestration/composer` module MUST remain an orchestration-layer composition boundary and MUST NOT absorb provider protocol logic, diagnostics storage writes, or MCP transport internals.

#### Scenario: Composer integrates runner and scheduler
- **WHEN** composer wires runner and scheduler for a composed execution path
- **THEN** composer only coordinates module integration and does not bypass existing ownership boundaries

### Requirement: Composer and scheduler SHALL use RuntimeRecorder as single diagnostics write path
Composer- and scheduler-related observability data MUST be emitted as standard events and MUST be persisted through `observability/event.RuntimeRecorder` single-writer path.

#### Scenario: Composer-managed run emits completion summary
- **WHEN** composer-managed execution completes
- **THEN** diagnostics persistence occurs through RuntimeRecorder event ingestion and not through direct `runtime/diagnostics` writes from orchestration modules

### Requirement: Recovery orchestration SHALL remain in orchestration and runtime domains
Recovery orchestration logic MUST remain within orchestration/runtime composition boundaries and MUST NOT introduce forbidden dependencies on MCP internal packages or direct diagnostics storage writes.

#### Scenario: Recovery feature is integrated across modules
- **WHEN** recovery integration updates orchestration and runtime packages
- **THEN** dependency direction and single-writer diagnostics boundary constraints remain satisfied

### Requirement: Recovery persistence SHALL preserve single-writer diagnostics discipline
Recovery execution and replay observability MUST be emitted as standard events and persisted only through RuntimeRecorder single-writer ingestion.

Required observability contract:
- recovery timeline reason namespace uses `recovery.*`
- recovery additive run summary fields are emitted via `run.finished` payload only

#### Scenario: Recovery replay updates run observability
- **WHEN** recovery replay path records timeline and run summary signals
- **THEN** persistence happens through event ingestion and not direct writes from recovery orchestrators

### Requirement: Scheduler QoS governance SHALL remain inside scheduler/runtime boundaries
QoS/fairness/dead-letter governance MUST remain implemented within scheduler/runtime boundaries and MUST NOT introduce forbidden dependencies or direct diagnostics store writes.

#### Scenario: QoS features are added to scheduler module
- **WHEN** scheduler QoS and dead-letter logic is implemented
- **THEN** dependency direction remains compliant with module boundary contract

### Requirement: QoS observability SHALL keep RuntimeRecorder single-writer path
Scheduler QoS and dead-letter observability MUST be emitted as standard events and persisted through RuntimeRecorder single-writer ingestion.

#### Scenario: QoS transitions update diagnostics
- **WHEN** qos/fairness/dlq events are emitted
- **THEN** diagnostics persistence uses RuntimeRecorder mapping and not direct writes from scheduler state logic

