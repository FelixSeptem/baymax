# diagnostics-replay-tooling Specification

## Purpose
TBD - created by archiving change improve-dx-d1-api-reference-and-diagnostics-replay-e7. Update Purpose after archive.
## Requirements
### Requirement: Replay tooling SHALL accept diagnostics JSON as primary input
Diagnostics replay tooling MUST accept diagnostics JSON artifacts as input and MUST parse required timeline fields without requiring live runtime connectivity.

On malformed input, tooling MUST fail fast with deterministic machine-readable reason codes.

#### Scenario: Replay runs with valid diagnostics JSON
- **WHEN** tooling receives JSON payload containing supported diagnostics timeline schema
- **THEN** tooling produces replay output without requiring runtime API access

#### Scenario: Replay runs with malformed JSON
- **WHEN** tooling receives malformed JSON or missing required fields
- **THEN** tooling exits with deterministic validation reason code and no partial success status

### Requirement: Replay output SHALL support minimal timeline summary mode
Replay tooling MUST provide a minimal output mode that includes `phase`, `status`, `reason`, and `timestamp` fields, plus minimal correlation identifiers required for traceability.

#### Scenario: Minimal replay mode requested
- **WHEN** caller invokes replay in default minimal mode
- **THEN** output contains only required summary fields and deterministic ordering by replay sequence

#### Scenario: Missing optional details in source payload
- **WHEN** diagnostics source lacks optional extended fields
- **THEN** minimal replay output remains valid and omits unavailable optional fields without failure

### Requirement: Replay contract SHALL be regression-testable
The repository MUST provide contract tests for replay tooling using fixed sample inputs covering success and failure paths, and expected outputs/error codes MUST remain stable unless intentionally versioned.

#### Scenario: CI executes replay contract test suite
- **WHEN** standard test flow runs replay contract tests
- **THEN** expected normalized output snapshots and deterministic reason codes match version-controlled expectations

### Requirement: Replay tooling SHALL support readiness-timeout-health composite fixture mode
Diagnostics replay tooling MUST support composite fixture mode for readiness-timeout-health cross-domain validation.

Composite mode MUST:
- accept versioned fixture payload,
- emit normalized comparison output for canonical semantic fields,
- return deterministic error classification on fixture/schema mismatch.

#### Scenario: Composite fixture is replayed successfully
- **WHEN** tooling receives valid A47 composite fixture input
- **THEN** tooling emits deterministic normalized output with canonical semantic fields

#### Scenario: Composite fixture schema is invalid
- **WHEN** tooling receives malformed or unsupported fixture version
- **THEN** tooling fails fast with deterministic validation reason code

### Requirement: Replay tooling SHALL validate cross-domain primary-reason arbitration fixtures
Diagnostics replay tooling MUST support cross-domain primary-reason arbitration fixtures and MUST return deterministic drift classification on mismatch.

Drift classes MUST include at minimum:
- precedence drift
- tie-break drift
- taxonomy drift

#### Scenario: Replay fixture matches canonical arbitration output
- **WHEN** fixture expected arbitration output matches normalized actual output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay fixture detects precedence drift
- **WHEN** actual primary reason violates canonical precedence order
- **THEN** replay validation fails with deterministic precedence-drift classification

### Requirement: Replay tooling SHALL validate arbitration explainability fixtures
Diagnostics replay tooling MUST validate arbitration explainability fixtures, including secondary reason ordering, bounded count, remediation hint taxonomy, and rule-version stability.

Replay drift classes MUST include at minimum:
- `secondary_order_drift`
- `secondary_count_drift`
- `hint_taxonomy_drift`
- `rule_version_drift`

#### Scenario: Explainability fixture matches canonical output
- **WHEN** expected explainability fixture matches normalized replay output
- **THEN** replay validation passes deterministically

#### Scenario: Explainability fixture detects secondary-order drift
- **WHEN** replay output secondary reason ordering differs from canonical expectation
- **THEN** replay validation fails with deterministic `secondary_order_drift` classification

### Requirement: Replay tooling SHALL validate arbitration-version governance fixtures
Diagnostics replay tooling MUST support arbitration-version governance fixtures and MUST classify version-related semantic drift deterministically.

Drift classes MUST include at minimum:
- `version_mismatch`
- `unsupported_version`
- `cross_version_semantic_drift`

