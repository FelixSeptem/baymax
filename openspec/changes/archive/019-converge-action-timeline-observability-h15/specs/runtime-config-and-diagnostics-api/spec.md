## MODIFIED Requirements

### Requirement: Runtime SHALL expose diagnostics through library API only
The runtime MUST provide diagnostics query APIs for recent run summaries, recent MCP call summaries, and sanitized effective configuration, and MUST NOT require CLI support. Diagnostics returned by these APIs MUST follow single-writer and idempotent persistence semantics so repeated event submission does not alter logical aggregate counts.

Diagnostics MUST include capability-preflight and provider-fallback summary fields for each affected model step, including requested capability set, candidate providers considered, selected provider, and fail-fast reason when chain is exhausted.

Diagnostics MUST additionally include context assembler CA1 baseline fields for each assemble cycle and related run summary context, including `prefix_hash`, `assemble_latency_ms`, `assemble_status`, and `guard_violation`.

Diagnostics MUST additionally include context assembler CA2 stage and recap fields, including normalized stage statuses, stage2 skip reason, stage latencies, and recap status.

Diagnostics and event payloads MUST additionally apply unified S1 redaction policy before persistence and emission.

Diagnostics for CA2 retrieval MUST additionally expose normalized Stage2 retrieval summary fields: `stage2_hit_count`, `stage2_source`, `stage2_reason`, `stage2_reason_code`, `stage2_error_layer`, and `stage2_profile`.

Diagnostics for Action Timeline H1.5 MUST additionally expose run-level phase aggregates with minimum fields per phase: `count_total`, `failed_total`, `canceled_total`, `skipped_total`, `latency_ms`, and `latency_p95_ms`.

#### Scenario: Consumer requests recent run diagnostics
- **WHEN** application calls diagnostics API for recent runs
- **THEN** runtime returns bounded summary records with normalized fields and without duplicated logical run entries caused by retries or replay

#### Scenario: Consumer requests effective configuration
- **WHEN** application calls API to fetch effective configuration
- **THEN** runtime returns a sanitized snapshot that masks secret-like fields

#### Scenario: Consumer inspects fallback diagnostics
- **WHEN** application queries diagnostics for a run that triggered provider fallback
- **THEN** runtime returns normalized fallback summary fields sufficient to reconstruct capability decision path

#### Scenario: Consumer inspects context assembler diagnostics
- **WHEN** application queries diagnostics for runs with context assembler enabled
- **THEN** runtime returns assembler baseline fields that allow verification of prefix consistency and guard outcomes

#### Scenario: Consumer inspects CA2 stage diagnostics
- **WHEN** application queries diagnostics for runs with CA2 staged assembly enabled
- **THEN** runtime returns normalized stage and recap fields sufficient to reconstruct Stage1/Stage2 routing outcome

#### Scenario: Consumer inspects redaction behavior
- **WHEN** application queries diagnostics containing sensitive-key fields
- **THEN** returned payload contains masked values according to active redaction policy

#### Scenario: Consumer inspects Stage2 retrieval summary
- **WHEN** application queries diagnostics for runs that executed Stage2 retrieval
- **THEN** runtime returns normalized `stage2_hit_count`, `stage2_source`, `stage2_reason`, `stage2_reason_code`, `stage2_error_layer`, and `stage2_profile` fields

#### Scenario: Consumer inspects action timeline phase aggregates
- **WHEN** application queries diagnostics for runs with action timeline events
- **THEN** runtime returns phase-level aggregate metrics including counts and `latency_p95_ms`

### Requirement: Runtime diagnostics contract SHALL defer timeline aggregation fields in H1 with explicit TODO traceability
H1 MUST NOT introduce new timeline aggregation fields into persisted diagnostics run records. The repository documentation MUST record an explicit TODO for follow-up change(s) that converge timeline observability aggregation.

This constraint applies only to H1 scope. Starting from H1.5, timeline aggregate diagnostics fields are allowed and expected under backward-compatible field extension rules.

#### Scenario: Consumer queries diagnostics during H1
- **WHEN** application queries diagnostics APIs after timeline event rollout
- **THEN** existing diagnostics field schema remains stable without new timeline aggregate fields

#### Scenario: Maintainer reviews runtime docs after H1 rollout
- **WHEN** maintainer checks README and runtime diagnostics documentation
- **THEN** documentation contains explicit TODO notes for future timeline aggregation convergence

#### Scenario: Consumer queries diagnostics during H1.5+
- **WHEN** application queries diagnostics APIs after H1.5 observability convergence
- **THEN** run diagnostics include timeline aggregate fields while preserving backward compatibility for existing consumers
