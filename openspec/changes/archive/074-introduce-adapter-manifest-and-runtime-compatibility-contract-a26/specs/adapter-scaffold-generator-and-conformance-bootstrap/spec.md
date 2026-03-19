## ADDED Requirements

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