#### Scenario: Replay fixture matches expected version-governance output
- **WHEN** fixture expected requested/effective/source/policy output matches normalized actual output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay fixture detects unsupported-version drift
- **WHEN** actual output lacks expected unsupported-version classification
- **THEN** replay validation fails with deterministic `unsupported_version` drift classification

### Requirement: Replay tooling SHALL preserve backward-compatible fixture validation
Replay tooling MUST continue validating previously archived fixture schemas while adding version-governance fixture support.

#### Scenario: A47/A48 fixture validation runs with A50 tooling
- **WHEN** replay executes archived fixture suites and A50 fixture suites in one gate flow
- **THEN** archived fixture assertions remain valid and no cross-version parser regression is introduced

### Requirement: Replay tooling SHALL validate sandbox governance fixtures
Diagnostics replay tooling MUST support sandbox governance fixture validation with deterministic normalized output and drift classification.

Sandbox drift classes MUST include at minimum:
- `sandbox_policy_drift`
- `sandbox_fallback_drift`
- `sandbox_timeout_drift`
- `sandbox_capability_drift`
- `sandbox_resource_policy_drift`
- `sandbox_session_lifecycle_drift`

#### Scenario: Sandbox fixture matches canonical output
- **WHEN** replay tooling evaluates valid sandbox fixture and normalized output matches expected semantics
- **THEN** validation passes deterministically

#### Scenario: Sandbox fixture detects fallback drift
- **WHEN** replay output fallback behavior differs from canonical fixture expectation
- **THEN** validation fails with deterministic `sandbox_fallback_drift` classification

#### Scenario: Sandbox fixture detects capability drift
- **WHEN** replay output shows required capability satisfaction semantics diverging from canonical fixture
- **THEN** validation fails with deterministic `sandbox_capability_drift` classification

#### Scenario: Sandbox fixture detects session lifecycle drift
- **WHEN** replay output for per-call/per-session lifecycle semantics diverges from canonical fixture
- **THEN** validation fails with deterministic `sandbox_session_lifecycle_drift` classification

### Requirement: Replay tooling SHALL validate sandbox rollout-governance fixtures
Diagnostics replay tooling MUST support sandbox rollout-governance fixture validation using versioned fixture contract `a52.v1`.

Fixture validation MUST cover canonical fields:
- rollout phase
- health budget status
- capacity action
- freeze state and reason

#### Scenario: A52 rollout fixture matches canonical output
- **WHEN** replay tooling processes valid `a52.v1` fixture and actual output matches canonical expectation
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: A52 rollout fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `a52.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include rollout-governance drift classes
Replay tooling MUST classify rollout-governance semantic drift using canonical classes:
- `sandbox_rollout_phase_drift`
- `sandbox_health_budget_drift`
- `sandbox_capacity_action_drift`
- `sandbox_freeze_state_drift`

#### Scenario: Replay detects rollout phase drift
- **WHEN** actual rollout phase differs from expected fixture phase
- **THEN** replay validation fails with deterministic `sandbox_rollout_phase_drift` classification

#### Scenario: Replay detects capacity action drift
- **WHEN** actual capacity action differs from expected fixture action
- **THEN** replay validation fails with deterministic `sandbox_capacity_action_drift` classification

### Requirement: Replay tooling SHALL preserve backward compatibility for A51 fixtures
Adding A52 fixture support MUST NOT break existing A51 and earlier replay fixture validations.

#### Scenario: A51 and A52 fixtures run in single gate flow
- **WHEN** replay gate executes mixed fixture suites containing A51 and A52 fixture versions
- **THEN** both fixture generations are validated deterministically without parser regression

### Requirement: Replay tooling SHALL support memory fixture contract version memory v1
Diagnostics replay tooling MUST support versioned memory fixture contract `memory.v1`.

`memory.v1` fixture validation MUST cover at minimum:
- effective memory mode,
- provider and profile,
- operation counters,
- fallback classification,
- canonical reason codes.

#### Scenario: Replay validates canonical memory v1 fixture
- **WHEN** tooling replays valid `memory.v1` fixture with expected canonical output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay receives malformed memory fixture version
- **WHEN** tooling receives malformed or unsupported memory fixture schema
- **THEN** replay fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include canonical memory drift classes
Replay tooling MUST classify memory semantic drift using canonical classes:
- `memory_mode_drift`
- `memory_profile_drift`
- `memory_contract_version_drift`
- `memory_fallback_drift`
- `memory_error_taxonomy_drift`
- `memory_operation_aggregate_drift`

#### Scenario: Replay detects fallback behavior drift
- **WHEN** replay output fallback behavior differs from fixture expectation
- **THEN** replay validation fails with deterministic `memory_fallback_drift` classification

#### Scenario: Replay detects operation aggregate drift
- **WHEN** equivalent replay input produces non-equivalent memory operation aggregates
- **THEN** replay validation fails with deterministic `memory_operation_aggregate_drift` classification

### Requirement: Memory replay fixture support SHALL preserve backward-compatible mixed-fixture validation
Adding `memory.v1` support MUST NOT break validation of previously archived fixture versions.

#### Scenario: Mixed fixture suite includes A52 and memory v1 fixtures
- **WHEN** replay gate runs fixture suite containing historical fixtures and `memory.v1`
- **THEN** all fixture generations are parsed and validated deterministically without regression

#### Scenario: Historical fixture parser regression is introduced
- **WHEN** memory fixture support change breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge

### Requirement: Replay tooling SHALL validate observability export and bundle fixtures
Diagnostics replay tooling MUST support observability export and diagnostics bundle fixture validation using versioned fixture contract `observability.v1`.

Fixture validation MUST cover canonical fields at minimum:
- export profile and status,
- export degradation and failure reason taxonomy,
- bundle schema version and generation result,
- bundle redaction and gate-fingerprint metadata.

#### Scenario: Observability fixture matches canonical output
- **WHEN** replay tooling processes valid `observability.v1` fixture and actual output matches expected normalized semantics
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Observability fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `observability.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include observability and bundle drift classes
Replay tooling MUST classify observability and bundle semantic drift with canonical classes:
- `observability_export_profile_drift`
- `observability_export_status_drift`
- `observability_export_reason_drift`
- `diagnostics_bundle_schema_drift`
- `diagnostics_bundle_redaction_drift`
- `diagnostics_bundle_fingerprint_drift`

