## Purpose

Establish a bounded, opt-in host execution plan contract that binds versioned host descriptions to existing Baymax execution owners, rejects unsupported requests before dispatch, and preserves native admission, result, event, budget and recovery ownership without transferring business authorization to the runtime.

## ADDED Requirements

### Requirement: Versioned immutable plan identity
The bridge SHALL accept only the declared `host_execution_plan.v1` schema with a nonempty plan ID, positive integer plan version, authority digest and tool capability digest. It MUST return an immutable compiled identity/version/canonical digest snapshot. Replanning MUST require a newly supplied version; the bridge MUST NOT mutate an active compiled plan or maintain a global business revision registry.

#### Scenario: Compile a valid plan
- **WHEN** a supported descriptor and matching native binding are compiled
- **THEN** the output contains identity, version and a canonical SHA-256 digest
- **AND** compilation invokes no model, tool, handler, enqueue, publish, claim, commit or diagnostic write

#### Scenario: Unsupported schema or incomplete identity
- **WHEN** schema version is unknown or required identity/digest fields are missing or invalid
- **THEN** compilation rejects with a bounded field-specific error before source activation

#### Scenario: Replan without mutating an existing snapshot
- **WHEN** the host compiles a changed plan with a new version
- **THEN** the new snapshot has its own digest and the previous snapshot remains unchanged
- **AND** the host remains responsible for accepting that revision for business commit

### Requirement: Bounded descriptor and binding snapshots
The bridge MUST enforce limits before unbounded encoding or copying. Default limits SHALL be 64 KiB canonical descriptor bytes, 128 combined structural items, 1024 dependencies, 1024 UTF-8 bytes per reference and nesting depth 8; copied binding contents SHALL have a separate 64 KiB budget and the same depth limit. Zero limit overrides SHALL select defaults, positive overrides SHALL only tighten defaults, and negative or larger overrides MUST fail. Unsupported, cyclic or unbounded input values MUST be rejected without echoing business bodies.

#### Scenario: Exact boundary and one beyond
- **WHEN** otherwise valid inputs reach each declared limit and then exceed it by one
- **THEN** the boundary input is admitted and the excessive input is rejected as limit_exceeded before source activation

#### Scenario: Descriptor references do not bypass binding limits
- **WHEN** a small reference-only descriptor is bound to cyclic, unsupported, over-depth or excessive copied payload
- **THEN** compilation rejects without traversing or cloning unbounded content

#### Scenario: Invalid limit override
- **WHEN** a caller supplies a negative limit or attempts to loosen a default limit
- **THEN** bridge construction fails without silently substituting defaults

### Requirement: Canonical facts and detached mutable input
The digest SHALL cover schema, plan identity/version, axes, host reference/digest declarations, native structural facts, source-normalized executable defaults and compile-time budget facts. Canonicalization MUST be deterministic for map insertion order and MUST preserve semantically relevant task, step and dependency list order. Durations SHALL use integer nanoseconds. Compiled mutable facts MUST be detached from caller maps/slices. Opaque handler/input declarations MUST NOT be presented as authentication of code, input contents or authority.

#### Scenario: Deterministic digest vectors
- **WHEN** equivalent normalized facts differ only in map insertion order
- **THEN** canonical bytes and `sha256:<lowercase hex>` digest match the versioned golden vector

#### Scenario: Semantic facts affect digest
- **WHEN** plan version, authority/tool digest, task order, structure or budget changes
- **THEN** the canonical digest changes without rewriting an existing compiled snapshot

#### Scenario: Caller mutation after compile
- **WHEN** the caller changes descriptor or copied native binding maps/slices after compilation
- **THEN** compiled execution facts and digest remain unchanged
- **AND** non-copyable service/callback references follow the separately documented host trust and concurrency responsibility

