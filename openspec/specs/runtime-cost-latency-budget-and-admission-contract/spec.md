# runtime-cost-latency-budget-and-admission-contract Specification

## Purpose
TBD - created by archiving change introduce-runtime-cost-latency-budget-and-admission-contract-a60. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL build a unified budget snapshot before admission
Runtime MUST build a unified `budget_snapshot` before managed admission decision.

The snapshot MUST include at minimum:
- cost estimate for `token|tool|sandbox|memory`
- latency estimate for `token|tool|sandbox|memory`
- budget version and evaluation timestamp

Equivalent inputs and effective configuration MUST produce semantically equivalent snapshot outputs.

#### Scenario: Admission evaluates request with mixed resource usage
- **WHEN** managed request includes model inference, tool dispatch, sandbox checks, and memory retrieval
- **THEN** runtime produces one unified `budget_snapshot` that includes all four resource domains

#### Scenario: Equivalent request is evaluated repeatedly
- **WHEN** runtime evaluates equivalent input under unchanged budget config
- **THEN** `budget_snapshot` semantics remain deterministic and replay-stable

### Requirement: Admission SHALL apply deterministic budget decision mapping
Runtime MUST map unified budget evaluation to canonical decision:
- `allow`
- `degrade`
- `deny`

Decision mapping MUST be deterministic under equivalent inputs and MUST preserve side-effect-free deny semantics.

#### Scenario: Budget is within thresholds
- **WHEN** cost and latency estimates are within configured budget thresholds
- **THEN** admission decision is `allow`

#### Scenario: Budget exceeds hard threshold
- **WHEN** budget estimate exceeds configured hard cost or latency threshold
- **THEN** admission decision is `deny` and runtime performs no scheduler/mailbox/task side effects

### Requirement: Degrade policy SHALL be configurable and observable
When admission decision is `degrade`, runtime MUST apply configured degrade policy and emit canonical `degrade_action`.

Degrade action selection MUST be deterministic and MUST be machine-assertable in diagnostics and replay.

#### Scenario: Degrade policy applies deterministic action
- **WHEN** budget evaluation enters degrade range and degrade policy is enabled
- **THEN** runtime selects deterministic `degrade_action` according to configured policy order

#### Scenario: Degrade policy is invalid
- **WHEN** degrade policy configuration is malformed or unsupported
- **THEN** startup or hot reload fails fast and active snapshot remains unchanged

### Requirement: Run and Stream SHALL preserve budget-admission semantic equivalence
For equivalent request context, budget config, and dependency state, Run and Stream MUST produce semantically equivalent budget snapshot, decision, and degrade action.

#### Scenario: Equivalent Run and Stream in degrade range
- **WHEN** equivalent managed Run and Stream requests are evaluated in degrade range
- **THEN** both paths emit semantically equivalent `budget_decision=degrade` and `degrade_action`

#### Scenario: Equivalent Run and Stream in deny range
- **WHEN** equivalent managed Run and Stream requests are evaluated above hard threshold
- **THEN** both paths emit semantically equivalent `budget_decision=deny` and remain side-effect free

### Requirement: Budget admission SHALL remain library-embedded without control-plane dependencies
Budget admission implementation MUST remain library-embedded and MUST NOT require hosted control-plane services, remote admission schedulers, or platform-managed policy centers.

This boundary MUST be machine-assertable by contract gate.

#### Scenario: Budget contract gate validates control-plane absence
- **WHEN** budget-admission contract gate is executed
- **THEN** validation includes `budget_control_plane_absent` assertion and fails on control-plane dependency drift

### Requirement: Budget admission explainability SHALL reuse canonical upstream fields
Budget admission outputs MUST reuse canonical upstream fields from existing contracts when present:
- A58 policy explainability (`winner_stage`, `deny_source`, `policy_decision_path`)
- A59 memory additive semantics (memory budget/source indicators)

Runtime MUST NOT introduce parallel same-meaning fields for these semantics.

#### Scenario: Budget decision references policy winner
- **WHEN** budget admission is evaluated for a request with policy winner output
- **THEN** budget explainability reuses canonical A58 fields without redefining equivalent aliases

#### Scenario: Budget decision references memory contribution
- **WHEN** budget snapshot includes memory-domain contribution
- **THEN** output reuses canonical A59 semantics and does not emit duplicate same-meaning fields

