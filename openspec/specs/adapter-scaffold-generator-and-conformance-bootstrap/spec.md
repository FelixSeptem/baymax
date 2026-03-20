# adapter-scaffold-generator-and-conformance-bootstrap Specification

## Purpose
TBD - created by archiving change introduce-adapter-scaffold-generator-and-conformance-bootstrap-a23. Update Purpose after archive.
## Requirements
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

### Requirement: Adapter scaffold generator SHALL emit manifest template by default
Generated adapter scaffolds MUST include an adapter manifest template aligned with repository manifest schema.

The generated manifest template MUST include category-appropriate defaults for:
- `type`,
- `name`,
- `version`,
- `baymax_compat`,
- `capabilities.required`,
- `capabilities.optional`,
- `conformance_profile`.

#### Scenario: Contributor generates MCP scaffold
- **WHEN** contributor generates scaffold with `type=mcp`
- **THEN** generated files include MCP manifest template with schema-compliant defaults

#### Scenario: Contributor generates model or tool scaffold
- **WHEN** contributor generates scaffold with `type=model` or `type=tool`
- **THEN** generated files include corresponding manifest template and schema-compliant defaults

### Requirement: Scaffold manifest template SHALL align with conformance bootstrap profile
Generated scaffold manifest and generated conformance bootstrap MUST remain semantically aligned so bootstrap checks target the declared manifest profile.

#### Scenario: Maintainer audits generated scaffold alignment
- **WHEN** maintainer compares generated manifest template and bootstrap test skeleton
- **THEN** declared `conformance_profile` maps to matching bootstrap expectations without drift

### Requirement: Scaffold generator SHALL include capability negotiation and fallback test skeleton
Generated adapter scaffold MUST include minimal negotiation/fallback contract test skeletons aligned with repository taxonomy and strategy defaults.

The generated skeleton MUST cover:
- required capability missing fail-fast path,
- optional capability downgrade path,
- Run/Stream equivalence assertion for negotiation outcomes.

#### Scenario: Contributor generates scaffold and inspects tests
- **WHEN** contributor generates adapter scaffold
- **THEN** generated test skeleton includes negotiation and fallback contract cases with repository-default taxonomy markers

### Requirement: Scaffold defaults SHALL use fail_fast strategy and expose override hook
Generated scaffold configuration MUST default to `fail_fast` strategy and provide explicit request-level override hook for `best_effort`.

#### Scenario: Contributor uses default scaffold strategy
- **WHEN** contributor runs generated scaffold without strategy override
- **THEN** negotiation default behavior uses `fail_fast`

#### Scenario: Contributor uses generated override hook
- **WHEN** contributor applies request-level override to `best_effort`
- **THEN** generated scaffold path exercises downgrade behavior with deterministic reason output