#### Scenario: Replay detects export status drift
- **WHEN** actual export status semantics differ from fixture expectation
- **THEN** replay validation fails with deterministic `observability_export_status_drift` classification

#### Scenario: Replay detects bundle redaction drift
- **WHEN** bundle output includes non-redacted secret-like fields compared with fixture expectation
- **THEN** replay validation fails with deterministic `diagnostics_bundle_redaction_drift` classification

### Requirement: Replay tooling SHALL preserve backward compatibility for pre-A55 fixtures
Adding `observability.v1` support MUST NOT break validation of existing fixture suites.

#### Scenario: Mixed fixture suites execute in one replay gate flow
- **WHEN** replay gate runs archived fixtures and `observability.v1` fixtures together
- **THEN** parser and validation remain backward compatible and deterministic for all suites

### Requirement: Replay tooling SHALL support ReAct fixture contract version react.v1
Diagnostics replay tooling MUST support versioned ReAct fixture contract `react.v1`.

`react.v1` fixture validation MUST cover at minimum:
- loop step sequence,
- iteration and tool-call counters,
- terminal reason classification,
- Stream dispatch parity markers,
- provider tool-calling normalization summary.

#### Scenario: Replay validates canonical react.v1 fixture
- **WHEN** tooling replays valid `react.v1` fixture with expected canonical output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay receives malformed react.v1 schema
- **WHEN** tooling receives malformed or unsupported `react.v1` fixture payload
- **THEN** replay fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include ReAct-specific drift classes
Replay tooling MUST classify ReAct semantic drift using canonical classes:
- `react_loop_step_drift`
- `react_tool_call_budget_drift`
- `react_iteration_budget_drift`
- `react_termination_reason_drift`
- `react_stream_dispatch_drift`
- `react_provider_mapping_drift`

#### Scenario: Replay detects terminal reason drift
- **WHEN** actual replay output termination reason differs from fixture expectation
- **THEN** replay validation fails with deterministic `react_termination_reason_drift` classification

#### Scenario: Replay detects stream dispatch parity drift
- **WHEN** replay output indicates Stream dispatch semantics diverge from canonical fixture expectation
- **THEN** replay validation fails with deterministic `react_stream_dispatch_drift` classification

### Requirement: ReAct fixture support SHALL preserve backward-compatible mixed-fixture validation
Adding `react.v1` support MUST NOT break existing fixture generations and mixed fixture replay flows.

