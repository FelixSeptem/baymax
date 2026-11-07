# Role-Scoped Runtime Ports and Owner Evidence Index Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a validated static owner-evidence index for enrolled public semantics without changing runtime ports or runtime behavior.

**Architecture:** A versioned JSON artifact is canonical; a Go validator in `tool/contributioncheck` resolves every evidence path and validates owner/doc-source-map/gate relationships. Paired platform scripts invoke its focused test and are invoked from both existing quality gates. Public docs link to the single canonical artifact and state its authority limits.

**Tech Stack:** Go standard library (`encoding/json`, `os`, `path/filepath`), repository Markdown, PowerShell and POSIX shell gate scripts, OpenSpec.

---

## File structure

- Create `docs/owner-evidence-index.v1.json`: versioned canonical entries.
- Create `docs/owner-evidence-index.md`: human entry point and constraints.
- Create `tool/contributioncheck/owner_evidence_index.go`: JSON types and pure validator.
- Create `tool/contributioncheck/owner_evidence_index_test.go`: positive, malformed and repository-integrated coverage.
- Create `scripts/check-owner-evidence-index-contract.ps1` and `.sh`: paired focused gates.
- Modify `scripts/check-quality-gate.ps1` and `.sh`: run the dedicated gate.
- Modify `README.md`, `docs/README.md`, `docs/documentation-ownership.md`, `docs/mainline-contract-test-index.md`, `docs/runtime-module-boundaries.md`, and `docs/development-roadmap.md`: public navigation, exact owners, evidence limits and current status.
- Create OpenSpec proposal/design/tasks/spec delta/verification under `openspec/changes/role-scoped-runtime-ports-and-owner-index`.

### Task 1: Create the OpenSpec and canonical design inputs

**Files:**
- Create: `openspec/changes/role-scoped-runtime-ports-and-owner-index/{proposal.md,design.md,tasks.md,specs/role-scoped-runtime-ports-and-owner-index/spec.md}`
- Create: `docs/owner-evidence-index.v1.json`

- [ ] Write proposal, design, tasks and delta spec with one-owner-per-unsplit-semantic, explicit replay exemption rules, quality-gate wiring, no-port-refactor stop line, Example Impact Assessment and Public Semantic Documentation Commitment.
- [ ] Run `openspec validate role-scoped-runtime-ports-and-owner-index --strict` and correct artifact schema errors.
- [ ] Create a v1 JSON document with three actual historical semantics and only repository-relative evidence paths.

### Task 2: Specify validation behavior with red tests

**Files:**
- Create: `tool/contributioncheck/owner_evidence_index_test.go`

- [ ] Write `TestValidateOwnerEvidenceIndexAcceptsRepositoryIndex` that parses the checked-in JSON and requires no issues.
- [ ] Write table-driven tests with inline JSON/filesystem fixtures for duplicate IDs, absent/ambiguous/multiple owners, missing paths, absent gate, unwired gate, fixture replay with a missing fixture, replay exemption without a reason, documentation-owner contradiction, and unsupported version.
- [ ] Run `go test ./tool/contributioncheck -run TestValidateOwnerEvidenceIndex -count=1`; confirm it fails because `ValidateOwnerEvidenceIndex` is undefined.

### Task 3: Implement the minimal pure validator

**Files:**
- Create: `tool/contributioncheck/owner_evidence_index.go`

- [ ] Define `OwnerEvidenceIndex`, `OwnerEvidenceSemantic`, `OwnerEvidenceReplay`, `OwnerEvidenceIssue`, and `ValidateOwnerEvidenceIndex(root, index, sourceMap, qualityPowerShell, qualityShell string) []OwnerEvidenceIssue`.
- [ ] Decode exactly version `owner-evidence-index.v1`; require stable IDs, exactly one semantic owner, existing main spec/test/fixture/gate/documentation paths, concrete rollback/non-goal values and the exact source-map owner cell.
- [ ] Require each gate basename in both quality scripts; permit replay only as `fixture` with a path or `not_applicable` with a non-whitespace reason.
- [ ] Run the focused test command from Task 2 and confirm it passes.

### Task 4: Add dedicated paired gate and quality wiring

**Files:**
- Create: `scripts/check-owner-evidence-index-contract.ps1`
- Create: `scripts/check-owner-evidence-index-contract.sh`
- Modify: `scripts/check-quality-gate.ps1`
- Modify: `scripts/check-quality-gate.sh`
- Modify: `tool/contributioncheck/owner_evidence_index_test.go`

- [ ] Write failing tests that require strict shell execution, PowerShell `native-strict` usage, focused validator test invocation, and both quality scripts to name the gate.
- [ ] Run the focused test and confirm those assertions fail before scripts/wiring exist.
- [ ] Implement the scripts and add an explicit blocking quality-gate step beside other contract gates.
- [ ] Run `pwsh -File scripts/check-owner-evidence-index-contract.ps1` and `go test ./tool/contributioncheck -run TestOwnerEvidenceIndex -count=1` successfully.

### Task 5: Document exact evidence ownership and current proposal status

**Files:**
- Create: `docs/owner-evidence-index.md`
- Modify: `README.md`
- Modify: `docs/README.md`
- Modify: `docs/documentation-ownership.md`
- Modify: `docs/mainline-contract-test-index.md`
- Modify: `docs/runtime-module-boundaries.md`
- Modify: `docs/development-roadmap.md`

- [ ] Replace broad ownership wording for the enrolled semantics with exact owners compatible with the canonical index.
- [ ] Link the public navigation and contract index to the canonical JSON/page, state replay is evidence only, and state no global registry/authorization/execution/recovery authority is created.
- [ ] Mark E as in progress under roadmap current status while its OpenSpec directory is active.
- [ ] Run `pwsh -File scripts/check-docs-consistency.ps1` and `pwsh -File scripts/check-openspec-documentation-impact.ps1`.

### Task 6: Complete evidence, verification and archival

**Files:**
- Modify: `openspec/changes/role-scoped-runtime-ports-and-owner-index/tasks.md`
- Create: `openspec/changes/role-scoped-runtime-ports-and-owner-index/verification.md`

- [ ] Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `openspec validate --all`, and `git diff --check`; record fresh commands/results.
- [ ] Independently review the diff against the no-runtime-port/no-registry/non-authorization boundaries; update tasks only after each item has evidence.
- [ ] Archive only with `pwsh -File scripts/openspec-archive-seq.ps1 -ChangeName "role-scoped-runtime-ports-and-owner-index"`, update archive-linked roadmap/index status, rerun documentation/OpenSpec checks, then commit, integrate latest `master`, push and remove only this merged worktree/branch.

## Self-review

Coverage: Tasks 1–3 implement all model invariants, Task 4 makes the validation non-optional, Task 5 makes the public ownership map precise, and Task 6 supplies delivery evidence and lifecycle work. The plan deliberately contains no runtime-interface task because the approved scope found no two-caller overreach proof.

Placeholder scan: no incomplete implementation placeholders remain. Names, JSON fields and validator signature are consistent between Tasks 2–4.
