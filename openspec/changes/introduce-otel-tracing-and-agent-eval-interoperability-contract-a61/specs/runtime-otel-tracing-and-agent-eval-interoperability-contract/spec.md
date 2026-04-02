## ADDED Requirements

### Requirement: Runtime SHALL provide collector-first OTel tracing interoperability
Runtime MUST provide collector-first OTel tracing interoperability for agent runtime flows.

Minimum guarantees:
- canonical export contract targets OTLP-compatible collector ingestion,
- trace export behavior is configurable without code changes,
- export failures map to deterministic reason taxonomy.

#### Scenario: Runtime enables OTel tracing collection
- **WHEN** effective tracing configuration enables OTel export to a reachable collector endpoint
- **THEN** runtime exports canonical tracing payloads that are collector-compatible without custom per-backend patching

#### Scenario: Collector endpoint is unavailable
- **WHEN** runtime attempts OTel export and collector endpoint is unreachable
- **THEN** runtime emits deterministic export failure classification and preserves configured on-error policy semantics

### Requirement: Runtime SHALL freeze canonical span topology and attribute mapping for core agent domains
Runtime MUST freeze canonical span topology and attribute mapping for at least:
- `run`
- `model`
- `tool`
- `mcp`
- `memory`
- `hitl`

Equivalent inputs under unchanged effective config MUST produce semantically equivalent span topology and canonical attributes.

#### Scenario: Run executes model and tool steps
- **WHEN** a managed run performs model reasoning and tool invocation
- **THEN** emitted spans preserve canonical parent-child topology and required semantic attributes

#### Scenario: Equivalent inputs are traced repeatedly
- **WHEN** runtime traces equivalent requests under unchanged tracing config
- **THEN** canonical span names and required attribute keys remain replay-stable

### Requirement: Run and Stream SHALL preserve tracing semantic equivalence
For equivalent request context, effective tracing config, and dependency state, Run and Stream paths MUST produce semantically equivalent tracing outcomes.

Non-semantic emission ordering differences are allowed, but topology class and canonical attribute semantics MUST remain equivalent.

#### Scenario: Equivalent request via Run and Stream
- **WHEN** equivalent requests are executed through Run and Stream with tracing enabled
- **THEN** both paths emit semantically equivalent span topology class and canonical attributes

#### Scenario: Stream emits incremental events
- **WHEN** Stream emits additional incremental lifecycle events compared with Run
- **THEN** tracing outputs remain semantically equivalent after normalization

### Requirement: Agent eval interoperability SHALL expose minimal canonical metric contract
Runtime MUST expose a minimal interoperable agent evaluation metric contract that is backend-agnostic and replay-assertable.

Minimum metric surface MUST include:
- task-success outcome summary,
- tool-call correctness summary,
- deny/intercept correctness summary,
- cost-latency constraint summary.

#### Scenario: Local eval suite completes
- **WHEN** runtime executes eval suite in local mode
- **THEN** runtime emits canonical eval summary metrics using deterministic field semantics

#### Scenario: Eval metric mismatch is detected
- **WHEN** equivalent fixture replay observes metric divergence from canonical expectation
- **THEN** replay classifies mismatch deterministically as eval metric drift

### Requirement: Eval execution SHALL support embedded local and distributed modes with deterministic aggregation
Runtime MUST support `local|distributed` eval execution as embedded library behavior.

Distributed mode MUST support shard execution, retry, resume, and idempotent aggregation without requiring hosted control-plane dependency.

#### Scenario: Distributed shard retry and resume
- **WHEN** a shard fails transiently and runtime retries then resumes remaining shards
- **THEN** final eval aggregation remains deterministic and idempotent

#### Scenario: Control-plane dependency drift is introduced
- **WHEN** eval execution path introduces hosted scheduler or remote control-plane dependency
- **THEN** contract gate fails with deterministic control-plane-absence assertion

### Requirement: A61 same-domain extensions SHALL be absorbed as additive updates
Tracing and eval same-domain extensions (attributes, metrics, execution controls, replay classes, gate assertions) MUST be absorbed as additive updates under this capability contract and MUST NOT create parallel same-domain semantics.

#### Scenario: New trace attribute is added
- **WHEN** maintainers introduce additional OTel semantic attributes
- **THEN** change is expressed as additive contract update with replay and gate coverage under this capability

#### Scenario: New eval aggregation dimension is introduced
- **WHEN** maintainers add new eval aggregation dimension in distributed mode
- **THEN** change is absorbed as additive update in this capability without parallel proposal semantics

### Requirement: Tracing and eval outputs SHALL reuse canonical upstream explainability fields
A61 tracing and eval outputs MUST reuse canonical upstream fields from existing contracts when present:
- A58 policy explainability fields,
- A59 memory additive semantics,
- A60 budget-admission semantics.

Runtime MUST NOT introduce parallel same-meaning observability data-plane fields for these semantics.

#### Scenario: Eval summary references deny decision context
- **WHEN** eval summary is produced for requests involving deny or degrade decisions
- **THEN** output reuses canonical upstream policy and budget semantics without redefining same-meaning aliases

#### Scenario: Tracing includes memory and budget contributions
- **WHEN** tracing/export output includes memory and budget contribution context
- **THEN** output reuses canonical upstream memory and budget semantics and avoids duplicate same-meaning fields