#### Scenario: Mixed fixture suite includes A52 A53 memory v1 observability v1 and react.v1
- **WHEN** replay gate runs mixed fixture suite containing historical fixtures and `react.v1`
- **THEN** all fixture generations are parsed and validated deterministically without parser regression

#### Scenario: Historical fixture parser regression is introduced by react.v1 changes
- **WHEN** replay tooling update for `react.v1` breaks historical fixture parsing
- **THEN** replay validation fails and blocks merge

### Requirement: Replay tooling SHALL support sandbox egress fixture contract version sandbox_egress.v1
Diagnostics replay tooling MUST support versioned fixture contract `sandbox_egress.v1`.

Fixture validation MUST cover at minimum:
- egress action decision,
- egress policy source,
- violation classification,
- allowlist decision and primary code.

#### Scenario: Replay validates canonical sandbox_egress.v1 fixture
- **WHEN** tooling processes valid `sandbox_egress.v1` fixture and normalized output matches expectation
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay receives malformed sandbox_egress.v1 payload
- **WHEN** tooling receives malformed or unsupported `sandbox_egress.v1` fixture
- **THEN** replay fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include egress and allowlist drift classes
Replay tooling MUST classify A57 semantic drift using canonical classes:
- `sandbox_egress_action_drift`
- `sandbox_egress_policy_source_drift`
- `sandbox_egress_violation_taxonomy_drift`
- `adapter_allowlist_decision_drift`
- `adapter_allowlist_taxonomy_drift`

#### Scenario: Replay detects egress action drift
- **WHEN** replay output egress action differs from fixture expectation
- **THEN** validation fails with deterministic `sandbox_egress_action_drift` classification

#### Scenario: Replay detects allowlist taxonomy drift
- **WHEN** replay output allowlist reason taxonomy differs from fixture expectation
- **THEN** validation fails with deterministic `adapter_allowlist_taxonomy_drift` classification

### Requirement: A57 replay support SHALL preserve mixed-fixture backward compatibility
Adding `sandbox_egress.v1` support MUST NOT break validation of historical fixture versions.

#### Scenario: Mixed fixture suite includes A52 sandbox.v1 memory.v1 react.v1 and sandbox_egress.v1
- **WHEN** replay gate runs mixed fixture suite across multiple versions
- **THEN** all fixtures are validated deterministically without parser regression

#### Scenario: A57 fixture support breaks historical parser behavior
- **WHEN** tooling update for A57 introduces parser regression for archived fixtures
- **THEN** replay validation fails and blocks merge

### Requirement: Replay tooling SHALL validate policy precedence fixtures
Diagnostics replay tooling MUST support policy precedence fixture validation using versioned fixture contract `policy_stack.v1`.

Fixture validation MUST cover at minimum:
- winner stage
- deny source
- decision path
- tie-break reason