### Requirement: Orthogonal axes and conservative capabilities
Execution shape, delivery and trigger SHALL be separate dimensions. Supported shapes SHALL be direct, subagent, team and workflow, with explicit path constraints. The initial bridge SHALL support direct inline, inline teams parallel/vote, flat inline workflow and isolated exclusive local scheduler-backed child dispatch. Fanout SHALL use native teams/workflow composition. Mailbox delivery, resumed-trigger execution, native checkpoint resume, cross-owner durable continuation and shared-queue child dispatch MUST reject before dispatch. Manual/scheduled/callback labels MUST NOT create workers, schedules or authority.

#### Scenario: Scheduled direct execution
- **WHEN** a scheduled-labeled direct inline descriptor has a valid binding and host admission
- **THEN** the native direct owner executes
- **AND** the bridge creates no cron scheduler or additional authorization

#### Scenario: Mailbox or resumed request
- **WHEN** a descriptor requests mailbox delivery, resumed execution or native checkpoint recovery
- **THEN** compilation returns unsupported_capability before enqueue/publish or any new Run
- **AND** reusing a run ID or checkpoint reference does not override the rejection

#### Scenario: Shared scheduler cannot dispatch a specific child
- **WHEN** the requested child binding uses a shared queue or cannot prove exclusive isolated ownership
- **THEN** the bridge rejects before enqueue or claim
- **AND** no unrelated queued task is claimed or mutated

#### Scenario: Local child source streaming limits
- **WHEN** an isolated local child is invoked through bridge Stream
- **THEN** only actual supported native source callbacks are forwarded
- **AND** token streaming or missing source capabilities are not fabricated from a synchronous child result

### Requirement: Full native plan preflight and binding consistency
The bridge MUST validate the whole plan before any source activation and SHALL reuse native validation/normalization for dependency cycles, dangling references, team aggregation/strategy, required callbacks, child guardrails and timeout resolution. Described IDs, ordered structure, strategy, parent correlation and executable budget facts MUST agree with the bound native plan. Source normalization/defaults MUST NOT be duplicated by an independent bridge rule set.

#### Scenario: Later invalid node prevents partial dispatch
- **WHEN** a workflow or team contains an otherwise valid first item and an invalid later item
- **THEN** the bridge rejects the entire plan before any callback, tool, enqueue or source activation

#### Scenario: Cyclic or dangling workflow dependency
- **WHEN** a flat workflow has a cycle or a missing dependency
- **THEN** native validation rejects during compilation without executing earlier steps

#### Scenario: Described facts disagree with binding
- **WHEN** parent reference, IDs, order, strategy or executable budget differs between descriptor and binding
- **THEN** compilation returns binding_mismatch and no source is activated

#### Scenario: Native defaults remain authoritative
- **WHEN** a valid descriptor relies on supported omitted native settings
- **THEN** compilation uses native normalization and matches the direct owner's normalized default facts
- **AND** it does not replace those defaults with an independent bridge policy

### Requirement: Source live admission and budget ownership
Compilation MUST NOT authorize business writes or reserve spendable budget. Actual activation SHALL retain native readiness/action admission, scheduler guardrails, lease checks and live parent budget clamp. Semantic configuration/capability drift that invalidates compilation MUST fail before activation and require explicit recompilation. The bridge MUST NOT add budget accounting, admission caching, retry, cancellation or timeout owners.

#### Scenario: Parent budget shrinks after compile
- **WHEN** a valid compiled child request reaches activation with less available parent budget
- **THEN** the native owner clamps the child timeout to the current available budget or rejects exhausted budget
- **AND** compile-time remaining budget is not used as an authorization to overspend

#### Scenario: Activation policy rejects
- **WHEN** the native owner or host denies current admission
- **THEN** the denial is preserved without handler execution or bridge retry

#### Scenario: Semantic configuration drift
- **WHEN** an executable default or supported capability changes after compilation
- **THEN** the bridge rejects before dispatch and requires explicit recompilation rather than silently changing the compiled digest

### Requirement: Native execution results and pause semantics
The bridge SHALL delegate supported execution to the corresponding existing owner and preserve tagged native results, errors, parent/child references, attempts, budget resolution, aggregation and checkpoint facts. It MUST NOT create a cross-owner terminal reducer or infer business commit. Direct native input_required MUST remain a pause. Team/workflow/local child bindings SHALL be terminal-only in the initial bridge; explicitly required dynamic pause/continuation MUST reject before dispatch. A runner adapter encountering an unexpected pause MUST report unsupported continuation with recoverable source references instead of silently converting it to successful completion.

