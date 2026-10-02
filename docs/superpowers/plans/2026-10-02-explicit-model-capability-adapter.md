# Explicit Model Capability Adapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let minimal local/test model clients explicitly opt into provider capability preflight and make missing discovery contracts actionable.

**Architecture:** Add a dependency-light `adapter/modelcapability` wrapper around `types.ModelClient`. The wrapper declares normalized provider/model/capability metadata and delegates model execution unchanged. Keep runner fallback semantics intact while adding bounded diagnostic fields for candidates that lack `ModelCapabilityDiscovery`; update the canonical model template and adapter docs to demonstrate the opt-in.

**Tech Stack:** Go, existing `core/types`, `core/runner`, OpenSpec contract files, PowerShell quality gates.

---

### Task 1: Lock the adapter contract with failing tests

**Files:**
- Create: `adapter/modelcapability/adapter_test.go`
- Create: `adapter/modelcapability/adapter.go` (after the test is red)

- [ ] Write tests for valid wrapping, duplicate capability normalization, nil model/provider validation, unknown capability rejection, discovery output, Generate delegation, Stream callback/error delegation, and compile-time interface assertions.
- [ ] Run `go test ./adapter/modelcapability -count=1`; confirm failure is due to missing `adapter/modelcapability` implementation rather than test setup.

### Task 2: Implement the explicit capability adapter

**Files:**
- Create: `adapter/modelcapability/adapter.go`

- [ ] Define `Config`, `Adapter`, `Wrap`, and `New`/validation APIs using existing `types.ModelCapability` values only.
- [ ] Normalize provider/model strings and capabilities, reject invalid inputs, sort/deduplicate capabilities, and build deterministic `ProviderCapabilities` with `Source: explicit_adapter`.
- [ ] Delegate `Generate` and `Stream` directly and assert `types.ModelClient` and `types.ModelCapabilityDiscovery` at compile time.
- [ ] Run `gofmt` and `go test ./adapter/modelcapability -count=1`; confirm all adapter tests pass.

### Task 3: Add runner regression tests first

**Files:**
- Modify: `core/runner/runner_test.go`

- [ ] Add a minimal model fixture implementing only `types.ModelClient` and a test that runs strict Stream preflight, asserting no model invocation and diagnostic details naming `ModelCapabilityDiscovery`.
- [ ] Add a test that wraps the same fixture with `modelcapability.Wrap` declaring `streaming` and asserts Stream completes.
- [ ] Run the focused tests and confirm they fail before runner diagnostics are changed.

### Task 4: Improve runner preflight diagnostics

**Files:**
- Modify: `core/runner/runner.go`

- [ ] Track whether candidates were skipped because the discovery interface is unavailable separately from discovered capability mismatches.
- [ ] Preserve the existing classified model error and `capability_unsupported` details for real capability gaps; use `capability_discovery_unavailable`, `missing_interfaces`, and an actionable message when appropriate.
- [ ] Run `go test ./core/runner -run 'Capability|Stream' -count=1` and the full `go test ./core/runner -count=1`.

### Task 5: Cover integration parity

**Files:**
- Create or modify: `integration/model_capability_adapter_contract_test.go`

- [ ] Verify equivalent Run and Stream requests use the same explicit capability declaration and that a discovery-less candidate is never invoked under strict preflight.
- [ ] Run `go test ./integration -run 'Capability|Stream' -count=1`.

### Task 6: Update template and documentation

**Files:**
- Modify: `examples/templates/model-adapter-template/main.go`
- Modify: `docs/external-adapter-template-index.md`
- Modify: `adapter/README.md`

- [ ] Wrap the template model with `modelcapability.Config{Provider: "template", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming}}` and document why this explicit declaration is required for Stream.
- [ ] Add usage and error guidance to adapter docs, keeping provider SDK ownership in `model/<provider>`.
- [ ] Run `go run ./examples/templates/model-adapter-template` and `pwsh -File scripts/check-docs-consistency.ps1`.

### Task 7: Validate OpenSpec and repository gates

**Files:**
- Modify: `openspec/changes/add-explicit-model-capability-adapter/tasks.md`

- [ ] Run `openspec validate --all` and mark completed OpenSpec tasks with code/test/doc evidence.
- [ ] Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; fix regressions and record any blocked command with its reason.