#### Scenario: Policy precedence fixture matches canonical output
- **WHEN** replay tooling processes valid `policy_stack.v1` fixture and normalized output matches expected semantics
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Policy precedence fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `policy_stack.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include canonical policy-stack drift classes
Replay tooling MUST classify policy-stack semantic drift using canonical classes:
- `precedence_conflict`
- `tie_break_drift`
- `deny_source_mismatch`

#### Scenario: Replay detects precedence conflict drift
- **WHEN** actual winner stage violates expected precedence matrix
- **THEN** replay validation fails with deterministic `precedence_conflict` classification

#### Scenario: Replay detects deny source mismatch
- **WHEN** actual deny source differs from expected canonical source
- **THEN** replay validation fails with deterministic `deny_source_mismatch` classification

### Requirement: Policy fixture support SHALL preserve mixed-fixture backward compatibility
Adding `policy_stack.v1` support MUST NOT break existing fixture validations.

#### Scenario: Mixed fixture suites run in one gate flow
- **WHEN** replay gate executes `a50.v1`、`react.v1`、`sandbox_egress.v1` 与 `policy_stack.v1`
- **THEN** all fixture generations are validated deterministically without parser regression

#### Scenario: Historical fixture parser regression is introduced
- **WHEN** policy fixture support change breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge

### Requirement: Replay tooling SHALL validate memory governance fixtures
Diagnostics replay tooling MUST support memory governance fixture contracts:
- `memory_scope.v1`
- `memory_search.v1`
- `memory_lifecycle.v1`

Fixture validation MUST cover canonical fields for scope resolution, budget usage, search/rerank aggregates, and lifecycle action summaries.

#### Scenario: Memory governance fixtures match canonical output
- **WHEN** replay tooling processes valid memory governance fixtures and normalized output matches expected semantics
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Memory governance fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported memory governance fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include memory governance drift classes
Replay tooling MUST classify memory governance semantic drift using canonical classes:
- `scope_resolution_drift`
- `retrieval_quality_regression`
- `lifecycle_policy_drift`
- `recovery_consistency_drift`

#### Scenario: Replay detects retrieval quality regression
- **WHEN** replay output top-k/rerank metrics diverge from fixture expectation
- **THEN** replay validation fails with deterministic `retrieval_quality_regression` classification

#### Scenario: Replay detects lifecycle policy drift
- **WHEN** replay output lifecycle action differs from configured fixture policy
- **THEN** replay validation fails with deterministic `lifecycle_policy_drift` classification

### Requirement: Memory governance fixtures SHALL preserve mixed-fixture backward compatibility
Adding memory governance fixture support MUST NOT break validation for archived fixture suites.

#### Scenario: Mixed fixture suites execute in one gate flow
- **WHEN** replay gate runs historical fixtures and memory governance fixtures together
- **THEN** parser and validation remain backward compatible and deterministic for all suites

#### Scenario: Legacy fixture parser regression is introduced
- **WHEN** memory governance fixture support breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge

### Requirement: Replay tooling SHALL validate budget-admission fixtures
Diagnostics replay tooling MUST support budget-admission fixture validation with versioned fixture contract `budget_admission.v1`.

Fixture validation MUST cover at minimum:
- budget snapshot thresholds
- budget decision
- degrade action

#### Scenario: Budget-admission fixture matches canonical output
- **WHEN** replay tooling processes valid `budget_admission.v1` fixture and normalized output matches expected semantics
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Budget-admission fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `budget_admission.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include canonical budget-admission drift classes
Replay tooling MUST classify budget-admission semantic drift using canonical classes:
- `budget_threshold_drift`
- `admission_decision_drift`
- `degrade_policy_drift`

#### Scenario: Replay detects budget threshold drift
- **WHEN** actual threshold evaluation output differs from expected fixture threshold semantics
- **THEN** replay validation fails with deterministic `budget_threshold_drift` classification

#### Scenario: Replay detects degrade policy drift
- **WHEN** actual degrade action selection differs from fixture policy expectation
- **THEN** replay validation fails with deterministic `degrade_policy_drift` classification

### Requirement: Budget fixture support SHALL preserve mixed-fixture backward compatibility
Adding `budget_admission.v1` support MUST NOT break existing archived fixture validations.

#### Scenario: Mixed fixture suites run in one gate flow
- **WHEN** replay gate executes historical fixtures together with `budget_admission.v1`
- **THEN** all fixture generations are parsed and validated deterministically without regression

#### Scenario: Historical parser regression is introduced
- **WHEN** budget fixture support breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge

### Requirement: Replay tooling SHALL validate OTel semantic-convention fixtures
Diagnostics replay tooling MUST support OTel tracing semantic-convention fixture validation using versioned fixture contract `otel_semconv.v1`.

Fixture validation MUST cover at minimum:
- canonical attribute mapping,
- span topology class.

