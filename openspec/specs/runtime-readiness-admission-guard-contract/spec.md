# runtime-readiness-admission-guard-contract Specification

## Purpose
TBD - created by archiving change introduce-runtime-readiness-admission-guard-and-degradation-policy-contract-a44. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL provide readiness-admission guard before managed execution
Runtime MUST provide a readiness-admission guard that evaluates readiness status before managed Run/Stream execution begins.

Admission evaluation MUST consume readiness preflight result and MUST produce deterministic decision:
- `allow`
- `deny`

#### Scenario: Admission guard evaluates blocked readiness
- **WHEN** readiness preflight result is `blocked` and admission feature is enabled
- **THEN** admission decision is `deny` and managed execution does not start

#### Scenario: Admission guard evaluates ready readiness
- **WHEN** readiness preflight result is `ready` and admission feature is enabled
- **THEN** admission decision is `allow` and managed execution can proceed

### Requirement: Readiness-admission deny path SHALL be side-effect free
When admission decision is `deny`, runtime MUST fail fast before any scheduler enqueue, mailbox publish, or task lifecycle mutation.

#### Scenario: Deny path rejects run without task mutation
- **WHEN** admission guard returns `deny` for a managed run request
- **THEN** runtime returns deterministic admission error and scheduler/mailbox state remains unchanged

#### Scenario: Equivalent deny path under Run and Stream
- **WHEN** equivalent requests under Run and Stream both hit admission deny
- **THEN** both paths return semantically equivalent admission classification with no lifecycle side effects

### Requirement: Degraded readiness SHALL support policy-controlled admission
Runtime MUST support policy-controlled handling for `degraded` readiness:
- `allow_and_record`
- `fail_fast`

#### Scenario: Degraded readiness with allow-and-record policy
- **WHEN** readiness result is `degraded` and degraded policy is `allow_and_record`
- **THEN** admission allows execution and records degraded-admission observability markers

#### Scenario: Degraded readiness with fail-fast policy
- **WHEN** readiness result is `degraded` and degraded policy is `fail_fast`
- **THEN** admission denies execution with deterministic degraded-admission reason classification

### Requirement: Admission guard SHALL consume arbitration-aligned primary reason without reclassification drift
Runtime readiness admission guard MUST consume primary reason output from cross-domain arbitration without introducing per-path reclassification drift.

Admission decision explanation fields MUST preserve:
- primary domain,
- primary code,
- primary source.

#### Scenario: Admission deny consumes blocked primary reason
- **WHEN** admission guard receives blocked-class primary reason from arbitration
- **THEN** deny decision explanation preserves the same primary domain/code/source semantics

#### Scenario: Admission allow-and-record consumes degraded primary reason
- **WHEN** admission guard receives degraded-class primary reason under allow-and-record policy
- **THEN** allow decision explanation preserves arbitration primary reason without remapping

### Requirement: Admission guard SHALL preserve arbitration explainability semantics
Admission guard decisions MUST preserve arbitration explainability semantics without per-path remapping drift.

Admission explanation MUST include:
- primary reason fields,
- bounded secondary reason fields,
- remediation hint fields.

#### Scenario: Admission deny path includes explainability output
- **WHEN** admission guard denies execution using arbitration result
- **THEN** deny explanation preserves canonical primary and secondary explainability fields

#### Scenario: Admission allow-and-record path includes explainability output
- **WHEN** admission guard allows degraded execution with record policy
- **THEN** allow explanation preserves canonical explainability fields without reclassification drift

### Requirement: Admission guard SHALL enforce arbitration-version governance policy before execution
Runtime readiness-admission guard MUST evaluate arbitration-version governance outcomes before managed execution begins.

When version policy requires fail-fast (`on_unsupported=fail_fast` or `on_mismatch=fail_fast`), admission guard MUST deny execution deterministically.

#### Scenario: Admission guard denies unsupported-version request
- **WHEN** admission evaluates readiness result containing unsupported arbitration version finding under fail-fast policy
- **THEN** admission decision is `deny` and execution does not start

#### Scenario: Admission guard denies compatibility-mismatch request
- **WHEN** admission evaluates readiness result containing compatibility-mismatch finding under fail-fast policy
- **THEN** admission decision is `deny` with deterministic reason classification

### Requirement: Admission decision SHALL preserve arbitration-version explainability fields
Admission explanation fields MUST preserve arbitration-version explainability metadata without per-path remap:
- requested version,
- effective version,
- version source,
- policy action.

#### Scenario: Admission allow path preserves version metadata
- **WHEN** readiness result passes version-governance checks and admission decision is `allow`
- **THEN** admission explanation exposes deterministic arbitration-version metadata

#### Scenario: Admission deny path preserves version metadata
- **WHEN** readiness result fails version-governance checks and admission decision is `deny`
- **THEN** admission explanation exposes same canonical version metadata used by readiness/arbitration outputs

