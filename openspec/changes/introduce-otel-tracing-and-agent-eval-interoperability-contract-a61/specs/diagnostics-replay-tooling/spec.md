## ADDED Requirements

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