#### Scenario: OTel semconv fixture matches canonical output
- **WHEN** replay tooling processes valid `otel_semconv.v1` fixture and normalized output matches expectation
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: OTel semconv fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `otel_semconv.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include canonical OTel drift classes
Replay tooling MUST classify OTel semantic drift using canonical classes:
- `otel_attr_mapping_drift`
- `span_topology_drift`

#### Scenario: Replay detects OTel attribute mapping drift
- **WHEN** actual attribute mapping differs from canonical fixture expectation
- **THEN** replay validation fails with deterministic `otel_attr_mapping_drift` classification

#### Scenario: Replay detects span topology drift
- **WHEN** actual span topology class differs from canonical fixture expectation
- **THEN** replay validation fails with deterministic `span_topology_drift` classification

### Requirement: Replay tooling SHALL validate agent eval fixtures
Diagnostics replay tooling MUST support agent eval fixture validation using:
- `agent_eval.v1`
- `agent_eval_distributed.v1`

Fixture validation MUST cover at minimum:
- eval metric summary semantics,
- execution mode semantics,
- distributed shard aggregation semantics.

#### Scenario: Agent eval fixture matches canonical output
- **WHEN** replay tooling processes valid `agent_eval.v1` fixture and normalized output matches expectation
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Distributed eval fixture matches canonical output
- **WHEN** replay tooling processes valid `agent_eval_distributed.v1` fixture and normalized output matches expectation
- **THEN** replay validation succeeds with deterministic pass result

### Requirement: Replay drift classification SHALL include canonical eval drift classes
Replay tooling MUST classify eval semantic drift using canonical classes:
- `eval_metric_drift`
- `eval_aggregation_drift`
- `eval_shard_resume_drift`

#### Scenario: Replay detects eval metric drift
- **WHEN** actual eval metrics diverge from fixture expectation
- **THEN** replay validation fails with deterministic `eval_metric_drift` classification

#### Scenario: Replay detects shard resume drift
- **WHEN** distributed eval resume/aggregation behavior diverges from fixture expectation
- **THEN** replay validation fails with deterministic `eval_shard_resume_drift` classification

### Requirement: OTel and eval fixture support SHALL preserve mixed-fixture backward compatibility
Adding `otel_semconv.v1`, `agent_eval.v1`, and `agent_eval_distributed.v1` support MUST NOT break validation for archived fixture suites.

#### Scenario: Mixed fixture suites run in one gate flow
- **WHEN** replay gate executes historical fixtures together with A61 fixtures
- **THEN** all fixture generations are parsed and validated deterministically without regression

#### Scenario: Historical parser regression is introduced
- **WHEN** A61 fixture support breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge

### Requirement: A65 Replay Fixture Coverage
Diagnostics replay tooling MUST support A65 fixtures `hooks_middleware.v1`, `skill_discovery_sources.v1`, and `skill_preprocess_and_mapping.v1`.

#### Scenario: Fixture parsing compatibility
- **WHEN** replay runner loads A65 fixtures together with historical fixtures
- **THEN** parser MUST accept mixed versions and preserve deterministic normalized output

#### Scenario: Fixture schema validation
- **WHEN** required A65 fixture fields are missing or invalid
- **THEN** replay tooling MUST fail fast with deterministic schema mismatch classification

### Requirement: A65 Drift Classification
Replay tooling MUST classify hook/middleware/discovery/mapping drifts with canonical error taxonomy.

#### Scenario: Hook order drift classification
- **WHEN** hook execution order deviates from canonical sequence
- **THEN** replay MUST classify drift as `hooks_order_drift`

#### Scenario: Discovery source drift classification
- **WHEN** discovery source merge or dedup result deviates under identical input
- **THEN** replay MUST classify drift as `skill_discovery_source_drift`

#### Scenario: Bundle mapping drift classification
- **WHEN** prompt augmentation or whitelist mapping output deviates from configured policy
- **THEN** replay MUST classify drift as `skill_bundle_mapping_drift`

### Requirement: State Session Snapshot Replay Fixture Support
Diagnostics replay tooling MUST support `state_session_snapshot.v1` fixture schema with deterministic normalization and mixed-version compatibility.

#### Scenario: Replay parses v1 fixture deterministically
- **WHEN** replay executes against valid `state_session_snapshot.v1` fixture input
- **THEN** normalized output MUST be deterministic across repeated executions

#### Scenario: Mixed fixture compatibility
- **WHEN** replay executes with historical fixtures and `state_session_snapshot.v1` together
- **THEN** parser MUST preserve backward compatibility and reject only true schema violations

### Requirement: Snapshot Drift Classification
Replay tooling MUST classify snapshot drifts using canonical taxonomy for schema, compatibility, restore semantics, and partial restore behavior.

#### Scenario: Schema drift classification
- **WHEN** required snapshot manifest fields drift from expected schema
- **THEN** replay MUST classify failure as `snapshot_schema_drift`

#### Scenario: Restore semantic drift classification
- **WHEN** restore action/conflict outcome differs under equivalent fixture input
- **THEN** replay MUST classify failure as `state_restore_semantic_drift`

#### Scenario: Compatibility window drift classification
- **WHEN** compatible/strict acceptance behavior differs for same version inputs
- **THEN** replay MUST classify failure as `snapshot_compat_window_drift`