### Requirement: Admission guard SHALL deny managed execution when required sandbox dependency is unavailable
Managed Run/Stream admission MUST deny execution when readiness reports required sandbox dependency as unavailable or invalid.

The deny path MUST remain side-effect free and preserve deterministic admission error classification.

#### Scenario: Managed run denied by required sandbox-unavailable finding
- **WHEN** admission guard receives blocked readiness primary reason indicating required sandbox dependency unavailable
- **THEN** admission decision is `deny` and scheduler/mailbox/task lifecycle state remains unchanged

#### Scenario: Managed stream denied by required sandbox-profile-invalid finding
- **WHEN** admission guard receives blocked readiness primary reason indicating sandbox profile invalid
- **THEN** admission decision is `deny` with semantically equivalent classification and no lifecycle mutation

#### Scenario: Managed run denied by sandbox capability mismatch
- **WHEN** admission guard receives blocked readiness primary reason indicating sandbox capability mismatch
- **THEN** admission decision is `deny` with deterministic capability-mismatch classification and no lifecycle mutation

### Requirement: Admission explainability SHALL preserve sandbox-related arbitration fields
Admission outputs for sandbox-driven deny/allow decisions MUST preserve canonical arbitration explainability fields without remapping drift.

#### Scenario: Sandbox-driven deny includes explainability payload
- **WHEN** admission denies execution due to sandbox-required readiness finding
- **THEN** output includes canonical primary reason and bounded explainability fields

#### Scenario: Equivalent sandbox-driven decisions in Run and Stream
- **WHEN** equivalent managed Run and Stream requests hit same sandbox admission outcome
- **THEN** outputs preserve semantically equivalent explainability and reason taxonomy

### Requirement: Admission guard SHALL enforce sandbox rollout freeze semantics before execution
Runtime readiness-admission guard MUST deny managed execution when rollout phase is `frozen` under enforce policy.

Deny path MUST remain side-effect free (no scheduler enqueue, mailbox publish, or lifecycle mutation).

#### Scenario: Admission evaluates frozen rollout under enforce policy
- **WHEN** admission receives readiness output indicating `sandbox.rollout.frozen`
- **THEN** admission decision is `deny` with deterministic freeze reason classification

#### Scenario: Frozen deny path remains side-effect free
- **WHEN** managed Run/Stream request is denied because rollout is frozen
- **THEN** runtime does not mutate scheduler/mailbox/task lifecycle state

### Requirement: Admission guard SHALL apply sandbox capacity action deterministically
Admission guard MUST consume canonical capacity action (`allow|throttle|deny`) and apply deterministic policy mapping.

For `throttle`, admission MUST follow configured degraded policy:
- `allow_and_record`
- `fail_fast`

#### Scenario: Capacity action is throttle with allow-and-record policy
- **WHEN** readiness reports capacity action `throttle` and degraded policy is `allow_and_record`
- **THEN** admission allows execution and records canonical throttle observability markers

#### Scenario: Capacity action is throttle with fail-fast policy
- **WHEN** readiness reports capacity action `throttle` and degraded policy is `fail_fast`
- **THEN** admission denies execution with deterministic throttle classification

#### Scenario: Capacity action is deny
- **WHEN** readiness reports capacity action `deny`
- **THEN** admission denies execution deterministically and preserves side-effect-free deny semantics

### Requirement: Run and Stream SHALL preserve rollout-capacity admission equivalence
For equivalent inputs and effective configuration, Run and Stream admission decisions MUST remain semantically equivalent for frozen and capacity-governed paths.

#### Scenario: Equivalent Run and Stream requests under throttle policy
- **WHEN** equivalent requests are evaluated with capacity action `throttle`
- **THEN** both paths produce semantically equivalent allow-or-deny admission outcomes according to degraded policy

#### Scenario: Equivalent Run and Stream requests under frozen rollout
- **WHEN** equivalent requests are evaluated with rollout phase `frozen`
- **THEN** both paths return semantically equivalent deny classification

### Requirement: Admission guard SHALL enforce ReAct readiness findings before managed execution
Runtime readiness-admission guard MUST evaluate ReAct-related blocking findings before managed Run/Stream execution starts.

When ReAct blocking finding is present, admission decision MUST be `deny` and deny path MUST remain side-effect free.

#### Scenario: Managed Run is denied by ReAct blocking finding
- **WHEN** admission evaluates readiness output containing blocking `react.provider_tool_calling_unsupported`
- **THEN** admission decision is `deny` and runtime performs no scheduler enqueue, mailbox publish, or lifecycle mutation

#### Scenario: Managed Stream is denied by ReAct blocking finding
- **WHEN** admission evaluates readiness output containing blocking `react.stream_dispatch_unavailable`
- **THEN** admission decision is `deny` with semantically equivalent side-effect-free semantics

