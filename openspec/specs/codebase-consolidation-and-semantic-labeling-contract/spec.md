# codebase-consolidation-and-semantic-labeling-contract Specification

## Purpose
TBD - created by archiving change introduce-codebase-consolidation-and-semantic-labeling-contract-a63. Update Purpose after archive.
## Requirements
### Requirement: Active Repository Surface SHALL Use Semantic Context Assembler Naming
The repository MUST use semantic naming for Context Assembler stages in active code, tests, scripts, and documentation.

`ca1`, `ca2`, `ca3`, and `ca4` MUST NOT remain as active module or stage naming vocabulary outside approved legacy index scopes.
Detection MUST be case-insensitive (`ca*` and `CA*`) and applied to governed active-path matrix.

#### Scenario: Contributor edits active context assembly logic
- **WHEN** contributor changes active files under implementation, tests, scripts, or user-facing docs
- **THEN** Context Assembler naming MUST use semantic labels and MUST NOT introduce `ca1|ca2|ca3|ca4` stage terms

#### Scenario: Legacy naming appears only in index scope
- **WHEN** repository retains historical references for traceability
- **THEN** legacy names MUST be constrained to approved mapping/index locations and MUST NOT leak into active behavior descriptions

### Requirement: Active Repository Surface SHALL Replace Axx Wording with Semantic Labels
The repository MUST replace scattered `Axx` wording outside `openspec/**` with semantic labels.

`Axx` MAY exist only under `openspec/**` for historical proposal traceability.
Outside `openspec/**`, `Axx` MUST NOT appear in:
- file content,
- file path,
- file name.

#### Scenario: Active user-facing docs are updated
- **WHEN** contributor updates README or docs in active paths
- **THEN** semantic labels MUST be used and `A[0-9]{2,3}` wording MUST be absent from non-`openspec/**` narrative sections

#### Scenario: Historical proposal numbering remains traceable
- **WHEN** maintainers need to locate archived proposal lineage
- **THEN** mapping MUST be resolved through `openspec/**` index and archive artifacts rather than scattered inline Axx mentions elsewhere

#### Scenario: File path or file name contains Axx outside openspec
- **WHEN** contributor introduces new file or directory path containing `A[0-9]{2,3}` outside `openspec/**`
- **THEN** validation MUST fail and require semanticized path/file-name replacement

### Requirement: Consolidation Changes SHALL Preserve Runtime Semantic Compatibility
Naming and documentation consolidation MUST NOT alter runtime behavior contracts.

Public keys, diagnostics fields, script entrypoints, and fixture references touched by consolidation MUST provide deterministic compatibility bridge or alias during migration.
For public config and diagnostics contracts, migration MUST define semantic primary keys, legacy-compatible read path, and bounded deprecation window.

#### Scenario: Existing script or config entrypoint uses legacy alias
- **WHEN** contributor or CI invokes a legacy-compatible entrypoint during migration window
- **THEN** system MUST preserve existing behavior semantics and provide deterministic migration hint

#### Scenario: Consolidation path is rolled back
- **WHEN** maintainers revert consolidation changes due regression risk
- **THEN** compatibility alias and mapping table MUST allow deterministic rollback without behavior drift

#### Scenario: External consumer still uses legacy config or diagnostics keys
- **WHEN** consumer submits or parses legacy CA/Axx-era keys during migration window
- **THEN** runtime MUST keep behavior equivalent via compatibility read-path and provide deterministic migration guidance

### Requirement: Repository SHALL Maintain a Single Semantic Mapping Source
The repository MUST provide one canonical semantic mapping source that maps semantic names to legacy numbering/naming for historical traceability.

Code comments, docs, script help text, and test naming guidance MUST reference this canonical source instead of maintaining divergent local mappings.

#### Scenario: Contributor needs legacy-to-semantic mapping
- **WHEN** contributor searches for migration references
- **THEN** contributor MUST find canonical mapping in one documented source-of-truth location

#### Scenario: Duplicate local mapping is introduced
- **WHEN** a new local file introduces parallel legacy mapping definitions
- **THEN** consistency checks MUST fail and require convergence back to canonical mapping source

### Requirement: Consolidation Governance SHALL Block Legacy Naming Regression
Repository validation MUST include regression checks that block reintroduction of legacy Context Assembler and Axx wording into active paths.

Shell and PowerShell validation flows MUST enforce equivalent fail-fast semantics for this regression class.

#### Scenario: Legacy naming is reintroduced in active path
- **WHEN** validation detects `ca1|ca2|ca3|ca4` in governed active directories or detects `A[0-9]{2,3}` in non-`openspec/**` content/path/file-name
- **THEN** validation MUST exit non-zero and block merge

#### Scenario: No legacy naming regression detected
- **WHEN** governed active directories pass naming scan and mapping consistency checks
- **THEN** consolidation validation passes without blocking merge

### Requirement: Consolidation SHALL Remove Stale Temporary Assets from Active Surface
Repository MUST remove or archive stale temporary artifacts that are not source-of-truth deliverables, including offline scaffold bulk copies and accidental timestamp backup files.

#### Scenario: Repository contains offline scaffold bulk copies outside retained sample policy
- **WHEN** consolidation validation scans active examples and docs assets
- **THEN** non-retained bulk offline scaffolds MUST be removed or archived with index-based traceability

#### Scenario: Repository contains accidental timestamp backup source files
- **WHEN** validation detects files matching timestamp backup naming pattern in active source directories
- **THEN** validation MUST fail and require cleanup before merge

### Requirement: Repository SHALL Govern Single-File Code Size and Require Semantic Split for Oversized Files
For governed `*.go` files outside `openspec/**`, repository MUST enforce single-file line budget and require semantic split when a file exceeds hard threshold.

Other file types are out of this requirement scope in A63.

Governance MUST include:
- deterministic line-count rule definition,
- controlled exception list with owner/reason/expiry,
- prohibition on introducing new oversized files or increasing oversized-file debt,
- semantic-equivalence strong validation before merge.

#### Scenario: New or modified code file exceeds hard line threshold
- **WHEN** validation detects a governed `*.go` file above hard threshold and not covered by active exception
- **THEN** change MUST fail and require semantic split or threshold-compliant refactor

#### Scenario: Existing oversized file is modified without split
- **WHEN** a governed `*.go` file is already oversized and change increases line-count debt without approved exception policy
- **THEN** validation MUST fail and block merge

#### Scenario: Oversized file is split without semantic drift
- **WHEN** contributor extracts oversized `*.go` file into smaller semantic units
- **THEN** contract/replay outcomes MUST remain semantically equivalent

#### Scenario: Split passes only after strong semantic-equivalence checks
- **WHEN** a `*.go` oversized-file split change is validated
- **THEN** merge MUST be blocked unless Run/Stream parity, impacted contract suites, and replay idempotency checks are all green

