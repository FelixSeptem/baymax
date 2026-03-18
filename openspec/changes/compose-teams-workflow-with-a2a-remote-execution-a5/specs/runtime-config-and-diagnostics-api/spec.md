## ADDED Requirements

### Requirement: Runtime SHALL expose composed-orchestration config with deterministic precedence
Runtime configuration MUST expose composed-orchestration controls with precedence `env > file > default` for workflow-remote and teams-remote execution paths.

At minimum for this milestone, config MUST include:
- workflow remote-step enablement and defaults,
- teams remote-worker enablement and defaults,
- validation controls that prevent ambiguous or conflicting domain semantics.

#### Scenario: Startup resolves composed-orchestration config with env override
- **WHEN** composed-orchestration controls are defined in both file and environment variables
- **THEN** effective configuration resolves by `env > file > default` and invalid values fail fast

#### Scenario: Invalid composed-orchestration hot reload update
- **WHEN** watched config updates composed-orchestration fields to invalid values
- **THEN** runtime rejects update and keeps previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive composed-orchestration summary fields
Run diagnostics MUST expose additive summary fields for composed orchestration paths, including at minimum remote execution totals and failure markers, while preserving backward compatibility.

#### Scenario: Consumer inspects composed run summary
- **WHEN** application queries diagnostics for a run that mixes workflow/teams with A2A remote execution
- **THEN** diagnostics include additive composed summary fields without breaking existing consumers

### Requirement: Composed diagnostics SHALL remain replay-idempotent
Repeated ingestion of identical composed orchestration events MUST NOT inflate logical aggregate counters.

#### Scenario: Duplicate composed events are replayed
- **WHEN** composed orchestration event stream is replayed multiple times for one run
- **THEN** diagnostics aggregates remain stable after first logical ingestion
