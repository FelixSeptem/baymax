## ADDED Requirements

### Requirement: Runtime config SHALL expose skill embedding trigger scoring controls with deterministic precedence
Runtime configuration MUST expose embedding-enhanced skill trigger scoring controls under `skill.trigger_scoring.embedding.*` with precedence `env > file > default`.

At minimum, runtime MUST support:
- embedding scorer enablement and strategy activation controls,
- embedding timeout control,
- similarity metric selector for this milestone,
- lexical/embedding linear fusion weights.

For this milestone, configuration is managed through runtime JSON/YAML path only and MUST NOT require additional CLI parameters.

#### Scenario: Startup with skill embedding scoring overrides
- **WHEN** runtime starts with skill embedding scoring controls defined in both YAML and environment variables
- **THEN** effective skill embedding controls resolve by `env > file > default`

#### Scenario: Invalid skill embedding timeout
- **WHEN** runtime configuration sets non-positive skill embedding timeout
- **THEN** startup or hot reload fails fast with validation error

#### Scenario: Invalid skill embedding fusion weights
- **WHEN** runtime configuration sets invalid lexical/embedding fusion weights
- **THEN** startup or hot reload fails fast with validation error

### Requirement: Runtime diagnostics SHALL expose additive skill trigger scoring observability fields
Runtime diagnostics MUST expose additive skill-trigger observability fields sufficient for lexical-plus-embedding triage, including at minimum:
- active trigger scoring strategy,
- final trigger score,
- embedding score contribution (when available),
- embedding fallback reason (when fallback occurs).

These fields MUST be backward-compatible and MUST NOT redefine existing skill lifecycle diagnostics semantics.

#### Scenario: Consumer inspects successful lexical-plus-embedding trigger
- **WHEN** application queries skill diagnostics for runs using embedding-enhanced trigger scoring
- **THEN** diagnostics include strategy, final score, and embedding score contribution fields

#### Scenario: Consumer inspects embedding fallback
- **WHEN** application queries skill diagnostics for runs where embedding path falls back to lexical
- **THEN** diagnostics include normalized fallback reason while preserving existing lifecycle fields

### Requirement: Run and Stream SHALL preserve skill trigger scoring diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent skill trigger scoring diagnostics fields.

#### Scenario: Equivalent skill trigger diagnostics in Run and Stream
- **WHEN** equivalent requests execute under the same skill trigger scoring configuration and scorer behavior
- **THEN** diagnostics fields for strategy, final score, and fallback class are semantically equivalent across Run and Stream
