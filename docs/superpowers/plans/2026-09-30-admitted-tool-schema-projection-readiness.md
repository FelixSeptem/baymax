# Admitted Tool Schema Projection Readiness Implementation Plan

> **For agentic workers:** Execute this plan inline with the OpenSpec apply workflow and keep `tasks.md` checkboxes synchronized after each verified task.

**Goal:** Add a bounded offline `tool_schema_projection_readiness.v1` evaluator, replay contract, governance gates, and status documentation without changing runtime tool selection or provider projection.

**Architecture:** Extend the existing provider-neutral `tool/schemaaudit` owner with a separate readiness model/evaluator that consumes admitted canonical facts and finite sample windows. Keep replay/parity in `tool/diagnosticsreplay`, source-boundary checks in `tool/contributioncheck`, and gate scripts as thin deterministic wrappers over focused Go tests and fixtures.

**Tech Stack:** Go, JSON fixtures, PowerShell, POSIX shell, OpenSpec validation, existing schema-audit and diagnostics-replay packages.

---

### Task 1: Define the readiness contract and failing tests

**Files:**
- Create: `tool/schemaaudit/readiness.go`
- Create: `tool/schemaaudit/readiness_test.go`
- Modify: `tool/schemaaudit/replay.go`

- [ ] Write failing tests for version/bound/admission/privacy validation, canonical sample-window digest, stable/transient classification, admitted-subset opportunity, evidence axes, and three-way conclusion.
- [ ] Run `go test ./tool/schemaaudit -run Readiness` and confirm failures are due to missing readiness APIs.
- [ ] Implement minimal bounded types, enums, validation, window normalization, opportunity evaluation, and conclusion aggregation using existing schema facts; keep all output reference-only.
- [ ] Run the focused tests and confirm they pass; refactor only after green.

### Task 2: Add fixtures and diagnostics replay/parity

**Files:**
- Create: `tool/schemaaudit/testdata/tool_schema_projection_readiness.v1.json` and negative/default variants.
- Create: `tool/diagnosticsreplay/tool_schema_projection_readiness.go`
- Create: `tool/diagnosticsreplay/tool_schema_projection_readiness_test.go`

- [ ] Add failing replay tests for idempotence, unknown additive fields, historical defaults, metric/conclusion drift, privacy/overflow errors, and Run/Stream parity drift.
- [ ] Run the focused replay tests and confirm the expected missing-adapter failures.
- [ ] Implement replay and semantic parity comparison by delegating normalization to `tool/schemaaudit`, with stable reason codes and no raw payload retention.
- [ ] Run `go test ./tool/schemaaudit ./tool/diagnosticsreplay -run 'Readiness|ToolSchemaProjection'` and confirm all fixtures pass.

### Task 3: Add boundary checks and dedicated gates

**Files:**
- Create: `tool/contributioncheck/tool_schema_projection_readiness_boundary_test.go`
- Create: `scripts/check-tool-schema-projection-readiness-contract.sh`
- Create: `scripts/check-tool-schema-projection-readiness-contract.ps1`
- Modify: `scripts/check-quality-gate.sh`
- Modify: `scripts/check-quality-gate.ps1`

- [ ] Write failing boundary/gate tests for runtime selector/router wiring, admission bypass, remote discovery, provider SDK use, raw payload persistence, RuntimeRecorder writes, and both script paths.
- [ ] Run `go test ./tool/contributioncheck -run ToolSchemaProjectionReadiness` and verify the missing gate markers fail.
- [ ] Implement static boundary assertions and thin shell/PowerShell gate wrappers that run focused schemaaudit, diagnosticsreplay, and contributioncheck suites.
- [ ] Run both gate scripts and focused contribution tests; confirm fail-fast behavior and matching fixture coverage.

### Task 4: Synchronize contract documentation and status governance

**Files:**
- Modify: `docs/mainline-contract-test-index.md`
- Modify: `docs/development-roadmap.md`
- Modify: `README.md`

- [ ] Add the readiness owner, fixtures, replay/parity tests, and shell/PowerShell gate mapping to the contract index.
- [ ] Remove the archived scenario-simulation change from active/in-progress status, update the archive baseline, and add the readiness candidate with its trigger/non-goals.
- [ ] Run roadmap status, release status, docs consistency, and example-impact checks; fix only scoped status/mapping failures.

### Task 5: Verify, review, and finish the change

**Files:**
- Modify: `openspec/changes/establish-admitted-tool-schema-projection-readiness-contract/tasks.md`

- [ ] Run focused tests, `openspec validate --all`, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `scripts/check-quality-gate.ps1`, and `scripts/check-docs-consistency.ps1`; record environment-only limitations.
- [ ] Review `git diff --check`, architecture boundaries, additive/default compatibility, proposal-number naming, and the worktree diff for unrelated files.
- [ ] Mark each OpenSpec task complete only after its evidence is recorded, then commit the implementation in focused commits suitable for review and later archive.