#### Scenario: Direct input required
- **WHEN** a direct runner produces native input_required
- **THEN** bridge Run and Stream preserve that pause and its native checkpoint references
- **AND** neither adds completed/canceled terminal or automatically resumes

#### Scenario: Composed pause requirement unsupported
- **WHEN** a team, workflow or local child binding explicitly requires dynamic pause or continuation
- **THEN** compilation rejects before source execution

#### Scenario: Terminal-only runner adapter encounters unexpected pause
- **WHEN** a runner-based task adapter unexpectedly receives input_required during an admitted composed execution
- **THEN** it reports unsupported continuation and preserves bounded checkpoint/error references
- **AND** it does not return a successful task result or claim prior effects were rolled back

#### Scenario: Failure cancellation and timeout propagation
- **WHEN** a native execution fails, is canceled or times out
- **THEN** the bridge preserves the native result/error classification and source correlation
- **AND** it neither retries nor creates a second terminal outcome

#### Scenario: Native child completion is not business commit
- **WHEN** scheduler/composer commits a terminal child task
- **THEN** the bridge returns those native task/attempt/commit facts
- **AND** the host business commit gate remains independently required

### Requirement: Real event projection and separate host references
Bridge integration SHALL consume actual source callbacks through the existing native execution event adapter support contract. Plan version/digest SHALL be separate host references; source identity, parent/causation facts and ordering MUST remain source-owned. The bridge MUST NOT fabricate parent/terminal events, global sequence, durable cursor, dedup identity or new diagnostic writes.

#### Scenario: Plan correlation in real callbacks
- **WHEN** a supported owner emits events during bridged execution
- **THEN** the observer receives those real source facts with separate plan version/digest references
- **AND** plan ID can be correlated by the host execution record without changing source event identity

#### Scenario: Parallel sources and observer failure
- **WHEN** parallel children use separate observers and one observer fails
- **THEN** each observer follows its existing source/failure contract
- **AND** the bridge does not introduce global ordering, retry or rewrite the source outcome from observer status

### Requirement: Run Stream conformance and offline evidence
The delivered bridge SHALL include positive, negative and boundary unit tests, real-owner fake-model/tool integration, versioned reference-only canonical/replay fixtures and equivalent shell/PowerShell contract gates. Equivalent supported Run/Stream executions MUST agree on normalized native outcomes, correlation and budget facts. Unsupported paths MUST provide no-side-effect-before-rejection evidence. Replay MUST NOT execute handlers or restore runtime state.

#### Scenario: Supported path conformance
- **WHEN** direct, teams parallel/vote, flat workflow and isolated local child fixtures execute through Run and Stream
- **THEN** normalized source results/correlation/budget facts agree with their native owner paths
- **AND** real event evidence is present according to each source's declared capabilities

#### Scenario: Drift and unsafe rejection fail the gate
- **WHEN** digest/outcome/parent/budget facts drift or a rejected path activates a source
- **THEN** the contract gate fails with a stable fixture-local classification

### Requirement: Public semantic maintenance documentation
The proposal, design and tasks MUST accept identical public semantic documentation commitments. Public API, limits/defaults, errors, supported/unsupported behavior, native ownership, compatibility and rollback MUST be synchronized with maintenance docs and the offline example in the same change. Implementation completion and archive MUST be blocked while documentation or required tests/gates are missing.

#### Scenario: Documentation is incomplete
- **WHEN** implementation changes externally visible behavior without synchronized maintenance documents or matching commitments
- **THEN** documentation gates fail and implementation tasks cannot be declared complete or archived

#### Scenario: Documented offline integration
- **WHEN** the example and bridge are delivered
- **THEN** example README identifies semantic anchors, native runtime paths, capability restrictions, expected markers and rollback
- **AND** the runnable offline example demonstrates both supported execution and pre-dispatch rejection without live provider dependencies

