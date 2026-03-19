## ADDED Requirements

### Requirement: Repository SHALL provide adapter scaffold generator for MCP, Model, and Tool categories
The repository MUST provide a scaffold generator command for external adapters and MUST support `mcp`, `model`, and `tool` categories in one consistent interface.

#### Scenario: Contributor generates MCP scaffold
- **WHEN** contributor runs scaffold generation with type `mcp` and a valid adapter name
- **THEN** generator creates a valid MCP adapter scaffold in the expected structure

#### Scenario: Contributor generates Model and Tool scaffold
- **WHEN** contributor runs scaffold generation with type `model` or `tool`
- **THEN** generator creates corresponding category-specific scaffold outputs using the same command contract

### Requirement: Scaffold output SHALL be deterministic and offline by default
Given identical inputs, scaffold generation MUST produce deterministic file layout and content, and MUST NOT require network connectivity or external credentials.

#### Scenario: Same inputs produce identical output
- **WHEN** contributor executes scaffold generation twice with same type, name, and output path
- **THEN** generated file set and contents are identical

#### Scenario: Disconnected environment generation
- **WHEN** contributor runs scaffold generation in an environment without network access
- **THEN** scaffold generation completes without external service dependency

### Requirement: Scaffold generator SHALL provide default output path and conflict fail-fast behavior
The default output path MUST be `examples/adapters/<type>-<name>`.
If target files already exist and `--force` is not set, generation MUST fail fast before writing new files.
When `--force` is explicitly set, generation MAY overwrite existing files in the target path.

#### Scenario: Default output path is used
- **WHEN** contributor omits explicit output flag
- **THEN** generator writes scaffold to `examples/adapters/<type>-<name>`

#### Scenario: Existing file conflict without force
- **WHEN** target path contains one or more files that would be generated and `--force` is not provided
- **THEN** generator exits non-zero with conflict details and performs no partial write

#### Scenario: Existing file conflict with force
- **WHEN** target path contains existing files and contributor enables `--force`
- **THEN** generator overwrites planned files and completes with success

### Requirement: Generated scaffold SHALL include conformance bootstrap aligned with adapter conformance harness
Generated adapter scaffold MUST include a conformance bootstrap entry that maps to repository adapter conformance harness execution path.

This bootstrap MUST be enabled by default so generated adapters can enter conformance validation without manual test skeleton authoring.

#### Scenario: Contributor runs conformance bootstrap from generated scaffold
- **WHEN** contributor follows generated scaffold instructions for conformance validation
- **THEN** generated bootstrap path invokes adapter conformance harness using repository-standard contract flow

#### Scenario: Maintainer audits scaffold-contract alignment
- **WHEN** maintainer checks generated scaffold files and conformance harness expectations
- **THEN** required bootstrap test entry and mapping hints are present and semantically aligned

### Requirement: Generated scaffold SHALL include minimum executable onboarding artifacts
Each generated scaffold MUST include at least:
- adapter implementation skeleton,
- local README onboarding notes,
- minimum unit-test skeleton,
- minimum conformance bootstrap test skeleton.

#### Scenario: Integrator inspects generated scaffold
- **WHEN** integrator opens generated adapter directory
- **THEN** required onboarding artifacts exist and are ready for incremental implementation
