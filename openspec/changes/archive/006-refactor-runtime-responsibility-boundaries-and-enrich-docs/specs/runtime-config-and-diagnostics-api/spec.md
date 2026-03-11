## MODIFIED Requirements

### Requirement: Runtime SHALL load configuration with deterministic precedence
The runtime MUST load configuration from defaults, YAML file, and environment variables with precedence `env > file > default`. This configuration capability MUST be owned by a global runtime module and be consumable by MCP, runner, local tool, skill loader, and observability components through stable interfaces.

#### Scenario: Startup with file and environment overrides
- **WHEN** runtime starts with a YAML config file and overlapping environment variables
- **THEN** effective configuration uses environment values first, then file values, then defaults for unset keys

#### Scenario: Startup without config file
- **WHEN** runtime starts without a config file
- **THEN** runtime uses default values and applicable environment overrides

## ADDED Requirements

### Requirement: Runtime config API migration SHALL preserve behavior compatibility
When runtime config implementation paths are refactored, the system MUST preserve existing precedence, validation, and hot-reload semantics for callers migrating from previous package locations.

#### Scenario: Caller migrates from legacy package path
- **WHEN** caller switches from legacy runtime config package path to the new global runtime package path
- **THEN** resolved effective config and diagnostics behavior remain semantically equivalent under the same inputs

### Requirement: Runtime diagnostics API migration SHALL preserve behavior compatibility
When diagnostics API implementation paths are refactored, the system MUST preserve existing normalized fields, bounded history semantics, and sanitized config output behavior for callers migrating from previous package locations.

#### Scenario: Caller migrates diagnostics API package path
- **WHEN** caller switches diagnostics API imports from legacy package path to new global runtime package path
- **THEN** recent runs/calls/reloads outputs remain semantically equivalent for the same recorded inputs

### Requirement: Runtime diagnostics API SHALL cover skill lifecycle semantics
The runtime diagnostics API MUST support normalized skill lifecycle diagnostics, including discovery, trigger matching, compile outcomes, and failure classification, while preserving shared correlation fields.

#### Scenario: Skill loader emits diagnostics
- **WHEN** skill discovery or compile pipeline runs
- **THEN** diagnostics API returns normalized skill records with shared correlation fields and skill-specific payload metadata

### Requirement: Runtime documentation SHALL publish config field index and migration mapping
The repository MUST publish a configuration field index and package migration mapping, including old-to-new API references and deprecation notes.

#### Scenario: Maintainer updates runtime config docs
- **WHEN** config fields or package paths change
- **THEN** docs include synchronized schema reference and migration table for affected APIs
