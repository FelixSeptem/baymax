# jit-context-organization-and-reference-first-assembly-contract Specification

## Purpose
TBD - created by archiving change introduce-jit-context-organization-and-reference-first-assembly-contract-a67-ctx. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL Provide Reference-First Two-Phase Context Injection
Runtime MUST provide a two-phase JIT context assembly flow for ReAct paths:
- `discover_refs`
- `resolve_selected_refs`

Reference metadata MUST be injected before full-body expansion, and full-body resolution MUST be bounded by configured budget.

#### Scenario: Context injection resolves selected references first
- **WHEN** runtime assembles stage2 context for a ReAct step
- **THEN** runtime MUST first emit/select canonical references and only then resolve full content for selected references

#### Scenario: Missing reference resolution follows configured policy
- **WHEN** selected reference cannot be resolved by locator/path/id
- **THEN** runtime MUST apply deterministic configured missing-reference policy and record outcome

### Requirement: Runtime SHALL Enforce Isolate Handoff Contract
Sub-agent handoff payload MUST use canonical structure:
- `summary`
- `artifacts[]`
- `evidence_refs[]`
- `confidence`
- `ttl`

Main-agent ingestion MUST default to summary-plus-reference mode and MUST NOT inline full sub-agent payload by default.

#### Scenario: Main agent consumes structured isolate handoff
- **WHEN** sub-agent returns canonical handoff payload
- **THEN** main agent MUST consume summary and evidence references as default input for subsequent reasoning

#### Scenario: Expired isolate handoff is rejected
- **WHEN** handoff payload exceeds configured `ttl`
- **THEN** runtime MUST reject or downgrade ingestion deterministically and record canonical reason

### Requirement: Runtime SHALL Gate Context Editing by Clear-At-Least Benefit
Runtime MUST gate aggressive context editing behind clear-at-least benefit checks using configured thresholds.

Edit action MUST execute only when estimated token savings and stability-benefit ratio satisfy configured minimums.

#### Scenario: Benefit threshold met triggers edit action
- **WHEN** estimated saved tokens and benefit ratio are above configured thresholds
- **THEN** runtime MUST permit edit action and record gate decision as allowed

#### Scenario: Benefit threshold not met blocks edit action
- **WHEN** estimated saved tokens or benefit ratio are below configured thresholds
- **THEN** runtime MUST skip aggressive edit action and preserve equivalent context semantics

### Requirement: Runtime SHALL Provide Relevance-Aware Swap-Back and Lifecycle Tiering
Runtime MUST support relevance-aware swap-back and canonical context lifecycle tiers:
- `hot`
- `warm`
- `cold`

Swap-back MUST be based on current query plus evidence tags, not only run-level completion.

#### Scenario: Relevant context is swapped back for active query
- **WHEN** current query relevance score for spilled context is above configured threshold
- **THEN** runtime MUST re-inject relevant context using canonical swap-back pathway

#### Scenario: Low relevance context remains externalized
- **WHEN** relevance score is below configured threshold
- **THEN** runtime MUST keep context externalized and avoid non-canonical reinjection

### Requirement: Runtime SHALL Emit Task-Aware Recap
Runtime MUST emit structured recap grounded in actual context actions taken in the current step, including selection/edit/externalization decisions.

#### Scenario: Recap reflects actual context decisions
- **WHEN** runtime completes a reasoning step with context operations
- **THEN** recap MUST include deterministic source markers describing actual performed operations

#### Scenario: Recap avoids fixed-template fallback drift
- **WHEN** context decisions differ across equivalent workloads
- **THEN** recap semantics MUST remain action-grounded and MUST NOT degrade into unrelated static template output

### Requirement: JIT Context Contract MUST Preserve Existing Runtime Boundaries
JIT context organization MUST remain within existing runtime boundaries and MUST NOT:
- introduce parallel ReAct termination taxonomy,
- bypass security governance chain,
- or directly couple `context/*` packages to provider official SDKs.

#### Scenario: JIT context enabled with canonical ReAct loop
- **WHEN** JIT context organization is enabled
- **THEN** runtime MUST preserve canonical A56 termination taxonomy and A58 decision-trace semantics

#### Scenario: Boundary checks detect direct provider SDK coupling from context packages
- **WHEN** `context/*` introduces direct import of provider official SDK packages
- **THEN** boundary validation MUST fail with deterministic violation classification