### Requirement: Degraded ReAct readiness SHALL follow policy-controlled admission mapping
When ReAct readiness status is `degraded`, admission guard MUST apply configured degraded policy:
- `allow_and_record`
- `fail_fast`

#### Scenario: Degraded ReAct finding with allow-and-record policy
- **WHEN** ReAct readiness is degraded and policy is `allow_and_record`
- **THEN** admission decision is `allow` and runtime records canonical degraded-admission markers

#### Scenario: Degraded ReAct finding with fail-fast policy
- **WHEN** ReAct readiness is degraded and policy is `fail_fast`
- **THEN** admission decision is `deny` with deterministic degraded-class reason

### Requirement: Run and Stream admission SHALL preserve ReAct reason taxonomy equivalence
For equivalent readiness input and policy, Run and Stream admission decisions MUST preserve equivalent ReAct reason taxonomy and explainability fields.

#### Scenario: Equivalent degraded ReAct admission under Run and Stream
- **WHEN** equivalent Run and Stream requests receive the same degraded ReAct readiness output
- **THEN** both paths produce semantically equivalent admission decision and reason taxonomy

#### Scenario: Equivalent blocked ReAct admission under Run and Stream
- **WHEN** equivalent Run and Stream requests receive the same blocked ReAct readiness output
- **THEN** both paths produce semantically equivalent deny classification and explainability output

### Requirement: Admission guard SHALL deny execution on blocking egress findings
Runtime readiness-admission guard MUST deny managed execution when readiness reports blocking sandbox egress findings.

Deny path MUST remain side-effect free.

#### Scenario: Admission receives blocking egress finding
- **WHEN** readiness output contains blocking `sandbox.egress.policy_invalid`
- **THEN** admission decision is `deny` and runtime performs no scheduler or mailbox side effects

#### Scenario: Equivalent Run and Stream egress deny
- **WHEN** equivalent managed Run and Stream requests consume the same blocking egress finding
- **THEN** both paths return semantically equivalent deny classification

### Requirement: Admission guard SHALL deny activation on blocking allowlist findings
Admission MUST deny execution when readiness reports blocking adapter allowlist findings for required runtime adapters.

#### Scenario: Required adapter missing allowlist entry
- **WHEN** readiness output includes blocking `adapter.allowlist.missing_entry`
- **THEN** admission decision is `deny` with deterministic allowlist reason taxonomy

#### Scenario: Signature invalid under enforce mode
- **WHEN** readiness output includes blocking `adapter.allowlist.signature_invalid`
- **THEN** admission decision is `deny` and managed execution does not start

### Requirement: Admission explainability SHALL preserve egress and allowlist primary reason fields
Admission outputs MUST preserve canonical arbitration explainability fields when deny is driven by egress or allowlist findings.

#### Scenario: Egress-driven deny includes explainability payload
- **WHEN** admission denies due to egress blocking finding
- **THEN** output includes canonical primary domain code source and bounded secondary reasons

#### Scenario: Allowlist-driven deny includes explainability payload
- **WHEN** admission denies due to allowlist blocking finding
- **THEN** output includes canonical allowlist primary reason fields without remapping drift

### Requirement: Admission guard SHALL consume canonical policy precedence output
Runtime readiness-admission guard MUST consume policy precedence winner output and apply deterministic deny/allow mapping without entrypoint-specific drift.

#### Scenario: Admission receives policy winner from sandbox egress stage
- **WHEN** policy winner stage is `sandbox_egress`
- **THEN** admission returns deterministic deny classification and preserves stage/source semantics

#### Scenario: Equivalent Run and Stream admission mapping
- **WHEN** equivalent Run and Stream requests consume the same policy winner output
- **THEN** both paths return semantically equivalent admission decision and reason taxonomy

### Requirement: Admission deny path SHALL remain side-effect-free under policy precedence
When policy precedence yields deny, admission MUST reject execution before scheduler/mailbox/tool dispatch side effects.

#### Scenario: Policy winner is blocking before execution
- **WHEN** admission receives blocking winner stage from policy evaluator
- **THEN** runtime denies request and does not emit runtime execution side effects

#### Scenario: Policy winner changes after hot reload rollback
- **WHEN** invalid hot reload is rejected and previous snapshot is restored
- **THEN** admission uses restored policy winner semantics deterministically

### Requirement: Admission explainability SHALL preserve decision trace fields
Admission response MUST preserve canonical decision-trace fields for policy precedence winners.

#### Scenario: Deny response includes decision trace
- **WHEN** admission denies due to policy precedence
- **THEN** response includes canonical `deny_source` and `winner_stage`

#### Scenario: Tie-break deny includes tie-break reason
- **WHEN** deny winner is selected via same-stage tie-break
- **THEN** response includes canonical `tie_break_reason` without remapping drift

